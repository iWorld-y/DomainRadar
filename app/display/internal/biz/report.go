package biz

import (
	"context"
	"github.com/go-kratos/kratos/v2/log"
)

type Article struct {
	Title   string
	Link    string
	Source  string
	PubDate string
}

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

type ReportRepo interface {
	ListReports(ctx context.Context, page, pageSize int) ([]*Report, int, error)
	GetReport(ctx context.Context, id int) (*Report, error)
}

type ReportUseCase struct {
	repo ReportRepo
	log  *log.Helper
}

func NewReportUseCase(repo ReportRepo, logger log.Logger) *ReportUseCase {
	return &ReportUseCase{repo: repo, log: log.NewHelper(logger)}
}

func (uc *ReportUseCase) List(ctx context.Context, page, pageSize int) ([]*Report, int, error) {
	return uc.repo.ListReports(ctx, page, pageSize)
}

func (uc *ReportUseCase) Get(ctx context.Context, id int) (*Report, error) {
	return uc.repo.GetReport(ctx, id)
}
