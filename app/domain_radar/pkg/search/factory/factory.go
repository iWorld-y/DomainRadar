package factory

import (
	"fmt"

	"github.com/iWorld-y/domain_radar/app/domain_radar/pkg/config"
	"github.com/iWorld-y/domain_radar/app/domain_radar/pkg/search"
	"github.com/iWorld-y/domain_radar/app/domain_radar/pkg/searxng"
	"github.com/iWorld-y/domain_radar/app/domain_radar/pkg/tavily"
)

// NewSearcher 根据配置创建搜索实例
func NewSearcher(cfg *config.Config) (search.Searcher, error) {
	provider := cfg.Search.Provider
	if provider == "" {
		// 默认回退逻辑：如果有 tavily key，则使用 tavily
		if cfg.TavilyAPIKey != "" || cfg.Search.Tavily.APIKey != "" {
			provider = "tavily"
		} else {
			return nil, fmt.Errorf("search provider not configured")
		}
	}

	switch provider {
	case "tavily":
		apiKey := cfg.Search.Tavily.APIKey
		if apiKey == "" {
			apiKey = cfg.TavilyAPIKey // 兼容旧配置
		}
		if apiKey == "" {
			return nil, fmt.Errorf("tavily api key is missing")
		}
		return tavily.NewClient(apiKey), nil

	case "searxng":
		baseURL := cfg.Search.SearXNG.BaseURL
		if baseURL == "" {
			return nil, fmt.Errorf("searxng base url is missing")
		}
		return searxng.NewClient(baseURL, cfg.Search.SearXNG.Timeout), nil

	default:
		return nil, fmt.Errorf("unknown search provider: %s", provider)
	}
}
