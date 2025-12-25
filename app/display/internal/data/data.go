package data

import (
	"context"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/iWorld-y/domain_radar/app/display/internal/conf"
	"github.com/iWorld-y/domain_radar/ent"
	_ "github.com/lib/pq"
)

type Data struct {
	db *ent.Client
}

func NewData(c *conf.Data, logger log.Logger) (*Data, func(), error) {
	connStr := c.Database.Source
	client, err := ent.Open(c.Database.Driver, connStr)
	if err != nil {
		return nil, nil, err
	}

	// Run the auto migration tool.
	if err := client.Schema.Create(context.Background()); err != nil {
		client.Close()
		return nil, nil, err
	}

	cleanup := func() {
		log.NewHelper(logger).Info("closing the data resources")
		client.Close()
	}
	return &Data{db: client}, cleanup, nil
}
