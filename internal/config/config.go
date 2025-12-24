package config

import (
	"os"

	"gopkg.in/yaml.v3"
)

// Config 项目配置结构体
type Config struct {
	LLM         LLMConfig         `yaml:"llm"`
	RSSLinks    []string          `yaml:"rss_links"`
	Log         LogConfig         `yaml:"log"`
	Concurrency ConcurrencyConfig `yaml:"concurrency"`
}

// LLMConfig LLM 相关配置
type LLMConfig struct {
	BaseURL string `yaml:"base_url"`
	APIKey  string `yaml:"api_key"`
	Model   string `yaml:"model"`
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
