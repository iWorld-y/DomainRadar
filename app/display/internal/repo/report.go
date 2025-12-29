package repo

import (
	"context"

	"github.com/iWorld-y/domain_radar/app/display/internal/domain"
)

// ReportRepo 报表仓库接口
type ReportRepo interface {
	// ListReports 分页获取报表摘要列表
	ListReports(ctx context.Context, page, pageSize int) ([]*domain.ReportSummary, int, error)
	// GetReportByID 根据ID获取报表详情
	GetReportByID(ctx context.Context, id int, userID int) (*domain.GroupedReport, error)
}
