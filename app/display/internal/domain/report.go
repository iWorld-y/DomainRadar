package domain

// Article 关联文章信息
type Article struct {
	Title   string
	Link    string
	Source  string
	PubDate string
}

// Report 报表领域对象
type Report struct {
	ID         int
	DomainName string
	Score      int
	Overview   string
	Trends     string
	KeyEvents  []string
	Articles   []Article
	CreatedAt  string
}

// ReportSummary 报表摘要信息
type ReportSummary struct {
	ID           int
	Title        string
	Date         string
	DomainCount  int
	AverageScore int
}

// DeepAnalysisResult 全局深度解读
type DeepAnalysisResult struct {
	MacroTrends   string
	Opportunities string
	Risks         string
	ActionGuides  []string
}

// GroupedReport 报表详情
type GroupedReport struct {
	ID           int
	Date         string
	Domains      []*Report
	DeepAnalysis *DeepAnalysisResult
}
