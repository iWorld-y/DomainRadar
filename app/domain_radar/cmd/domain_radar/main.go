package main

import (
	"context"
	"embed"
	"encoding/json"
	"flag"
	"fmt"
	"html/template"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
	"github.com/go-shiori/go-readability"
	"golang.org/x/time/rate"

	"github.com/iWorld-y/domain_radar/app/domain_radar/internal/config"
	"github.com/iWorld-y/domain_radar/app/domain_radar/internal/logger"
	dm "github.com/iWorld-y/domain_radar/app/domain_radar/internal/model"
	"github.com/iWorld-y/domain_radar/app/domain_radar/internal/storage"
	"github.com/iWorld-y/domain_radar/app/domain_radar/internal/tavily"
)

//go:embed resource/index.html
var templateFS embed.FS

// HTMLData 用于模板渲染的数据
type HTMLData struct {
	Date          string
	Count         int // 总阅读文章数
	DomainReports []dm.DomainReport
	DeepAnalysis  *dm.DeepAnalysisResult
}

func main() {
	// 0. 定义命令行参数
	configPath := flag.String("config", "configs/config.yaml", "配置文件路径")
	flag.Parse()

	// 1. 加载配置
	cfg, err := config.LoadConfig(*configPath)
	if err != nil {
		log.Fatalf("无法加载配置文件 [%s]: %v", *configPath, err)
	}

	// 验证配置
	if cfg.TavilyAPIKey == "" {
		log.Fatal("配置错误: 未设置 tavily_api_key")
	}
	if len(cfg.Domains) == 0 {
		log.Fatal("配置错误: 未设置感兴趣的领域 (domains)")
	}

	// 2. 初始化日志
	if err = logger.InitLogger(cfg.Log.Level, cfg.Log.File); err != nil {
		log.Fatalf("无法初始化日志: %v", err)
	}
	logger.Log.Info("启动领域雷达...")

	ctx := context.Background()

	// 初始化数据库连接
	// 如果配置了数据库信息，则尝试连接
	var store *storage.Storage
	if cfg.DB.Host != "" {
		s, err := storage.NewStorage(cfg.DB)
		if err != nil {
			logger.Log.Errorf("无法连接数据库: %v. 将仅生成 HTML 文件。", err)
		} else {
			store = s
			defer store.Close()
			logger.Log.Info("已成功连接到数据库")
		}
	} else {
		logger.Log.Info("未配置数据库信息，跳过数据库连接")
	}

	// 3. 初始化 LLM
	chatModel, err := openai.NewChatModel(ctx, &openai.ChatModelConfig{
		BaseURL: cfg.LLM.BaseURL,
		APIKey:  cfg.LLM.APIKey,
		Model:   cfg.LLM.Model,
	})
	if err != nil {
		logger.Log.Fatalf("LLM 初始化失败: %v", err)
	}

	// 4. 初始化限流器
	limit := rate.Limit(float64(cfg.Concurrency.RPM) / 60.0)
	burst := cfg.Concurrency.QPS
	limiter := rate.NewLimiter(limit, burst)
	logger.Log.Infof("限流器已配置: Limit=%.2f req/s, Burst=%d", limit, burst)

	var domainReports []dm.DomainReport
	var mu sync.Mutex
	var wg sync.WaitGroup

	// 用于统计总文章数
	var totalArticles int

	// 5. 初始化 Tavily 客户端
	tavilyClient := tavily.NewClient(cfg.TavilyAPIKey)

	// 计算日期范围 (最近 3 天)
	now := time.Now()
	endDate := now.Format(time.DateOnly)
	startDate := now.AddDate(0, 0, -3).Format(time.DateOnly)

	// 6. 遍历领域进行搜索和处理
	// 这是一个串行过程还是并行？为了避免并发过高触发 LLM/Tavily 限制，
	// 我们可以对 Domain 进行并行，但控制并发数。这里简单起见，使用 waitgroup。

	for _, domain := range cfg.Domains {
		wg.Add(1)
		go func(domain string) {
			defer wg.Done()
			logger.Log.Infof("正在处理领域: %s", domain)

			// 6.1 搜索文章 (请求更多结果以确保有足够的高质量文章)
			req := tavily.SearchRequest{
				Query:             domain,
				Topic:             "news",
				MaxResults:        10, // 增加抓取数量，确保至少有 5 篇可用
				StartDate:         startDate,
				EndDate:           endDate,
				IncludeRawContent: false,
			}

			resp, err := tavilyClient.Search(req)
			if err != nil {
				logger.Log.Errorf("搜索领域失败 [%s]: %v", domain, err)
				return
			}

			// 6.2 抓取正文
			var validArticles []dm.Article
			for _, item := range resp.Results {
				// 简单的去重或过滤逻辑可以在这里添加
				content := item.Content

				// 尝试获取正文，如果摘要太短
				if len(content) < 500 {
					fetched, err := fetchAndCleanContent(item.URL)
					if err == nil && len(fetched) > len(content) {
						content = fetched
					}
				}

				// 截断过长内容
				if len(content) > 5000 {
					content = content[:5000]
				}

				if len(content) > 100 { // 只有内容足够才算有效
					validArticles = append(validArticles, dm.Article{
						Title:   item.Title,
						Link:    item.URL,
						Source:  domain,
						PubDate: item.PublishedDate,
						Content: content,
					})
				}

				if len(validArticles) >= 6 { // 只要前 6 篇优质文章即可
					break
				}
			}

			if len(validArticles) < 1 {
				logger.Log.Warnf("领域 [%s] 未找到足够的有效文章", domain)
				return
			}

			// 6.3 生成领域报告
			report, err := generateDomainReport(ctx, chatModel, domain, validArticles, limiter)
			if err != nil {
				logger.Log.Errorf("生成领域报告失败 [%s]: %v", domain, err)
				return
			}
			report.Articles = validArticles // 关联原文引用

			// 保存到数据库
			if store != nil {
				if err := store.SaveDomainReport(report); err != nil {
					logger.Log.Errorf("保存领域报告失败 [%s]: %v", domain, err)
				} else {
					logger.Log.Infof("领域报告已保存到数据库 [%s]", domain)
				}
			}

			mu.Lock()
			domainReports = append(domainReports, *report)
			totalArticles += len(validArticles)
			mu.Unlock()
			logger.Log.Infof("领域 [%s] 处理完成 (Score: %d)", domain, report.Score)
		}(domain)
	}

	wg.Wait()

	// 7. 排序：按领域评分从高到低
	sort.Slice(domainReports, func(i, j int) bool {
		return domainReports[i].Score > domainReports[j].Score
	})

	// 8. 深度解读
	var deepAnalysis *dm.DeepAnalysisResult
	if cfg.UserPersona != "" && len(domainReports) > 0 {
		logger.Log.Info("正在生成全局深度解读报告...")

		// 构造输入：使用各领域的 Summary 和 Trends
		var sb strings.Builder
		for _, report := range domainReports {
			fmt.Fprintf(&sb, "## 领域：%s (评分: %d)\n", report.DomainName, report.Score)
			fmt.Fprintf(&sb, "### 综述\n%s\n", report.Overview)
			fmt.Fprintf(&sb, "### 趋势\n%s\n", report.Trends)
			fmt.Fprintf(&sb, "### 关键事件\n- %s\n\n", strings.Join(report.KeyEvents, "\n- "))
		}

		analysis, err := deepInterpretReport(ctx, chatModel, sb.String(), cfg.UserPersona, limiter)
		if err != nil {
			logger.Log.Errorf("全局深度解读失败: %v", err)
		} else {
			deepAnalysis = analysis
			logger.Log.Info("全局深度解读报告生成完成")

			// 保存到数据库
			if store != nil {
				if err := store.SaveDeepAnalysis(deepAnalysis); err != nil {
					logger.Log.Errorf("保存深度解读失败: %v", err)
				} else {
					logger.Log.Info("深度解读报告已保存到数据库")
				}
			}
		}
	}

	// 9. 生成 HTML
	data := HTMLData{
		Date:          time.Now().Format("2006-01-02"),
		Count:         totalArticles,
		DomainReports: domainReports,
		DeepAnalysis:  deepAnalysis,
	}

	if err := generateHTML(data); err != nil {
		logger.Log.Fatalf("生成 HTML 失败: %v", err)
	}

	logger.Log.Info("✅ 领域雷达早报生成完毕")
}

