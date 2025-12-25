package data

import (
	"context"
	"time"

	"entgo.io/ent/dialect/sql"
	"github.com/go-kratos/kratos/v2/errors"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/iWorld-y/domain_radar/app/display/internal/biz"
	"github.com/iWorld-y/domain_radar/app/common/ent"
	"github.com/iWorld-y/domain_radar/app/common/ent/domainreport"
	"github.com/iWorld-y/domain_radar/app/common/ent/reportrun"
)

type reportRepo struct {
	data *Data
	log  *log.Helper
}

func NewReportRepo(data *Data, logger log.Logger) biz.ReportRepo {
	return &reportRepo{
		data: data,
		log:  log.NewHelper(logger),
	}
}

func (r *reportRepo) ListReports(ctx context.Context, page, pageSize int) ([]*biz.ReportSummary, int, error) {
	offset := (page - 1) * pageSize

	var results []struct {
		ID          int       `sql:"id"`
		Title       string    `sql:"title"`
		CreatedAt   time.Time `sql:"created_at"`
		DomainCount int       `sql:"domain_count"`
		AvgScore    float64   `sql:"avg_score"`
	}

	// Using Modify to perform custom SQL aggregation
	err := r.data.db.ReportRun.Query().
		Limit(pageSize).
		Offset(offset).
		Order(ent.Desc(reportrun.FieldCreatedAt)).
		Modify(func(s *sql.Selector) {
			t := sql.Table(reportrun.Table)
			dr := sql.Table(domainreport.Table)
			s.LeftJoin(dr).On(t.C(reportrun.FieldID), dr.C(domainreport.FieldRunID))
			s.Select(
				t.C(reportrun.FieldID),
				t.C(reportrun.FieldTitle),
				t.C(reportrun.FieldCreatedAt),
				sql.As(sql.Count(dr.C(domainreport.FieldID)), "domain_count"),
				sql.As(sql.Avg(dr.C(domainreport.FieldScore)), "avg_score"),
			)
			s.GroupBy(t.C(reportrun.FieldID), t.C(reportrun.FieldCreatedAt), t.C(reportrun.FieldTitle))
		}).
		Scan(ctx, &results)
	if err != nil {
		return nil, 0, err
	}

	var summaries []*biz.ReportSummary
	for _, res := range results {
		summaries = append(summaries, &biz.ReportSummary{
			ID:           res.ID,
			Title:        res.Title,
			Date:         res.CreatedAt.Format("2006-01-02 15:04:05"),
			DomainCount:  res.DomainCount,
			AverageScore: int(res.AvgScore),
		})
	}

	total, err := r.data.db.ReportRun.Query().Count(ctx)
	if err != nil {
		return nil, 0, err
	}

	return summaries, total, nil
}

func (r *reportRepo) GetReportByID(ctx context.Context, id int) (*biz.GroupedReport, error) {
	run, err := r.data.db.ReportRun.Query().
		Where(reportrun.ID(id)).
		WithDeepAnalysisResults(func(q *ent.DeepAnalysisResultQuery) {
			q.WithActionGuides()
		}).
		WithDomainReports(func(q *ent.DomainReportQuery) {
			q.WithArticles()
			q.WithKeyEvents()
		}).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, errors.NotFound("REPORT_NOT_FOUND", "report not found")
		}
		return nil, err
	}

	grouped := &biz.GroupedReport{
		ID:   run.ID,
		Date: run.CreatedAt.Format("2006-01-02 15:04:05"),
	}

	// Map DeepAnalysis
	if len(run.Edges.DeepAnalysisResults) > 0 {
		da := run.Edges.DeepAnalysisResults[0]
		grouped.DeepAnalysis = &biz.DeepAnalysisResult{
			MacroTrends:   da.MacroTrends,
			Opportunities: da.Opportunities,
			Risks:         da.Risks,
		}
		for _, ag := range da.Edges.ActionGuides {
			grouped.DeepAnalysis.ActionGuides = append(grouped.DeepAnalysis.ActionGuides, ag.GuideContent)
		}
	}

	// Map DomainReports
	for _, dr := range run.Edges.DomainReports {
		rp := &biz.Report{
			ID:         dr.ID,
			DomainName: dr.DomainName,
			Overview:   dr.Overview,
			Trends:     dr.Trends,
			Score:      dr.Score,
		}
		for _, art := range dr.Edges.Articles {
			rp.Articles = append(rp.Articles, biz.Article{
				Title:   art.Title,
				Link:    art.Link,
				Source:  art.Source,
				PubDate: art.PubDate,
			})
		}
		for _, ke := range dr.Edges.KeyEvents {
			rp.KeyEvents = append(rp.KeyEvents, ke.EventContent)
		}
		grouped.Domains = append(grouped.Domains, rp)
	}

	return grouped, nil
}
