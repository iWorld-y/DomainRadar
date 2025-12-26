package searxng

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/iWorld-y/domain_radar/app/domain_radar/pkg/search"
)

// Client SearXNG API 客户端
type Client struct {
	baseURL string
	timeout time.Duration
	client  *http.Client
}

// NewClient 创建一个新的 SearXNG 客户端
func NewClient(baseURL string, timeout int) *Client {
	t := time.Duration(timeout) * time.Second
	if t == 0 {
		t = 30 * time.Second
	}
	return &Client{
		baseURL: baseURL,
		timeout: t,
		client: &http.Client{
			Timeout: t,
		},
	}
}

// Ensure Client implements search.Searcher
var _ search.Searcher = (*Client)(nil)

// SearchResponse SearXNG 响应结构
type SearchResponse struct {
	Query   string         `json:"query"`
	Results []SearchResult `json:"results"`
}

// SearchResult SearXNG 单条结果
type SearchResult struct {
	Title         string  `json:"title"`
	URL           string  `json:"url"`
	Content       string  `json:"content"`
	PublishedDate string  `json:"publishedDate"` // 注意: 字段名可能因版本而异，SearXNG 通常使用 publishedDate
	Score         float64 `json:"score"`
}

// Search 执行搜索
func (c *Client) Search(ctx context.Context, req *search.Request) (*search.Response, error) {
	u, err := url.Parse(c.baseURL)
	if err != nil {
		return nil, fmt.Errorf("invalid base URL: %w", err)
	}
	u.Path = "/search"

	q := u.Query()
	q.Set("q", req.Query)
	q.Set("format", "json")

	// 映射 Topic
	if req.Topic == "news" {
		q.Set("categories", "news")
	} else {
		q.Set("categories", "general")
	}

	// 映射语言 (默认中文/英文? 这里先不设，或设为 auto)
	// q.Set("language", "auto")

	u.RawQuery = q.Encode()

	httpReq, err := http.NewRequestWithContext(ctx, "GET", u.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("create request failed: %w", err)
	}

	// 添加 User-Agent 避免被简单的反爬虫策略拦截
	httpReq.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")

	res, err := c.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(res.Body)
		return nil, fmt.Errorf("searxng api error (status %d): %s", res.StatusCode, string(body))
	}

	var searchResp SearchResponse
	if err := json.NewDecoder(res.Body).Decode(&searchResp); err != nil {
		return nil, fmt.Errorf("decode response failed: %w", err)
	}

	var results []search.Result
	for _, r := range searchResp.Results {
		results = append(results, search.Result{
			Title:         r.Title,
			URL:           r.URL,
			Content:       r.Content,
			Score:         r.Score,
			PublishedDate: r.PublishedDate,
		})
	}

	return &search.Response{Results: results}, nil
}
