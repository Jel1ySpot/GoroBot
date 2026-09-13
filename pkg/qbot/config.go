package qbot

import (
	_ "embed"
	"fmt"
	"os"
	"path"
	"strings"

	"github.com/Jel1ySpot/GoroBot/pkg/util"
)

const (
	DefaultConfigPath = "conf/qbot/"
)

//go:embed example_config.yaml
var ExampleConfig []byte

type WebhookConfig struct {
	Host    string `yaml:"host" json:"host"`
	Port    uint   `yaml:"port" json:"port"`
	Path    string `yaml:"path" json:"path"`
	BaseURL string `yaml:"base_url" json:"base_url"`
	TLS     struct {
		CertPath string `yaml:"cert_path" json:"cert_path"`
		KeyPath  string `yaml:"key_path" json:"key_path"`
	} `yaml:"tls" json:"tls"`
}

type Config struct {
	Debug       bool          `yaml:"debug" json:"debug"`
	Mode        string        `yaml:"mode" json:"mode"` // "websocket" (默认), "webhook", 或 "both"
	Intents     int           `yaml:"intents" json:"intents"`
	Credentials Credentials   `yaml:"api" json:"api"`
	Webhook     WebhookConfig `yaml:"webhook" json:"webhook"`
	Http        WebhookConfig `yaml:"http" json:"http"` // 兼容旧版 http 命名配置
}

// GetWebhookConfig 获取 Webhook / HTTP 配置（优先读取 webhook，未配置时读取 http）
func (c *Config) GetWebhookConfig() *WebhookConfig {
	if c.Webhook.Port > 0 || c.Webhook.Host != "" || c.Webhook.Path != "" {
		return &c.Webhook
	}
	if c.Http.Port > 0 || c.Http.Host != "" || c.Http.Path != "" {
		return &c.Http
	}
	return &c.Webhook
}

// IsConfigured 检查是否已配置有效的 AppID 与 Secret
func (c *Config) IsConfigured() bool {
	appID := strings.TrimSpace(c.Credentials.AppID)
	secret := strings.TrimSpace(c.Credentials.ClientSecret())
	if appID == "" || appID == "your app id" {
		return false
	}
	if secret == "" || secret == "your secret key" {
		return false
	}
	return true
}

func (s *Service) initConfig() error {
	c := s.conic
	configFile := path.Join(s.configPath, "config.yaml")
	c.SetConfigFile(configFile)
	c.WatchConfig()
	c.BindRef("", &s.config)
	if s.logger != nil {
		c.SetLogger(s.logger.Debug)
	}

	if !util.FileExists(configFile) {
		if err := util.MkdirIfNotExists(s.configPath); err != nil {
			return err
		}
		if err := os.WriteFile(configFile, ExampleConfig, 0644); err != nil {
			return fmt.Errorf("failed to create config file: %v", err)
		}
		s.logInfo("QBot 默认配置文件已生成: %s", configFile)
	}

	if err := c.ReadConfig(); err != nil {
		return err
	}

	return nil
}
