package engine

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/bytedance/gg/gson"
	"github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
	"github.com/go-shiori/go-readability"
	"golang.org/x/time/rate"

	"github.com/iWorld-y/domain_radar/app/domain_radar/pkg/config"
	"github.com/iWorld-y/domain_radar/app/domain_radar/pkg/logger"
	dm "github.com/iWorld-y/domain_radar/app/domain_radar/pkg/model"
	"github.com/iWorld-y/domain_radar/app/domain_radar/pkg/search"
	"github.com/iWorld-y/domain_radar/app/domain_radar/pkg/search/factory"
	"github.com/iWorld-y/domain_radar/app/domain_radar/pkg/storage"
)

// Engine 核心处理引擎
type Engine struct {
	cfg       *config.Config
	store     *storage.Storage
	chatModel model.ChatModel
	searcher  search.Searcher
	limiter   *rate.Limiter
}

// NewEngine 创建引擎实例
func NewEngine(cfg *config.Config, store *storage.Storage) (*Engine, error) {
	ctx := context.Background()

	// 初始化 LLM
	chatModel, err := openai.NewChatModel(ctx, &openai.ChatModelConfig{
		BaseURL: cfg.LLM.BaseURL,
		APIKey:  cfg.LLM.APIKey,
		Model:   cfg.LLM.Model,
	})
	if err != nil {
		return nil, fmt.Errorf("LLM 初始化失败: %w", err)
	}

	// 初始化限流器
	limit := rate.Limit(float64(cfg.Concurrency.RPM) / 60.0)
	burst := cfg.Concurrency.QPS
	limiter := rate.NewLimiter(limit, burst)

	// 初始化搜索客户端
	searcher, err := factory.NewSearcher(cfg)
	if err != nil {
		return nil, fmt.Errorf("搜索客户端初始化失败: %w", err)
	}

	return &Engine{
		cfg:       cfg,
		store:     store,
		chatModel: chatModel,
		searcher:  searcher,
		limiter:   limiter,
	}, nil
}

// RunOptions 运行选项
type RunOptions struct {
	UserID           int
	Domains          []string
	Persona          string
	ProgressCallback func(status string, progress int)
}

