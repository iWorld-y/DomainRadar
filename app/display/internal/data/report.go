package data

import (
	"context"
	"database/sql"
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
	// Group by date of created_at
	rows, err := r.data.db.QueryContext(ctx, `
		SELECT DATE(created_at) as report_date, COUNT(*) as domain_count, AVG(score) as avg_score 
		FROM domain_reports 
		GROUP BY report_date 
		ORDER BY report_date DESC 
		LIMIT $1 OFFSET $2`, pageSize, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var summaries []*biz.ReportSummary
	for rows.Next() {
		var s biz.ReportSummary
		var date sql.NullString
		var avgScore float64
		if err := rows.Scan(&date, &s.DomainCount, &avgScore); err != nil {
			return nil, 0, err
		}
		s.Date = date.String
		s.AverageScore = int(avgScore)
		summaries = append(summaries, &s)
	}

	var total int
	if err := r.data.db.QueryRowContext(ctx, "SELECT COUNT(DISTINCT DATE(created_at)) FROM domain_reports").Scan(&total); err != nil {
		return nil, 0, err
	}

	return summaries, total, nil
}

func (r *reportRepo) GetReportByDate(ctx context.Context, date string) (*biz.GroupedReport, error) {
	rows, err := r.data.db.QueryContext(ctx, `
		SELECT id, domain_name, overview, trends, score, created_at 
		FROM domain_reports 
		WHERE DATE(created_at) = $1`, date)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	grouped := &biz.GroupedReport{Date: date}
	for rows.Next() {
		var rp biz.Report
		var createdAt sql.NullString
		if err := rows.Scan(&rp.ID, &rp.DomainName, &rp.Overview, &rp.Trends, &rp.Score, &createdAt); err != nil {
			return nil, err
		}
		rp.CreatedAt = createdAt.String

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

	if len(grouped.Domains) == 0 {
		return nil, errors.NotFound("REPORT_NOT_FOUND", "report not found for date: "+date)
	}

	return grouped, nil
}
