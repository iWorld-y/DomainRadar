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

func (r *reportRepo) ListReports(ctx context.Context, page, pageSize int) ([]*biz.Report, int, error) {
	offset := (page - 1) * pageSize
	rows, err := r.data.db.QueryContext(ctx, "SELECT id, domain_name, score, created_at FROM domain_reports ORDER BY created_at DESC LIMIT $1 OFFSET $2", pageSize, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var reports []*biz.Report
	for rows.Next() {
		var rp biz.Report
		var createdAt sql.NullString
		if err := rows.Scan(&rp.ID, &rp.DomainName, &rp.Score, &createdAt); err != nil {
			return nil, 0, err
		}
		rp.CreatedAt = createdAt.String
		reports = append(reports, &rp)
	}

	var total int
	if err := r.data.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM domain_reports").Scan(&total); err != nil {
		return nil, 0, err
	}

	return reports, total, nil
}

func (r *reportRepo) GetReport(ctx context.Context, id int) (*biz.Report, error) {
	var rp biz.Report
	var createdAt sql.NullString
	err := r.data.db.QueryRowContext(ctx, "SELECT id, domain_name, overview, trends, score, created_at FROM domain_reports WHERE id = $1", id).
		Scan(&rp.ID, &rp.DomainName, &rp.Overview, &rp.Trends, &rp.Score, &createdAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, errors.NotFound("REPORT_NOT_FOUND", "report not found")
		}
		return nil, err
	}
	rp.CreatedAt = createdAt.String

	// Get Articles
	aRows, err := r.data.db.QueryContext(ctx, "SELECT title, link, source, pub_date FROM articles WHERE domain_report_id = $1", id)
	if err == nil {
		defer aRows.Close()
		for aRows.Next() {
			var a biz.Article
			aRows.Scan(&a.Title, &a.Link, &a.Source, &a.PubDate)
			rp.Articles = append(rp.Articles, a)
		}
	}

	// Get Key Events
	eRows, err := r.data.db.QueryContext(ctx, "SELECT event_content FROM key_events WHERE domain_report_id = $1", id)
	if err == nil {
		defer eRows.Close()
		for eRows.Next() {
			var e string
			eRows.Scan(&e)
			rp.KeyEvents = append(rp.KeyEvents, e)
		}
	}

	return &rp, nil
}