// Run 执行一次报告生成任务
func (e *Engine) Run(ctx context.Context, opts RunOptions) error {
	logger.Log.Infof("开始为用户 [%d] 生成报告，包含 %d 个领域", opts.UserID, len(opts.Domains))
	if opts.ProgressCallback != nil {
		opts.ProgressCallback("starting", 0)
	}

	if len(opts.Domains) == 0 {
		return fmt.Errorf("no domains provided")
	}

	// 创建本次运行记录
	var runID int
	if e.store != nil {
		rid, err := e.store.CreateRun()
		if err != nil {
			logger.Log.Errorf("无法创建运行记录: %v", err)
		} else {
			runID = rid
		}
	}

	var domainReports []dm.DomainReport
	var mu sync.Mutex
	var wg sync.WaitGroup

	now := time.Now()
	endDate := now.Format(time.DateOnly)
	startDate := now.AddDate(0, 0, -3).Format(time.DateOnly)

	totalDomains := len(opts.Domains)
	completedDomains := 0

	for _, domain := range opts.Domains {
		wg.Add(1)
		go func(domain string) {
			defer wg.Done()

			// 1. 搜索
			req := &search.Request{
				Query:             domain,
				Topic:             "news",
				MaxResults:        20,
				StartDate:         startDate,
				EndDate:           endDate,
				IncludeRawContent: false,
			}

			resp, err := e.searcher.Search(ctx, req)
			if err != nil {
				logger.Log.Errorf("搜索领域失败 [%s]: %v", domain, err)
				return
			}
			logger.Log.Debugf("搜索领域 [%s] 成功: %s", domain, gson.ToString(resp))

			// 2. 抓取正文
			var validArticles []dm.Article
			for _, item := range resp.Results {
				content := item.Content
				if len(content) < 500 {
					fetched, err := fetchAndCleanContent(item.URL)
					if err == nil && len(fetched) > len(content) {
						content = fetched
					}
				}
				if len(content) > 5000 {
					content = content[:5000]
				}
				if len(content) > 100 {
					validArticles = append(validArticles, dm.Article{
						Title:   item.Title,
						Link:    item.URL,
						Source:  domain,
						PubDate: item.PublishedDate,
						Content: content,
					})
				}
				if len(validArticles) >= 6 {
					break
				}
			}

			if len(validArticles) < 1 {
				logger.Log.Warnf("领域 [%s] 未找到足够的有效文章", domain)
				return
			}

			// 3. 生成领域报告
			report, err := generateDomainReport(ctx, e.chatModel, domain, validArticles, e.limiter)
			if err != nil {
				logger.Log.Errorf("生成领域报告失败 [%s]: %v", domain, err)
				return
			}
			report.Articles = validArticles

			// 保存到数据库
			if e.store != nil && runID > 0 {
				if err := e.store.SaveDomainReport(runID, report); err != nil {
					logger.Log.Errorf("保存领域报告失败 [%s]: %v", domain, err)
				}
			}

			mu.Lock()
			domainReports = append(domainReports, *report)
			completedDomains++
			progress := 10 + int(float64(completedDomains)/float64(totalDomains)*70) // 10% -> 80%
			if opts.ProgressCallback != nil {
				opts.ProgressCallback(fmt.Sprintf("processed domain: %s", domain), progress)
			}
			mu.Unlock()
		}(domain)
	}

	wg.Wait()

	if len(domainReports) == 0 {
		return fmt.Errorf("no domain reports generated")
	}

	// 排序
	sort.Slice(domainReports, func(i, j int) bool {
		return domainReports[i].Score > domainReports[j].Score
	})

	// 4. 深度解读
	if opts.ProgressCallback != nil {
		opts.ProgressCallback("generating deep analysis", 85)
	}

	if opts.Persona != "" {
		var sb strings.Builder
		for _, report := range domainReports {
			fmt.Fprintf(&sb, "## 领域：%s (评分: %d)\n", report.DomainName, report.Score)
			fmt.Fprintf(&sb, "### 综述\n%s\n", report.Overview)
			fmt.Fprintf(&sb, "### 趋势\n%s\n", report.Trends)
			fmt.Fprintf(&sb, "### 关键事件\n- %s\n\n", strings.Join(report.KeyEvents, "\n- "))
		}

		analysis, err := deepInterpretReport(ctx, e.chatModel, sb.String(), opts.Persona, e.limiter)
		if err != nil {
			logger.Log.Errorf("深度解读失败: %v", err)
		} else {
			if e.store != nil && runID > 0 {
				if err := e.store.SaveDeepAnalysis(runID, opts.UserID, analysis); err != nil {
					logger.Log.Errorf("保存深度解读失败: %v", err)
				}
				if analysis.Title != "" {
					e.store.UpdateRunTitle(runID, analysis.Title)
				}
			}
		}
	}

	if opts.ProgressCallback != nil {
		opts.ProgressCallback("completed", 100)
	}
	return nil
}

// 辅助函数 (从 main.go 复制并适配)

func fetchAndCleanContent(url string) (string, error) {
	article, err := readability.FromURL(url, 30*time.Second)
	if err != nil {
		return "", err
	}
	return article.TextContent, nil
}

func generateDomainReport(ctx context.Context, cm model.ChatModel, domain string, articles []dm.Article, limiter *rate.Limiter) (*dm.DomainReport, error) {
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

func deepInterpretReport(ctx context.Context, cm model.ChatModel, content string, userPersona string, limiter *rate.Limiter) (*dm.DeepAnalysisResult, error) {
	promptTpl := `Role: 资深技术顾问与个人发展战略专家
Context
用户画像：%s
输入数据：这是一份多领域的每日新闻总结报告。
核心诉求：请跨领域交叉分析，识别宏观趋势，并为用户提供战略建议。

Instructions
请严格按照 JSON 格式输出：
{
    "title": "根据今日所有领域内容生成一个吸引人的简短标题（20字以内）",
    "macro_trends": "Markdown格式的核心趋势洞察...",
    "opportunities": "Markdown格式的机遇挖掘...",
    "risks": "Markdown格式的风险预警...",
    "action_guides": ["行动建议1", "行动建议2", "行动建议3"]
}

输入的新闻总结数据：
%s`

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
