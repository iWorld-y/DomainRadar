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

	"github.com/iWorld-y/domain_radar/src/internal/config"
	"github.com/iWorld-y/domain_radar/src/internal/logger"
	"github.com/iWorld-y/domain_radar/src/internal/tavily"
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

// DeepAnalysisResult 用于解析全局深度解读的 JSON
type DeepAnalysisResult struct {
	MacroTrends   string   `json:"macro_trends"`
	Opportunities string   `json:"opportunities"`
	Risks         string   `json:"risks"`
	ActionGuides  []string `json:"action_guides"`
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
			MaxResults:        2,
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

	// 10. 深度解读 (如果配置了用户画像，且有文章)
	var deepAnalysis *DeepAnalysisResult
	if cfg.UserPersona != "" && len(articles) > 0 {
		logger.Log.Info("正在生成全局深度解读报告...")
		// 拼接摘要
		var sb strings.Builder
		for i, article := range articles {
			fmt.Fprintf(&sb, "%d. 标题：%s\n   分类：%s\n   摘要：%s\n   评分：%d\n\n",
				i+1, article.Title, article.Category, article.Summary, article.Score)
		}
		analysis, err := deepInterpretReport(ctx, chatModel, sb.String(), cfg.UserPersona, limiter)
		if err != nil {
			logger.Log.Errorf("全局深度解读失败: %v", err)
		} else {
			deepAnalysis = analysis
			logger.Log.Info("全局深度解读报告生成完成")
		}
	}

	// 11. 生成 HTML
	if err := generateHTML(articles, deepAnalysis); err != nil {
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

// deepInterpretReport 全局深度解读报告
func deepInterpretReport(ctx context.Context, cm model.ChatModel, content string, userPersona string, limiter *rate.Limiter) (*DeepAnalysisResult, error) {
	maxRetries := 3
	baseDelay := 2 * time.Second
	var lastErr error

	promptTpl := `Role: 资深技术顾问与个人发展战略专家
核心能力：具备极敏锐的技术嗅觉与宏观视野，擅长从碎片化的新闻资讯中提炼出对特定用户最具价值的趋势判断、机会挖掘与风险预警。

Context
用户画像：%s
核心诉求：基于这一组新闻快讯，结合我的个人情况，进行全局性的深度分析。不要逐条点评新闻，而是要综合分析这些信息背后反映的宏观趋势，并给出针对性的建议。

Instructions
请执行以下分析步骤，并严格按照 JSON 格式输出：

1. 🔍 **核心趋势洞察 (Macro Trends)**
   - 综合所有新闻，识别出当前技术或行业的主要风向（例如：某个技术栈的崛起/衰落、政策监管的收紧/放松、新的商业模式等）。
   - 结合用户画像，指出这些趋势对"我"的职业护城河有何具体影响（正面或负面）。

2. 🚀 **机遇挖掘 (Opportunities)**
   - **职业发展**：有哪些新技术、新工具或新领域值得我现在开始投入精力学习？
   - **资产/副业**：是否有值得关注的投资方向或独立开发者机会？
   - 请务必具体，避免泛泛而谈（例如：不要只说"关注AI"，要说"关注AI在xx场景下的落地应用"）。

3. 🛡️ **风险预警 (Risks)**
   - **技术债风险**：我当前的技术栈是否面临被边缘化的风险？
   - **行业风险**：是否有政策或市场变化可能影响我的就业稳定性？
   - 给出具体的"避坑"建议。

4. 💡 **行动指南 (Actionable Advice)**
   - 给出 3 条在这个时间节点，我最应该做的具体行动建议（Action Items）。
   - 建议需具备实操性，符合"低成本试错"或"高杠杆收益"原则。

输出格式要求：
请务必严格按照以下 JSON 格式返回，不要包含任何 markdown 标记（如 '''json）或其他开场白/结束语：
{
    "macro_trends": "Markdown格式的核心趋势洞察内容...",
    "opportunities": "Markdown格式的机遇挖掘内容...",
    "risks": "Markdown格式的风险预警内容...",
    "action_guides": [
        "行动建议1",
        "行动建议2",
        "行动建议3"
    ]
}

注意：
- JSON 中的字符串字段支持 Markdown 格式（如 **加粗**）。
- 语气要客观、专业且真诚，像一位值得信赖的导师。
- 重点关注与用户画像高度相关的内容，忽略无关的噪音。

待分析的新闻列表：
%s`

	for i := 0; i <= maxRetries; i++ {
		if err := limiter.Wait(ctx); err != nil {
			return nil, fmt.Errorf("limiter wait error: %w", err)
		}

		messages := []*schema.Message{
			{
				Role:    schema.System,
				Content: "你是一个 JSON 生成器。请只输出 JSON 字符串，不要输出任何其他内容。",
			},
			{
				Role:    schema.User,
				Content: fmt.Sprintf(promptTpl, userPersona, content),
			},
		}

		resp, err := cm.Generate(ctx, messages)
		if err != nil {
			if strings.Contains(err.Error(), "429") || strings.Contains(strings.ToLower(err.Error()), "too many requests") {
				lastErr = err
				if i < maxRetries {
					delay := baseDelay * time.Duration(1<<i)
					logger.Log.Warnf("触发 429 限流 (深度解读)，等待 %v 后重试 (%d/%d)...", delay, i+1, maxRetries)
					select {
					case <-ctx.Done():
						return nil, ctx.Err()
					case <-time.After(delay):
						continue
					}
				}
			}
			return nil, err
		}

		cleanContent := strings.TrimSpace(resp.Content)
		cleanContent = strings.TrimPrefix(cleanContent, "```json")
		cleanContent = strings.TrimPrefix(cleanContent, "```")
		cleanContent = strings.TrimSuffix(cleanContent, "```")

		var result DeepAnalysisResult
		if err := json.Unmarshal([]byte(cleanContent), &result); err != nil {
			lastErr = fmt.Errorf("json unmarshal error: %w, content: %s", err, cleanContent)
			if i < maxRetries {
				logger.Log.Warnf("深度解读 JSON 解析失败，重试 (%d/%d): %v", i+1, maxRetries, lastErr)
				continue
			}
			return nil, lastErr
		}

		return &result, nil
	}

	return nil, fmt.Errorf("max retries exceeded: %v", lastErr)
}

// generateHTML 渲染模板
func generateHTML(articles []Article, deepAnalysis *DeepAnalysisResult) error {
	const htmlTpl = `
<!DOCTYPE html>
<html lang="zh-CN">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>领域雷达 | 每日精选</title>
    <script src="https://cdn.jsdelivr.net/npm/marked/marked.min.js"></script>
    <style>
        :root {
            --primary-color: #2563eb;
            --bg-color: #f8fafc;
            --card-bg: #ffffff;
            --text-main: #1e293b;
            --text-secondary: #64748b;
            --border-color: #e2e8f0;
            --accent-red: #ef4444;
            --accent-green: #22c55e;
            --accent-yellow: #eab308;
            --accent-purple: #a855f7;
        }
        body {
            font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, "Helvetica Neue", Arial, sans-serif;
            background-color: var(--bg-color);
            color: var(--text-main);
            line-height: 1.6;
            margin: 0;
            padding: 20px;
        }
        .container {
            max-width: 800px;
            margin: 0 auto;
        }
        header {
            text-align: center;
            margin-bottom: 40px;
            padding: 20px 0;
        }
        h1 {
            font-size: 2.5rem;
            color: var(--text-main);
            margin: 0 0 10px 0;
            letter-spacing: -0.025em;
        }
        .date-info {
            color: var(--text-secondary);
            font-size: 1rem;
        }
        .article-card {
            background: var(--card-bg);
            border-radius: 12px;
            padding: 24px;
            margin-bottom: 24px;
            box-shadow: 0 4px 6px -1px rgba(0, 0, 0, 0.1), 0 2px 4px -1px rgba(0, 0, 0, 0.06);
            transition: transform 0.2s, box-shadow 0.2s;
            border: 1px solid var(--border-color);
        }
        .article-card:hover {
            transform: translateY(-2px);
            box-shadow: 0 10px 15px -3px rgba(0, 0, 0, 0.1), 0 4px 6px -2px rgba(0, 0, 0, 0.05);
        }
        .card-header {
            display: flex;
            justify-content: space-between;
            align-items: flex-start;
            margin-bottom: 12px;
            gap: 16px;
        }
        .title {
            font-size: 1.4rem;
            font-weight: 700;
            color: var(--text-main);
            text-decoration: none;
            line-height: 1.4;
            flex: 1;
        }
        .title:hover {
            color: var(--primary-color);
        }
        .score-badge {
            background-color: #fee2e2;
            color: #991b1b;
            padding: 4px 12px;
            border-radius: 9999px;
            font-weight: bold;
            font-size: 0.9rem;
            white-space: nowrap;
            display: flex;
            align-items: center;
        }
        .score-high {
            background-color: #dcfce7;
            color: #166534;
        }
        .meta-row {
            display: flex;
            flex-wrap: wrap;
            gap: 12px;
            align-items: center;
            margin-bottom: 16px;
            font-size: 0.85rem;
            color: var(--text-secondary);
        }
        .tag {
            padding: 2px 10px;
            border-radius: 6px;
            font-weight: 500;
            background-color: #f1f5f9;
            color: var(--text-secondary);
        }
        .tag-category {
            background-color: #e0f2fe;
            color: #0369a1;
        }
        .summary {
            background-color: #f8fafc;
            padding: 16px;
            border-radius: 8px;
            color: #334155;
            font-size: 1rem;
            border-left: 4px solid var(--primary-color);
        }
        .deep-analysis {
            background: var(--card-bg);
            padding: 24px;
            border-radius: 12px;
            margin-bottom: 32px;
            border: 1px solid var(--border-color);
            box-shadow: 0 4px 6px -1px rgba(0, 0, 0, 0.1);
        }
        .analysis-header {
            font-size: 1.2rem;
            font-weight: bold;
            margin-bottom: 20px;
            display: flex;
            align-items: center;
            gap: 8px;
            color: var(--text-main);
        }
        .analysis-grid {
            display: grid;
            grid-template-columns: 1fr;
            gap: 20px;
        }
        @media (min-width: 768px) {
            .analysis-grid {
                grid-template-columns: 1fr 1fr;
            }
            .analysis-section.full-width {
                grid-column: span 2;
            }
        }
        .analysis-section {
            background-color: #f8fafc;
            padding: 20px;
            border-radius: 8px;
            border-left: 4px solid #cbd5e1;
        }
        .analysis-section h3 {
            margin-top: 0;
            font-size: 1.1rem;
            color: var(--text-main);
            display: flex;
            align-items: center;
            gap: 8px;
        }
        .section-trends { border-left-color: var(--primary-color); background-color: #eff6ff; }
        .section-trends h3 { color: #1e40af; }
        
        .section-opportunities { border-left-color: var(--accent-green); background-color: #f0fdf4; }
        .section-opportunities h3 { color: #166534; }
        
        .section-risks { border-left-color: var(--accent-red); background-color: #fef2f2; }
        .section-risks h3 { color: #991b1b; }
        
        .section-actions { border-left-color: var(--accent-purple); background-color: #faf5ff; }
        .section-actions h3 { color: #6b21a8; }
        
        .markdown-content p { margin: 0 0 10px 0; }
        .markdown-content p:last-child { margin: 0; }
        .markdown-content ul { margin: 0; padding-left: 20px; }

        .footer {
            text-align: center;
            margin-top: 40px;
            color: var(--text-secondary);
            font-size: 0.9rem;
        }
        @media (max-width: 600px) {
            .card-header {
                flex-direction: column-reverse;
                gap: 8px;
            }
            .score-badge {
                align-self: flex-start;
            }
        }
    </style>
</head>
<body>
    <div class="container">
        <header>
            <h1>☕️ 领域雷达</h1>
            <div class="date-info">{{ .Date }} • 精选 {{ .Count }} 篇优质内容</div>
        </header>
        
        {{if .DeepAnalysis}}
        <div class="deep-analysis">
            <div class="analysis-header">💡 全局深度解读</div>
            <div class="analysis-grid">
                <div class="analysis-section full-width section-trends">
                    <h3>🔍 核心趋势洞察</h3> 
                    <div class="markdown-content" id="render-trends"></div>
                    <div style="display:none" id="raw-trends">{{.DeepAnalysis.MacroTrends}}</div>
                </div>
                
                <div class="analysis-section section-opportunities">
                    <h3>🚀 机遇挖掘</h3>
                    <div class="markdown-content" id="render-opps"></div>
                    <div style="display:none" id="raw-opps">{{.DeepAnalysis.Opportunities}}</div>
                </div>
                
                <div class="analysis-section section-risks">
                    <h3>🛡️ 风险预警</h3>
                    <div class="markdown-content" id="render-risks"></div>
                    <div style="display:none" id="raw-risks">{{.DeepAnalysis.Risks}}</div>
                </div>
                
                <div class="analysis-section full-width section-actions">
                    <h3>💡 行动指南</h3>
                    <ul style="padding-left: 20px; margin: 0;">
                    {{range .DeepAnalysis.ActionGuides}}
                        <li>{{.}}</li>
                    {{end}}
                    </ul>
                </div>
            </div>
        </div>
        <script>
            document.getElementById('render-trends').innerHTML = marked.parse(document.getElementById('raw-trends').textContent);
            document.getElementById('render-opps').innerHTML = marked.parse(document.getElementById('raw-opps').textContent);
            document.getElementById('render-risks').innerHTML = marked.parse(document.getElementById('raw-risks').textContent);
        </script>
        {{end}}

        {{range .Articles}}
        <article class="article-card">
            <div class="card-header">
                <a href="{{.Link}}" class="title" target="_blank">{{.Title}}</a>
                <div class="score-badge {{if ge .Score 8}}score-high{{end}}">
                    Score: {{.Score}}
                </div>
            </div>
            
            <div class="meta-row">
                <span class="tag tag-category">{{.Category}}</span>
                <span>来源: {{.Source}}</span>
                <span>•</span>
                <span>{{.PubDate}}</span>
            </div>
            
            <div class="summary">
                {{.Summary}}
            </div>
        </article>
        {{end}}

        <div class="footer">
            Generated by Domain Radar
        </div>
    </div>
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
		Date         string
		Count        int
		Articles     []Article
		DeepAnalysis *DeepAnalysisResult
	}{
		Date:         time.Now().Format("2006-01-02"),
		Count:        len(articles),
		Articles:     articles,
		DeepAnalysis: deepAnalysis,
	}

	return t.Execute(f, data)
}