// fetchAndCleanContent 抓取 URL 并提取核心文本
func fetchAndCleanContent(url string) (string, error) {
	article, err := readability.FromURL(url, 30*time.Second)
	if err != nil {
		return "", err
	}
	return article.TextContent, nil
}

// generateDomainReport 生成单个领域的总结报告
func generateDomainReport(ctx context.Context, cm model.ChatModel, domain string, articles []dm.Article, limiter *rate.Limiter) (*dm.DomainReport, error) {
	// 构造 Prompt
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("以下是关于领域【%s】的一组新闻文章，请阅读并总结：\n\n", domain))
	for i, art := range articles {
		sb.WriteString(fmt.Sprintf("文章 %d:\n标题: %s\n内容摘要: %s\n\n", i+1, art.Title, art.Content))
	}

	prompt := `你是一个资深行业分析师。请根据提供的文章内容，撰写一份该领域的深度总结报告。
请务必严格按照以下 JSON 格式返回，不要包含任何 markdown 标记：
{
	"overview": "领域综述（Markdown格式，200字左右），总结当前领域的核心动态、热点话题。",
	"key_events": ["关键事件1", "关键事件2", "关键事件3"],
	"trends": "趋势分析（Markdown格式，100-200字），基于新闻分析未来的技术或市场走向。",
	"score": 8
}
评分说明：score 为 1-10 的整数，代表该领域今日的重要程度和关注价值。`

	// 调用 LLM (带重试机制)
	maxRetries := 3
	baseDelay := 2 * time.Second
	var lastErr error

	for i := 0; i <= maxRetries; i++ {
		if err := limiter.Wait(ctx); err != nil {
			return nil, err
		}

		messages := []*schema.Message{
			{Role: schema.System, Content: "你是一个 JSON 生成器。请只输出 JSON 字符串。"},
			{Role: schema.User, Content: sb.String() + "\n\n" + prompt},
		}

		resp, err := cm.Generate(ctx, messages)
		if err != nil {
			if strings.Contains(err.Error(), "429") || strings.Contains(strings.ToLower(err.Error()), "too many requests") {
				lastErr = err
				if i < maxRetries {
					time.Sleep(baseDelay * time.Duration(1<<i))
					continue
				}
			}
			return nil, err
		}

		cleanContent := strings.TrimSpace(resp.Content)
		cleanContent = strings.TrimPrefix(cleanContent, "```json")
		cleanContent = strings.TrimPrefix(cleanContent, "```")
		cleanContent = strings.TrimSuffix(cleanContent, "```")

		var report dm.DomainReport
		if err := json.Unmarshal([]byte(cleanContent), &report); err != nil {
			lastErr = err
			if i < maxRetries {
				continue
			}
			return nil, fmt.Errorf("json unmarshal: %w", err)
		}

		report.DomainName = domain
		return &report, nil
	}
	return nil, lastErr
}

