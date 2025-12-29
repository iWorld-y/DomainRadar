package usecase

import (
	"context"
	"testing"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/iWorld-y/domain_radar/app/display/internal/domain"
)

// mockReportRepo 模拟报表仓库
type mockReportRepo struct{}

func (m *mockReportRepo) ListReports(ctx context.Context, page, pageSize int) ([]*domain.ReportSummary, int, error) {
	return []*domain.ReportSummary{{ID: 1, Title: "Test Report"}}, 1, nil
}

func (m *mockReportRepo) GetReportByID(ctx context.Context, id int, userID int) (*domain.GroupedReport, error) {
	return &domain.GroupedReport{ID: id}, nil
}

func (m *mockReportRepo) SaveReport(ctx context.Context, report *domain.Report) error {
	return nil
}

func TestReportUseCase_List(t *testing.T) {
	repo := &mockReportRepo{}
	logger := log.DefaultLogger
	uc := NewReportUseCase(repo, logger)

	reports, total, err := uc.List(context.Background(), 1, 10)
	if err != nil {
		t.Errorf("List() error = %v", err)
		return
	}
	if total != 1 {
		t.Errorf("List() total = %v, want 1", total)
	}
	if len(reports) != 1 || reports[0].Title != "Test Report" {
		t.Errorf("List() reports = %v", reports)
	}
}
