package server

import (
	"context"
	"embed"
	nethttp "net/http"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/middleware/auth/jwt"
	"github.com/go-kratos/kratos/v2/middleware/recovery"
	"github.com/go-kratos/kratos/v2/middleware/selector"
	"github.com/go-kratos/kratos/v2/transport/http"
	jwtv5 "github.com/golang-jwt/jwt/v5"
	v1 "github.com/iWorld-y/domain_radar/api/proto/display/v1"
	"github.com/iWorld-y/domain_radar/app/display/internal/conf"
	"github.com/iWorld-y/domain_radar/app/display/internal/service"
)

//go:embed assets/*
var assets embed.FS

// NewWhiteListMatcher 创建白名单选择器
func NewWhiteListMatcher() selector.MatchFunc {
	whiteList := make(map[string]struct{})
	whiteList["/api.display.v1.Display/Login"] = struct{}{}
	whiteList["/api.display.v1.Display/Register"] = struct{}{}
	return func(ctx context.Context, operation string) bool {
		if _, ok := whiteList[operation]; ok {
			return false
		}
		return true
	}
}

func NewHTTPServer(c *conf.Server, auth *conf.Auth, s *service.DisplayService, logger log.Logger) *http.Server {
	jwtKey := "default-secret"
	if auth != nil && auth.JwtKey != "" {
		jwtKey = auth.JwtKey
	}

	opts := []http.ServerOption{
		http.Middleware(
			recovery.Recovery(),
			selector.Server(
				jwt.Server(func(token *jwtv5.Token) (interface{}, error) {
					return []byte(jwtKey), nil
				}, jwt.WithSigningMethod(jwtv5.SigningMethodHS256)),
			).Match(NewWhiteListMatcher()).Build(),
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
	v1.RegisterDisplayHTTPServer(srv, s)

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

	srv.HandleFunc("/style.css", func(w nethttp.ResponseWriter, r *nethttp.Request) {
		w.Header().Set("Content-Type", "text/css")
		content, _ := assets.ReadFile("assets/style.css")
		w.Write(content)
	})

	srv.HandleFunc("/dashboard", func(w nethttp.ResponseWriter, r *nethttp.Request) {
		content, _ := assets.ReadFile("assets/dashboard.html")
		w.Write(content)
	})

	srv.HandleFunc("/report", func(w nethttp.ResponseWriter, r *nethttp.Request) {
		content, _ := assets.ReadFile("assets/report/index.html")
		w.Write(content)
	})

	return srv
}