// deepInterpretReport 全局深度解读报告
func deepInterpretReport(ctx context.Context, cm model.ChatModel, content string, userPersona string, limiter *rate.Limiter) (*dm.DeepAnalysisResult, error) {
	// 复用之前的逻辑，只是 Prompt 略微调整以适应输入变化
	promptTpl := `Role: 资深技术顾问与个人发展战略专家
Context
用户画像：%s
输入数据：这是一份多领域的每日新闻总结报告。
核心诉求：请跨领域交叉分析，识别宏观趋势，并为用户提供战略建议。

Instructions
请严格按照 JSON 格式输出：
{
    "macro_trends": "Markdown格式的核心趋势洞察...",
    "opportunities": "Markdown格式的机遇挖掘...",
    "risks": "Markdown格式的风险预警...",
    "action_guides": ["行动建议1", "行动建议2", "行动建议3"]
}

输入的新闻总结数据：
%s`

	// ... (代码结构与之前类似，略作简化以适应单文件)
	maxRetries := 3
	baseDelay := 2 * time.Second
	var lastErr error

	for i := 0; i <= maxRetries; i++ {
		if err := limiter.Wait(ctx); err != nil {
			return nil, err
		}

		messages := []*schema.Message{
			{Role: schema.System, Content: "你是一个 JSON 生成器。"},
			{Role: schema.User, Content: fmt.Sprintf(promptTpl, userPersona, content)},
		}

		resp, err := cm.Generate(ctx, messages)
		if err != nil {
			// 简单的错误处理逻辑
			if strings.Contains(err.Error(), "429") {
				time.Sleep(baseDelay * time.Duration(1<<i))
				continue
			}
			return nil, err
		}

		cleanContent := strings.TrimSpace(resp.Content)
		cleanContent = strings.TrimPrefix(cleanContent, "```json")
		cleanContent = strings.TrimPrefix(cleanContent, "```")
		cleanContent = strings.TrimSuffix(cleanContent, "```")

		var result dm.DeepAnalysisResult
		if err := json.Unmarshal([]byte(cleanContent), &result); err != nil {
			lastErr = err
			continue
		}
		return &result, nil
	}
	return nil, fmt.Errorf("failed after retries: %v", lastErr)
}

// generateHTML 渲染模板
func generateHTML(data HTMLData) error {
	t, err := template.ParseFS(templateFS, "resource/index.html")
	if err != nil {
		return fmt.Errorf("解析模板失败: %w", err)
	}

	outputPath := "index.html"
	// 如果在根目录运行且 output 目录存在，则输出到 output 目录
	if info, err := os.Stat("output"); err == nil && info.IsDir() {
		outputPath = filepath.Join("output", "index.html")
	}

	// 确保目标目录存在
	dir := filepath.Dir(outputPath)
	if dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("创建目录失败 [%s]: %w", dir, err)
		}
	}

	f, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("创建文件失败 [%s]: %w", outputPath, err)
	}
	defer f.Close()

	if err := t.Execute(f, data); err != nil {
		return fmt.Errorf("渲染模板失败: %w", err)
	}

	logger.Log.Infof("报告已保存至: %s", outputPath)
	return nil
}
