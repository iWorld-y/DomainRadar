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
	"github.com/mmcdole/gofeed"
	"golang.org/x/time/rate"

	"github.com/iWorld-y/news_agent/internal/config"
	"github.com/iWorld-y/news_agent/internal/logger"
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

	// 5. 并发处理 RSS
	fp := gofeed.NewParser()
	for _, url := range cfg.RSSLinks {
		feed, err := fp.ParseURL(url)
		if err != nil {
			logger.Log.Errorf("解析 RSS 失败 [%s]: %v", url, err)
			continue
		}

		logger.Log.Infof("正在处理源: %s", feed.Title)

		// 只处理最近 24 小时的文章
		for _, item := range feed.Items {
			// 如果没有发布时间，默认处理；如果有，判断是否是今天
			if item.PublishedParsed != nil && time.Since(*item.PublishedParsed) > 24*time.Hour {
				continue
			}

			wg.Add(1)
			go func(item *gofeed.Item, sourceName string) {
				defer wg.Done()

				// 6. 获取并清洗正文
				content, err := fetchAndCleanContent(item.Link)
				if err != nil {
					// 如果抓取失败，回退到使用 RSS 里的摘要
					content = item.Description
					logger.Log.Warnf("原文抓取失败，使用摘要 [%s]: %v", item.Title, err)
				}

				// 截断内容以防止超出 Token 限制
				if len(content) > 6000 {
					content = content[:6000]
				}

				// 7. 调用 LLM 生成总结、分类和评分
				llmResp, err := summarizeContent(ctx, chatModel, content, limiter)
				if err != nil {
					logger.Log.Errorf("总结失败 [%s]: %v", item.Title, err)
					return
				}

				mu.Lock()
				articles = append(articles, Article{
					Title:    item.Title,
					Link:     item.Link,
					Source:   sourceName,
					Summary:  llmResp.Summary,
					PubDate:  item.Published,
					Category: llmResp.Category,
					Score:    llmResp.Score,
				})
				mu.Unlock()
				logger.Log.Infof("已完成: %s (Score: %d)", item.Title, llmResp.Score)
			}(item, feed.Title)
		}
	}

	wg.Wait()

	// 8. 排序：按重要性从高到低
	sort.Slice(articles, func(i, j int) bool {
		return articles[i].Score > articles[j].Score
	})

	// 9. 生成 HTML
	if err := generateHTML(articles); err != nil {
		logger.Log.Fatalf("生成 HTML 失败: %v", err)
	}

	logger.Log.Info("✅ 早报生成完毕: morning_report.html")
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
func summarizeContent(ctx context.Context, cm model.ChatModel, content string, limiter *rate.Limiter) (*LLMResponse, error) {
	// 等待限流令牌
	if err := limiter.Wait(ctx); err != nil {
		return nil, fmt.Errorf("limiter wait error: %w", err)
	}

	prompt := `你是一个专业的技术新闻编辑。请阅读用户提供的文章内容，生成一份简明扼要的中文摘要，并进行分类和评分。

请务必严格按照以下 JSON 格式返回，不要包含任何 markdown 标记（如 '''json）：
{
	"summary": "中文摘要（100-200字），提取核心观点、新技术或关键事件。",
	"category": "文章分类（例如：人工智能、前端开发、后端架构、云计算、行业资讯、其他）",
	"score": 8
}
评分说明：score 为 1-10 的整数，10分为非常重要，1分为不重要。

文章内容：
%s`

	messages := []*schema.Message{
		{
			Role:    schema.System,
			Content: "你是一个 JSON 生成器。请只输出 JSON 字符串，不要输出任何其他内容。",
		},
		{
			Role:    schema.User,
			Content: fmt.Sprintf(prompt, content),
		},
	}

	resp, err := cm.Generate(ctx, messages)
	if err != nil {
		return nil, err
	}

	// 清理可能的 markdown 标记
	cleanContent := strings.TrimSpace(resp.Content)
	cleanContent = strings.TrimPrefix(cleanContent, "```json")
	cleanContent = strings.TrimPrefix(cleanContent, "```")
	cleanContent = strings.TrimSuffix(cleanContent, "```")

	var llmResp LLMResponse
	if err := json.Unmarshal([]byte(cleanContent), &llmResp); err != nil {
		return nil, fmt.Errorf("json unmarshal error: %w, content: %s", err, cleanContent)
	}

	return &llmResp, nil
}

// generateHTML 渲染模板
func generateHTML(articles []Article) error {
	const htmlTpl = `
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>AI 每日早报</title>
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
    <h1>☕️ AI 每日早报</h1>
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

	f, err := os.Create("morning_report.html")
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
