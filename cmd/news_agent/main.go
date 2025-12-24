package main

import (
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
	"github.com/go-shiori/go-readability"
	"golang.org/x/time/rate"

	"github.com/iWorld-y/news_agent/internal/config"
	"github.com/iWorld-y/news_agent/internal/logger"
	"github.com/iWorld-y/news_agent/internal/tavily"
)

// Article 结构体用于存储处理后的文章
type Article struct {
	Title    string
	Link     string
	Source   string
	Summary  string
	PubDate  string
	Category string // 新增：文章分类
	Score    int    // 新增：重要性评分
}

// LLMResponse 用于解析 LLM 返回的 JSON
type LLMResponse struct {
	TitleZh  string `json:"title_zh"` // 新增：中文标题
	Summary  string `json:"summary"`
	Category string `json:"category"`
	Score    int    `json:"score"`
}

func main() {
	// 1. 加载配置
	cfg, err := config.LoadConfig("config.yaml")
	if err != nil {
		log.Fatalf("无法加载配置文件: %v", err)
	}

	// 验证配置
	if cfg.TavilyAPIKey == "" {
		log.Fatal("配置错误: 未设置 tavily_api_key")
	}
	if len(cfg.Topics) == 0 {
		log.Fatal("配置错误: 未设置感兴趣的话题 (topics)")
	}

	// 2. 初始化日志
	if err = logger.InitLogger(cfg.Log.Level, cfg.Log.File); err != nil {
		log.Fatalf("无法初始化日志: %v", err)
	}
	logger.Log.Info("启动新闻代理...")

	ctx := context.Background()

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
	// Limit 设置为 RPM/60，Burst 设置为 QPS
	limit := rate.Limit(float64(cfg.Concurrency.RPM) / 60.0)
	burst := cfg.Concurrency.QPS
	limiter := rate.NewLimiter(limit, burst)
	logger.Log.Infof("限流器已配置: Limit=%.2f req/s, Burst=%d", limit, burst)

	var articles []Article
	var wg sync.WaitGroup
	var mu sync.Mutex // 保护 articles 切片

	// 5. 初始化 Tavily 客户端
	tavilyClient := tavily.NewClient(cfg.TavilyAPIKey)

	// 计算日期范围 (最近 3 天)
	now := time.Now()
	endDate := now.Format(time.DateOnly)
	startDate := now.AddDate(0, 0, -3).Format(time.DateOnly)

	// 6. 遍历话题进行搜索
	for _, topic := range cfg.Topics {
		logger.Log.Infof("正在搜索话题: %s", topic)

		req := tavily.SearchRequest{
			Query:             topic,
			Topic:             "news",
			MaxResults:        5,
			StartDate:         startDate,
			EndDate:           endDate,
			IncludeRawContent: false,
		}

		resp, err := tavilyClient.Search(req)
		if err != nil {
			logger.Log.Errorf("搜索话题失败 [%s]: %v", topic, err)
			continue
		}

		for _, item := range resp.Results {
			wg.Add(1)
			go func(item tavily.SearchResult, topic string) {
				defer wg.Done()

				// 7. 获取并清洗正文
				// 优先使用 Tavily 返回的内容，如果太短则尝试抓取
				content := item.Content
				if len(content) < 200 {
					fetchedContent, err := fetchAndCleanContent(item.URL)
					if err == nil && len(fetchedContent) > len(content) {
						content = fetchedContent
					} else if err != nil {
						logger.Log.Warnf("原文抓取失败，使用 Tavily 摘要 [%s]: %v", item.Title, err)
					}
				}

				// 截断内容以防止超出 Token 限制
				if len(content) > 6000 {
					content = content[:6000]
				}

				// 8. 调用 LLM 生成总结、分类和评分
				llmResp, err := summarizeContent(ctx, chatModel, content, item.Title, limiter)
				if err != nil {
					logger.Log.Errorf("总结失败 [%s]: %v", item.Title, err)
					return
				}

				// 如果 LLM 返回了中文标题且不为空，则使用中文标题
				finalTitle := item.Title
				if llmResp.TitleZh != "" {
					finalTitle = llmResp.TitleZh
				}

				mu.Lock()
				articles = append(articles, Article{
					Title:    finalTitle,
					Link:     item.URL,
					Source:   topic, // 使用话题作为来源，或者使用 item.Domain (如果 API 返回)
					Summary:  llmResp.Summary,
					PubDate:  item.PublishedDate,
					Category: llmResp.Category,
					Score:    llmResp.Score,
				})
				mu.Unlock()
				logger.Log.Infof("已完成: %s (Score: %d)", finalTitle, llmResp.Score)
			}(item, topic)
		}
	}

	wg.Wait()

	// 9. 排序：按重要性从高到低
	sort.Slice(articles, func(i, j int) bool {
		return articles[i].Score > articles[j].Score
	})

	// 10. 生成 HTML
	if err := generateHTML(articles); err != nil {
		logger.Log.Fatalf("生成 HTML 失败: %v", err)
	}

	logger.Log.Info("✅ 早报生成完毕: index.html")
}

// fetchAndCleanContent 抓取 URL 并提取核心文本
func fetchAndCleanContent(url string) (string, error) {
	article, err := readability.FromURL(url, 30*time.Second)
	if err != nil {
		return "", err
	}
	return article.TextContent, nil
}

