package data

import (
	"context"
	"database/sql"
	"time"

	"github.com/go-kratos/kratos/v2/errors"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/iWorld-y/domain_radar/app/display/internal/biz"
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
	rows, err := r.data.db.QueryContext(ctx, `
		SELECT rr.id, rr.created_at, COUNT(dr.id) as domain_count, AVG(dr.score) as avg_score 
		FROM report_runs rr
		LEFT JOIN domain_reports dr ON rr.id = dr.run_id
		GROUP BY rr.id, rr.created_at 
		ORDER BY rr.created_at DESC 
		LIMIT $1 OFFSET $2`, pageSize, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var summaries []*biz.ReportSummary
	for rows.Next() {
		var s biz.ReportSummary
		var createdAt sql.NullTime
		var avgScore sql.NullFloat64
		if err := rows.Scan(&s.ID, &createdAt, &s.DomainCount, &avgScore); err != nil {
			return nil, 0, err
		}
		if createdAt.Valid {
			s.Date = createdAt.Time.Format("2006-01-02 15:04:05")
		}
		s.AverageScore = int(avgScore.Float64)
		summaries = append(summaries, &s)
	}

	var total int
	if err := r.data.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM report_runs").Scan(&total); err != nil {
		return nil, 0, err
	}

	return summaries, total, nil
}

func (r *reportRepo) GetReportByID(ctx context.Context, id int) (*biz.GroupedReport, error) {
	var createdAt time.Time
	err := r.data.db.QueryRowContext(ctx, "SELECT created_at FROM report_runs WHERE id = $1", id).Scan(&createdAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, errors.NotFound("REPORT_NOT_FOUND", "report not found")
		}
		return nil, err
	}

	grouped := &biz.GroupedReport{
		ID:   id,
		Date: createdAt.Format("2006-01-02 15:04:05"),
	}

	// Get Deep Analysis
	var da biz.DeepAnalysisResult
	var daID int
	err = r.data.db.QueryRowContext(ctx, `
		SELECT id, macro_trends, opportunities, risks 
		FROM deep_analysis_results 
		WHERE run_id = $1`, id).Scan(&daID, &da.MacroTrends, &da.Opportunities, &da.Risks)
	if err == nil {
		// Get Action Guides
		gRows, err := r.data.db.QueryContext(ctx, "SELECT guide_content FROM action_guides WHERE deep_analysis_id = $1", daID)
		if err == nil {
			defer gRows.Close()
			for gRows.Next() {
				var g string
				gRows.Scan(&g)
				da.ActionGuides = append(da.ActionGuides, g)
			}
		}
		grouped.DeepAnalysis = &da
	}

	// Get Domain Reports
	rows, err := r.data.db.QueryContext(ctx, `
		SELECT id, domain_name, overview, trends, score 
		FROM domain_reports 
		WHERE run_id = $1`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var rp biz.Report
		if err := rows.Scan(&rp.ID, &rp.DomainName, &rp.Overview, &rp.Trends, &rp.Score); err != nil {
			return nil, err
		}

		// Get Articles
		aRows, err := r.data.db.QueryContext(ctx, "SELECT title, link, source, pub_date FROM articles WHERE domain_report_id = $1", rp.ID)
		if err == nil {
			defer aRows.Close()
			for aRows.Next() {
				var a biz.Article
				aRows.Scan(&a.Title, &a.Link, &a.Source, &a.PubDate)
				rp.Articles = append(rp.Articles, a)
			}
		}

		// Get Key Events
		eRows, err := r.data.db.QueryContext(ctx, "SELECT event_content FROM key_events WHERE domain_report_id = $1", rp.ID)
		if err == nil {
			defer eRows.Close()
			for eRows.Next() {
				var e string
				eRows.Scan(&e)
				rp.KeyEvents = append(rp.KeyEvents, e)
			}
		}
		grouped.Domains = append(grouped.Domains, &rp)
	}

	return grouped, nil
}
