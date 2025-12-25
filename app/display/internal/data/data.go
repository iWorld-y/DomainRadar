package data

import (
	"database/sql"
	"fmt"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/iWorld-y/domain_radar/app/display/internal/conf"
	_ "github.com/lib/pq"
)

type Data struct {
	db *sql.DB
}

func NewData(c *conf.Data, logger log.Logger) (*Data, func(), error) {
	connStr := c.Database.Source
	db, err := sql.Open(c.Database.Driver, connStr)
	if err != nil {
		return nil, nil, err
	}
	if err := db.Ping(); err != nil {
		return nil, nil, err
	}

	// Init schema for users
	if _, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS users (
			id SERIAL PRIMARY KEY,
			username TEXT NOT NULL UNIQUE,
			password_hash TEXT NOT NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)
	`); err != nil {
		return nil, nil, fmt.Errorf("failed to init users table: %w", err)
	}

	cleanup := func() {
		log.NewHelper(logger).Info("closing the data resources")
		db.Close()
	}
	return &Data{db: db}, cleanup, nil
}
