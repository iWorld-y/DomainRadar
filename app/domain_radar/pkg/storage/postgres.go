package storage

import (
	"context"
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/iWorld-y/domain_radar/app/common/ent"
	"github.com/iWorld-y/domain_radar/app/common/ent/user"
	"github.com/iWorld-y/domain_radar/app/domain_radar/pkg/config"
	"github.com/iWorld-y/domain_radar/app/domain_radar/pkg/model"
	_ "github.com/lib/pq"
)

type Storage struct {
	client *ent.Client
}

func NewStorage(cfg config.DBConfig) (*Storage, error) {
	connStr := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		cfg.Host, cfg.Port, cfg.User, cfg.Password, cfg.Name)

	client, err := ent.Open("postgres", connStr)
	if err != nil {
		return nil, fmt.Errorf("failed to open database connection: %w", err)
	}

	// Run auto migration
	if err := client.Schema.Create(context.Background()); err != nil {
		client.Close()
		return nil, fmt.Errorf("failed to initialize schema: %w", err)
	}

	return &Storage{client: client}, nil
}

func (s *Storage) Close() error {
	return s.client.Close()
}

func (s *Storage) CreateRun() (int, error) {
	r, err := s.client.ReportRun.Create().Save(context.Background())
	if err != nil {
		return 0, err
	}
	return r.ID, nil
}

func (s *Storage) UpdateRunTitle(runID int, title string) error {
	return s.client.ReportRun.UpdateOneID(runID).
		SetTitle(title).
		Exec(context.Background())
}

func (s *Storage) GetUsersWithPersona() ([]*ent.User, error) {
	return s.client.User.Query().
		Where(user.PersonaNEQ("")).
		All(context.Background())
}

func (s *Storage) SaveDomainReport(runID int, report *model.DomainReport) error {
	ctx := context.Background()
	tx, err := s.client.Tx(ctx)
	if err != nil {
		return err
	}

	// Create DomainReport
	dr, err := tx.DomainReport.Create().
		SetRunID(runID).
		SetDomainName(report.DomainName).
		SetOverview(report.Overview).
		SetTrends(report.Trends).
		SetScore(report.Score).
		Save(ctx)
	if err != nil {
		if rerr := tx.Rollback(); rerr != nil {
			err = fmt.Errorf("%w: %v", err, rerr)
		}
		return err
	}

	// Create Articles
	if len(report.Articles) > 0 {
		builders := make([]*ent.ArticleCreate, len(report.Articles))
		for i, art := range report.Articles {
			content := art.Content
			// 移除无效的 UTF-8 字符
			if !utf8.ValidString(content) {
				v := make([]rune, 0, len(content))
				for _, r := range content {
					if r == utf8.RuneError {
						continue
					}
					v = append(v, r)
				}
				content = string(v)
			}
			// 移除 NULL 字符，PostgreSQL 文本字段不支持 NULL 字节
			content = removeNullBytes(content)

			builders[i] = tx.Article.Create().
				SetDomainReportID(dr.ID).
				SetTitle(art.Title).
				SetLink(art.Link).
				SetSource(art.Source).
				SetPubDate(art.PubDate).
				SetContent(content)
		}
		if _, err := tx.Article.CreateBulk(builders...).Save(ctx); err != nil {
			if rerr := tx.Rollback(); rerr != nil {
				err = fmt.Errorf("%w: %v", err, rerr)
			}
			return err
		}
	}

	// Create KeyEvents
	if len(report.KeyEvents) > 0 {
		builders := make([]*ent.KeyEventCreate, len(report.KeyEvents))
		for i, event := range report.KeyEvents {
			builders[i] = tx.KeyEvent.Create().
				SetDomainReportID(dr.ID).
				SetEventContent(event)
		}
		if _, err := tx.KeyEvent.CreateBulk(builders...).Save(ctx); err != nil {
			if rerr := tx.Rollback(); rerr != nil {
				err = fmt.Errorf("%w: %v", err, rerr)
			}
			return err
		}
	}

	return tx.Commit()
}

func (s *Storage) SaveDeepAnalysis(runID int, userID int, result *model.DeepAnalysisResult) error {
	ctx := context.Background()
	tx, err := s.client.Tx(ctx)
	if err != nil {
		return err
	}

	// Create DeepAnalysisResult
	da, err := tx.DeepAnalysisResult.Create().
		SetRunID(runID).
		SetUserID(userID).
		SetMacroTrends(result.MacroTrends).
		SetOpportunities(result.Opportunities).
		SetRisks(result.Risks).
		Save(ctx)
	if err != nil {
		if rerr := tx.Rollback(); rerr != nil {
			err = fmt.Errorf("%w: %v", err, rerr)
		}
		return err
	}

	// Create ActionGuides
	if len(result.ActionGuides) > 0 {
		builders := make([]*ent.ActionGuideCreate, len(result.ActionGuides))
		for i, guide := range result.ActionGuides {
			builders[i] = tx.ActionGuide.Create().
				SetDeepAnalysisID(da.ID).
				SetGuideContent(guide)
		}
		if _, err := tx.ActionGuide.CreateBulk(builders...).Save(ctx); err != nil {
			if rerr := tx.Rollback(); rerr != nil {
				err = fmt.Errorf("%w: %v", err, rerr)
			}
			return err
		}
	}

	return tx.Commit()
}

func removeNullBytes(s string) string {
	return strings.ReplaceAll(s, "\x00", "")
}
