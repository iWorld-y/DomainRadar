package biz

import (
	"context"

	"github.com/go-kratos/kratos/v2/log"
)

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

// ReportRepo 报表仓库接口
type ReportRepo interface {
	// ListReports 分页获取报表摘要列表
	ListReports(ctx context.Context, page, pageSize int) ([]*ReportSummary, int, error)
	// GetReportByID 根据ID获取报表详情
	GetReportByID(ctx context.Context, id int) (*GroupedReport, error)
}

// ReportUseCase 报表业务逻辑
type ReportUseCase struct {
	repo ReportRepo
	log  *log.Helper
}

// NewReportUseCase 创建报表业务逻辑实例
func NewReportUseCase(repo ReportRepo, logger log.Logger) *ReportUseCase {
	return &ReportUseCase{repo: repo, log: log.NewHelper(logger)}
}

// List 分页列出报表摘要
func (uc *ReportUseCase) List(ctx context.Context, page, pageSize int) ([]*ReportSummary, int, error) {
	return uc.repo.ListReports(ctx, page, pageSize)
}

// GetByID 根据ID获取报表详情
func (uc *ReportUseCase) GetByID(ctx context.Context, id int) (*GroupedReport, error) {
	return uc.repo.GetReportByID(ctx, id)
}
