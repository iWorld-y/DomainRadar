package server

import (
	"github.com/go-kratos/kratos/v2/log"
	"github.com/iWorld-y/domain_radar/app/display/internal/conf"
	"github.com/iWorld-y/domain_radar/app/domain_radar/pkg/config"
	"github.com/iWorld-y/domain_radar/app/domain_radar/pkg/engine"
	drLogger "github.com/iWorld-y/domain_radar/app/domain_radar/pkg/logger"
	"github.com/iWorld-y/domain_radar/app/domain_radar/pkg/storage"
)

// NewRadarEngine 初始化 domain_radar 引擎
func NewRadarEngine(c *conf.Radar, logger log.Logger) (*engine.Engine, func(), error) {
	if c == nil {
		return nil, func() {}, nil
	}

	// 将 internal/conf.Radar 转换为 pkg/config.Config
	drCfg := &config.Config{
		LLM: config.LLMConfig{
			BaseURL: c.Llm.BaseUrl,
			APIKey:  c.Llm.ApiKey,
			Model:   c.Llm.Model,
		},
		Search: config.SearchConfig{
			Provider: c.Search.Provider,
			Tavily: config.TavilyConfig{
				APIKey: c.Search.Tavily.ApiKey,
			},
			SearXNG: config.SearXNGConfig{
				BaseURL: c.Search.Searxng.BaseUrl,
				Timeout: int(c.Search.Searxng.Timeout),
			},
		},
		UserPersona: c.UserPersona,
		Domains:     c.Domains,
		Log: config.LogConfig{
			Level: c.Log.Level,
			File:  c.Log.File,
		},
		Concurrency: config.ConcurrencyConfig{
			QPS: int(c.Concurrency.Qps),
			RPM: int(c.Concurrency.Rpm),
		},
		DB: config.DBConfig{
			Host:     c.Db.Host,
			Port:     int(c.Db.Port),
			User:     c.Db.User,
			Password: c.Db.Password,
			Name:     c.Db.Name,
		},
	}

	// 初始化日志
	if err := drLogger.InitLogger(drCfg.Log.Level, drCfg.Log.File); err != nil {
		log.NewHelper(logger).Errorf("Failed to init domain_radar logger: %v", err)
		_ = drLogger.InitLogger("info", "") // 降级处理
	}

	// 初始化存储层
	store, err := storage.NewStorage(drCfg.DB)
	if err != nil {
		log.NewHelper(logger).Errorf("Failed to init storage for engine: %v", err)
		return nil, nil, err
	}

	// 初始化核心引擎
	eng, err := engine.NewEngine(drCfg, store)
	if err != nil {
		log.NewHelper(logger).Errorf("Failed to init engine: %v", err)
		return nil, nil, err
	}

	cleanup := func() {
		// 如果 engine 有 Close 方法，可以在这里调用
		log.NewHelper(logger).Info("Cleaning up domain_radar engine")
	}

	return eng, cleanup, nil
}