// summarizeContent 调用 LLM
func summarizeContent(ctx context.Context, cm model.ChatModel, content string, title string, limiter *rate.Limiter) (*LLMResponse, error) {
	maxRetries := 3
	baseDelay := 2 * time.Second

	var lastErr error

	for i := 0; i <= maxRetries; i++ {
		// 等待限流令牌
		if err := limiter.Wait(ctx); err != nil {
			return nil, fmt.Errorf("limiter wait error: %w", err)
		}

		prompt := `你是一个专业的技术新闻编辑。请阅读用户提供的文章内容和标题，生成一份简明扼要的中文摘要，并进行分类和评分。
如果原标题是英文，请将其翻译为中文；如果原标题已经是中文，则保持原样或进行适当优化。

请务必严格按照以下 JSON 格式返回，不要包含任何 markdown 标记（如 '''json）：
{
	"title_zh": "中文标题（如果原标题是英文则翻译，否则优化或保留）",
	"summary": "中文摘要（100-200字），提取核心观点、新技术或关键事件。",
	"category": "文章分类（例如：人工智能、前端开发、后端架构、云计算、行业资讯、其他）",
	"score": 8
}
评分说明：score 为 1-10 的整数，10分为非常重要，1分为不重要。

文章标题：
%s

文章内容：
%s`

		messages := []*schema.Message{
			{
				Role:    schema.System,
				Content: "你是一个 JSON 生成器。请只输出 JSON 字符串，不要输出任何其他内容。",
			},
			{
				Role:    schema.User,
				Content: fmt.Sprintf(prompt, title, content),
			},
		}

		resp, err := cm.Generate(ctx, messages)
		if err != nil {
			// 检查是否是 429 错误
			if strings.Contains(err.Error(), "429") || strings.Contains(strings.ToLower(err.Error()), "too many requests") {
				lastErr = err
				if i < maxRetries {
					delay := baseDelay * time.Duration(1<<i) // 指数退避
					logger.Log.Warnf("触发 429 限流，等待 %v 后重试 (%d/%d)...", delay, i+1, maxRetries)
					select {
					case <-ctx.Done():
						return nil, ctx.Err()
					case <-time.After(delay):
						continue // 重试
					}
				}
			}
			return nil, err
		}

		// 清理可能的 markdown 标记
		cleanContent := strings.TrimSpace(resp.Content)
		cleanContent = strings.TrimPrefix(cleanContent, "```json")
		cleanContent = strings.TrimPrefix(cleanContent, "```")
		cleanContent = strings.TrimSuffix(cleanContent, "```")

		var llmResp LLMResponse
		if err := json.Unmarshal([]byte(cleanContent), &llmResp); err != nil {
			lastErr = fmt.Errorf("json unmarshal error: %w, content: %s", err, cleanContent)
			if i < maxRetries {
				logger.Log.Warnf("JSON 解析失败，重试 (%d/%d): %v", i+1, maxRetries, lastErr)
				continue // 重试
			}
			return nil, lastErr
		}

		return &llmResp, nil
	}

	return nil, fmt.Errorf("max retries exceeded: %v", lastErr)
}

// generateHTML 渲染模板
func generateHTML(articles []Article) error {
	const htmlTpl = `
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>领域雷达</title>
    <style>
        body { font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, Helvetica, Arial, sans-serif; max-width: 800px; margin: 0 auto; padding: 20px; line-height: 1.6; color: #333; }
        .article { border-bottom: 1px solid #eee; padding-bottom: 20px; margin-bottom: 20px; }
        .title { font-size: 1.2em; font-weight: bold; color: #2c3e50; text-decoration: none; }
        .meta { font-size: 0.9em; color: #7f8c8d; margin-bottom: 10px; }
        .summary { background-color: #f9f9f9; padding: 15px; border-radius: 5px; border-left: 4px solid #3498db; }
        .tag { display: inline-block; padding: 2px 8px; border-radius: 12px; font-size: 0.8em; margin-right: 5px; color: white; }
        .tag-category { background-color: #3498db; }
        .tag-score { background-color: #e74c3c; }
        h1 { text-align: center; color: #2c3e50; }
    </style>
</head>
<body>
    <h1>☕️ 领域雷达</h1>
    <p style="text-align:center; color:#666;">{{ .Date }} • 共 {{ .Count }} 篇文章</p>
    
    {{range .Articles}}
    <div class="article">
        <a href="{{.Link}}" class="title" target="_blank">{{.Title}}</a>
        <div class="meta">
            <span class="tag tag-category">{{.Category}}</span>
            <span class="tag tag-score">评分: {{.Score}}</span>
            来源: {{.Source}} | 时间: {{.PubDate}}
        </div>
        <div class="summary">{{.Summary}}</div>
    </div>
    {{end}}
</body>
</html>`

	t, err := template.New("report").Parse(htmlTpl)
	if err != nil {
		return err
	}

	f, err := os.Create("index.html")
	if err != nil {
		return err
	}
	defer f.Close()

	data := struct {
		Date     string
		Count    int
		Articles []Article
	}{
		Date:     time.Now().Format("2006-01-02"),
		Count:    len(articles),
		Articles: articles,
	}

	return t.Execute(f, data)
}
