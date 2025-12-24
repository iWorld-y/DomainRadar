package main

import (
	"context"
	"fmt"
	"html/template"
	"log"
	"os"
	"sync"
	"time"

	"github.com/go-shiori/go-readability"
	"github.com/mmcdole/gofeed"
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/openai"
)

// 配置部分：替换为 DeepSeek 或 Qwen 的 API 信息
const (
	// 这里以 DeepSeek 为例 (DeepSeek 兼容 OpenAI 协议)
	LLMBaseURL = "https://api.xiaomimimo.com/v1"                       // 或者 Qwen 的地址
	LLMAPIKey  = "sk-cnq3lcmg346ea14lcfve0crv5negsxt4qg17kror2msfi0td" // 你的 API Key
	LLMModel   = "mimo-v2-flash"                                       // 模型名称
)

// Article 结构体用于存储处理后的文章
type Article struct {
	Title   string
	Link    string
	Source  string
	Summary string
	PubDate string
}

func main() {
	// 1. 输入 RSS 列表
	rssLinks := []string{
		"https://www.ruanyifeng.com/blog/atom.xml",          // 阮一峰博客
		"https://sspai.com/feed",                            // 少数派
		"https://tech.meituan.com/feed",                     // 美团技术团队
		"https://plink.anyfeeder.com/zaobao/realtime/china", // 《联合早报》-中港台-即时
		"https://36kr.com/feed",                             // 36氪
		"https://rss.huxiu.com/",                            // 虎嗅
	}

	ctx := context.Background()

	// 2. 初始化 LLM
	llm, err := openai.New(
		openai.WithBaseURL(LLMBaseURL),
		openai.WithToken(LLMAPIKey),
		openai.WithModel(LLMModel),
	)
	if err != nil {
		log.Fatalf("LLM 初始化失败: %v", err)
	}

	var articles []Article
	var wg sync.WaitGroup
	var mu sync.Mutex // 保护 articles 切片

	// 3. 并发处理 RSS
	fp := gofeed.NewParser()
	for _, url := range rssLinks {
		feed, err := fp.ParseURL(url)
		if err != nil {
			log.Printf("解析 RSS 失败 [%s]: %v", url, err)
			continue
		}

		fmt.Printf("正在处理源: %s\n", feed.Title)

		// 只处理最近 24 小时的文章
		for _, item := range feed.Items {
			// 如果没有发布时间，默认处理；如果有，判断是否是今天
			if item.PublishedParsed != nil && time.Since(*item.PublishedParsed) > 24*time.Hour {
				continue
			}

			wg.Add(1)
			go func(item *gofeed.Item, sourceName string) {
				defer wg.Done()

				// 4. 获取并清洗正文
				// RSS item.Description 通常太短，我们需要去抓取原文
				// 使用 readability 提取正文内容
				content, err := fetchAndCleanContent(item.Link)
				if err != nil {
					// 如果抓取失败，回退到使用 RSS 里的摘要
					content = item.Description
					log.Printf("原文抓取失败，使用摘要 [%s]: %v", item.Title, err)
				}

				// 截断内容以防止超出 Token 限制 (简单粗暴截断，生产环境需更精细)
				if len(content) > 6000 {
					content = content[:6000]
				}

				// 5. 调用 LLM 生成总结
				summary, err := summarizeContent(ctx, llm, content)
				if err != nil {
					log.Printf("总结失败 [%s]: %v", item.Title, err)
					return
				}

				mu.Lock()
				articles = append(articles, Article{
					Title:   item.Title,
					Link:    item.Link,
					Source:  sourceName,
					Summary: summary,
					PubDate: item.Published,
				})
				mu.Unlock()
				fmt.Printf("已完成: %s\n", item.Title)
			}(item, feed.Title)
		}
	}

	wg.Wait()

	// 6. 生成 HTML
	if err := generateHTML(articles); err != nil {
		log.Fatalf("生成 HTML 失败: %v", err)
	}

	fmt.Println("✅ 早报生成完毕: morning_report.html")
}

// fetchAndCleanContent 抓取 URL 并提取核心文本
func fetchAndCleanContent(url string) (string, error) {
	// 这里的 timeout 设置很重要，防止挂起
	article, err := readability.FromURL(url, 30*time.Second)
	if err != nil {
		return "", err
	}
	// 返回纯文本内容
	return article.TextContent, nil
}

// summarizeContent 调用 LLM
func summarizeContent(ctx context.Context, llm llms.Model, content string) (string, error) {
	prompt := fmt.Sprintf(`
你是一个专业的技术新闻编辑。请阅读以下文章内容，生成一份简明扼要的中文摘要（100-200字）。
要求：
1. 提取核心观点、新技术或关键事件。
2. 语言通俗易懂，适合快速阅读。
3. 输出格式直接是纯文本，不要 markdown 标题。

文章内容：
%s
`, content)

	completion, err := llms.GenerateFromSinglePrompt(ctx, llm, prompt)
	if err != nil {
		return "", err
	}
	return completion, nil
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
        h1 { text-align: center; color: #2c3e50; }
    </style>
</head>
<body>
    <h1>☕️ AI 每日早报</h1>
    <p style="text-align:center; color:#666;">{{ .Date }} • 共 {{ .Count }} 篇文章</p>
    
    {{range .Articles}}
    <div class="article">
        <a href="{{.Link}}" class="title" target="_blank">{{.Title}}</a>
        <div class="meta">来源: {{.Source}} | 时间: {{.PubDate}}</div>
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
