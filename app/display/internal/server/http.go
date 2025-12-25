package server

import (
	"embed"
	nethttp "net/http"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/middleware/recovery"
	"github.com/go-kratos/kratos/v2/transport/http"
	"github.com/iWorld-y/domain_radar/app/display/api"
	"github.com/iWorld-y/domain_radar/app/display/internal/conf"
	"github.com/iWorld-y/domain_radar/app/display/internal/service"
)

//go:embed assets/*
var assets embed.FS

func NewHTTPServer(c *conf.Server, s *service.DisplayService, logger log.Logger) *http.Server {
	var opts = []http.ServerOption{
		http.Middleware(
			recovery.Recovery(),
		),
	}
	if c.Http.Addr != "" {
		opts = append(opts, http.Address(c.Http.Addr))
	}
	if c.Http.Timeout != "" {
		if d, err := time.ParseDuration(c.Http.Timeout); err == nil {
			opts = append(opts, http.Timeout(d))
		}
	}

	srv := http.NewServer(opts...)
	api.RegisterDisplayHTTPServer(srv, s)

	// Serve Static Assets (HTML)
	// We handle "/" manually to serve index.html

	// Serve specific pages for cleaner URLs
	srv.HandleFunc("/", func(w nethttp.ResponseWriter, r *nethttp.Request) {
		if r.URL.Path == "/" {
			content, _ := assets.ReadFile("assets/index.html")
			w.Write(content)
			return
		}
	})

	srv.HandleFunc("/dashboard", func(w nethttp.ResponseWriter, r *nethttp.Request) {
		content, _ := assets.ReadFile("assets/dashboard.html")
		w.Write(content)
	})

	srv.HandleFunc("/report", func(w nethttp.ResponseWriter, r *nethttp.Request) {
		content, _ := assets.ReadFile("assets/report.html")
		w.Write(content)
	})

	return srv
}
