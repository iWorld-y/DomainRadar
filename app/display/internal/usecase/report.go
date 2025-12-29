package usecase

import (
	"context"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/iWorld-y/domain_radar/app/display/internal/domain"
	"github.com/iWorld-y/domain_radar/app/display/internal/repo"
)

// ReportUseCase 报表业务逻辑
type ReportUseCase struct {
	repo repo.ReportRepo
	log  *log.Helper
}

// NewReportUseCase 创建报表业务逻辑实例
func NewReportUseCase(repo repo.ReportRepo, logger log.Logger) *ReportUseCase {
	return &ReportUseCase{repo: repo, log: log.NewHelper(logger)}
}

// List 分页列出报表摘要
func (uc *ReportUseCase) List(ctx context.Context, page, pageSize int) ([]*domain.ReportSummary, int, error) {
	return uc.repo.ListReports(ctx, page, pageSize)
}

// GetByID 根据ID获取报表详情
func (uc *ReportUseCase) GetByID(ctx context.Context, id int, userID int) (*domain.GroupedReport, error) {
	return uc.repo.GetReportByID(ctx, id, userID)
}
