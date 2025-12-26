package config

import (
	"os"

	"gopkg.in/yaml.v3"
)

// Config 项目配置结构体
type Config struct {
	LLM          LLMConfig         `yaml:"llm"`
	TavilyAPIKey string            `yaml:"tavily_api_key"` // Deprecated: use Search.Tavily.APIKey
	Search       SearchConfig      `yaml:"search"`
	UserPersona  string            `yaml:"user_persona"`
	Domains      []string          `yaml:"domains"`
	Log          LogConfig         `yaml:"log"`
	Concurrency  ConcurrencyConfig `yaml:"concurrency"`
	DB           DBConfig          `yaml:"db"`
}

// LLMConfig LLM 相关配置
type LLMConfig struct {
	BaseURL string `yaml:"base_url"`
	APIKey  string `yaml:"api_key"`
	Model   string `yaml:"model"`
}

// DBConfig 数据库相关配置
type DBConfig struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
	Name     string `yaml:"name"`
}

// SearchConfig 搜索相关配置
type SearchConfig struct {
	Provider string        `yaml:"provider"`
	Tavily   TavilyConfig  `yaml:"tavily"`
	SearXNG  SearXNGConfig `yaml:"searxng"`
}

// TavilyConfig Tavily 配置
type TavilyConfig struct {
	APIKey string `yaml:"api_key"`
}

// SearXNGConfig SearXNG 配置
type SearXNGConfig struct {
	BaseURL string `yaml:"base_url"`
	Timeout int    `yaml:"timeout"`
}

// LogConfig 日志相关配置
type LogConfig struct {
	Level string `yaml:"level"`
	File  string `yaml:"file"`
}

// ConcurrencyConfig 并发控制配置
type ConcurrencyConfig struct {
	QPS int `yaml:"qps"`
	RPM int `yaml:"rpm"`
}

// LoadConfig 从指定路径加载配置
func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}
