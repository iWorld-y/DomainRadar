package storage

import (
	"database/sql"
	"fmt"

	"github.com/iWorld-y/domain_radar/app/domain_radar/internal/config"
	"github.com/iWorld-y/domain_radar/app/domain_radar/internal/model"
	_ "github.com/lib/pq"
)

type Storage struct {
	db *sql.DB
}

func NewStorage(cfg config.DBConfig) (*Storage, error) {
	connStr := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		cfg.Host, cfg.Port, cfg.User, cfg.Password, cfg.Name)

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, fmt.Errorf("failed to open database connection: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	s := &Storage{db: db}
	if err := s.initSchema(); err != nil {
		return nil, fmt.Errorf("failed to initialize schema: %w", err)
	}

	return s, nil
}

func (s *Storage) Close() error {
	return s.db.Close()
}

func (s *Storage) initSchema() error {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS domain_reports (
			id SERIAL PRIMARY KEY,
			domain_name TEXT NOT NULL,
			overview TEXT,
			trends TEXT,
			score INTEGER,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS articles (
			id SERIAL PRIMARY KEY,
			domain_report_id INTEGER REFERENCES domain_reports(id),
			title TEXT,
			link TEXT,
			source TEXT,
			pub_date TEXT,
			content TEXT
		)`,
		`CREATE TABLE IF NOT EXISTS key_events (
			id SERIAL PRIMARY KEY,
			domain_report_id INTEGER REFERENCES domain_reports(id),
			event_content TEXT
		)`,
		`CREATE TABLE IF NOT EXISTS deep_analysis_results (
			id SERIAL PRIMARY KEY,
			macro_trends TEXT,
			opportunities TEXT,
			risks TEXT,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS action_guides (
			id SERIAL PRIMARY KEY,
			deep_analysis_id INTEGER REFERENCES deep_analysis_results(id),
			guide_content TEXT
		)`,
	}

	for _, query := range queries {
		if _, err := s.db.Exec(query); err != nil {
			return fmt.Errorf("failed to execute query %s: %w", query, err)
		}
	}

	return nil
}

func (s *Storage) SaveDomainReport(report *model.DomainReport) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Insert report
	var reportID int
	err = tx.QueryRow(`
		INSERT INTO domain_reports (domain_name, overview, trends, score)
		VALUES ($1, $2, $3, $4)
		RETURNING id`,
		report.DomainName, report.Overview, report.Trends, report.Score).Scan(&reportID)
	if err != nil {
		return fmt.Errorf("failed to insert domain report: %w", err)
	}

	// Insert articles
	for _, article := range report.Articles {
		_, err = tx.Exec(`
			INSERT INTO articles (domain_report_id, title, link, source, pub_date, content)
			VALUES ($1, $2, $3, $4, $5, $6)`,
			reportID, article.Title, article.Link, article.Source, article.PubDate, article.Content)
		if err != nil {
			return fmt.Errorf("failed to insert article: %w", err)
		}
	}

	// Insert key events
	for _, event := range report.KeyEvents {
		_, err = tx.Exec(`
			INSERT INTO key_events (domain_report_id, event_content)
			VALUES ($1, $2)`,
			reportID, event)
		if err != nil {
			return fmt.Errorf("failed to insert key event: %w", err)
		}
	}

	return tx.Commit()
}

func (s *Storage) SaveDeepAnalysis(analysis *model.DeepAnalysisResult) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Insert analysis
	var analysisID int
	err = tx.QueryRow(`
		INSERT INTO deep_analysis_results (macro_trends, opportunities, risks)
		VALUES ($1, $2, $3)
		RETURNING id`,
		analysis.MacroTrends, analysis.Opportunities, analysis.Risks).Scan(&analysisID)
	if err != nil {
		return fmt.Errorf("failed to insert deep analysis: %w", err)
	}

	// Insert action guides
	for _, guide := range analysis.ActionGuides {
		_, err = tx.Exec(`
			INSERT INTO action_guides (deep_analysis_id, guide_content)
			VALUES ($1, $2)`,
			analysisID, guide)
		if err != nil {
			return fmt.Errorf("failed to insert action guide: %w", err)
		}
	}

	return tx.Commit()
}
