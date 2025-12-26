package tavily

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/iWorld-y/domain_radar/app/domain_radar/pkg/search"
)

const baseURL = "https://api.tavily.com/search"

// Client Tavily API 客户端
type Client struct {
	apiKey string
	client *http.Client
}

// NewClient 创建一个新的 Tavily 客户端
func NewClient(apiKey string) *Client {
	return &Client{
		apiKey: apiKey,
		client: http.DefaultClient,
	}
}

// Ensure Client implements search.Searcher
var _ search.Searcher = (*Client)(nil)

// Search implements search.Searcher
func (c *Client) Search(ctx context.Context, req *search.Request) (*search.Response, error) {
	tavilyReq := SearchRequest{
		Query:             req.Query,
		Topic:             req.Topic,
		MaxResults:        req.MaxResults,
		IncludeRawContent: req.IncludeRawContent,
		StartDate:         req.StartDate,
		EndDate:           req.EndDate,
	}

	resp, err := c.doSearch(ctx, tavilyReq)
	if err != nil {
		return nil, err
	}

	var results []search.Result
	for _, r := range resp.Results {
		results = append(results, search.Result{
			Title:         r.Title,
			URL:           r.URL,
			Content:       r.Content,
			RawContent:    r.RawContent,
			Score:         r.Score,
			PublishedDate: r.PublishedDate,
		})
	}

	return &search.Response{Results: results}, nil
}

// SearchRequest Tavily 搜索请求参数
type SearchRequest struct {
	Query             string   `json:"query"`
	SearchDepth       string   `json:"search_depth,omitempty"`        // basic or advanced
	Topic             string   `json:"topic,omitempty"`               // general or news
	MaxResults        int      `json:"max_results,omitempty"`
	IncludeRawContent bool     `json:"include_raw_content,omitempty"`
	IncludeImages     bool     `json:"include_images,omitempty"`
	IncludeAnswer     bool     `json:"include_answer,omitempty"`
	IncludeDomains    []string `json:"include_domains,omitempty"`
	ExcludeDomains    []string `json:"exclude_domains,omitempty"`
	StartDate         string   `json:"start_date,omitempty"`
	EndDate           string   `json:"end_date,omitempty"`
}

// SearchResponse Tavily 搜索响应
type SearchResponse struct {
	Query   string         `json:"query"`
	Results []SearchResult `json:"results"`
	Answer  string         `json:"answer"`
}

// SearchResult 单个搜索结果
type SearchResult struct {
	Title         string  `json:"title"`
	URL           string  `json:"url"`
	Content       string  `json:"content"`
	RawContent    string  `json:"raw_content"`
	Score         float64 `json:"score"`
	PublishedDate string  `json:"published_date"`
}

// doSearch 执行搜索 (Internal)
func (c *Client) doSearch(ctx context.Context, req SearchRequest) (*SearchResponse, error) {
	// 设置默认值
	if req.SearchDepth == "" {
		req.SearchDepth = "basic"
	}
	if req.MaxResults == 0 {
		req.MaxResults = 5
	}
	// 根据用户需求，搜索新闻时 topic 最好设为 news，或者由调用方指定
	if req.Topic == "" {
		req.Topic = "general"
	}

	payload, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request failed: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", baseURL, bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("create request failed: %w", err)
	}

	httpReq.Header.Add("Authorization", "Bearer "+c.apiKey)
	httpReq.Header.Add("Content-Type", "application/json")

	res, err := c.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("read body failed: %w", err)
	}

	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("tavily api error (status %d): %s", res.StatusCode, string(body))
	}

	var searchResp SearchResponse
	if err := json.Unmarshal(body, &searchResp); err != nil {
		return nil, fmt.Errorf("unmarshal response failed: %w", err)
	}

	return &searchResp, nil
}
