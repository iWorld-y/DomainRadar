package search

import "context"

// Searcher 定义通用的搜索接口
type Searcher interface {
	Search(ctx context.Context, req *Request) (*Response, error)
}

// Request 通用搜索请求
type Request struct {
	Query             string
	Topic             string // "news" or "general"
	MaxResults        int
	IncludeRawContent bool
	StartDate         string // Format: YYYY-MM-DD
	EndDate           string // Format: YYYY-MM-DD
}

// Response 通用搜索响应
type Response struct {
	Results []Result
}

// Result 单条搜索结果
type Result struct {
	Title         string
	URL           string
	Content       string
	RawContent    string
	Score         float64
	PublishedDate string
}
