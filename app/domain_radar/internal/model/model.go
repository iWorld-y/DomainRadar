package model

// Article 基础文章信息
type Article struct {
	Title   string
	Link    string
	Source  string
	PubDate string
	Content string // 临时存储用于 LLM 分析，不一定展示
}

// DomainReport 领域报告结构体
type DomainReport struct {
	DomainName string
	Overview   string    `json:"overview"`   // 领域综述
	KeyEvents  []string  `json:"key_events"` // 关键事件
	Trends     string    `json:"trends"`     // 趋势分析
	Score      int       `json:"score"`      // 领域热度评分
	Articles   []Article // 引用文章列表
}

// DeepAnalysisResult 全局深度解读
type DeepAnalysisResult struct {
	Title         string   `json:"title"`        // 报告标题
	MacroTrends   string   `json:"macro_trends"`
	Opportunities string   `json:"opportunities"`
	Risks         string   `json:"risks"`
	ActionGuides  []string `json:"action_guides"`
}
