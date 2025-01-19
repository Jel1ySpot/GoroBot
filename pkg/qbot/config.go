package qbot

import (
	_ "embed"
	"fmt"
	"github.com/Jel1ySpot/GoroBot/pkg/util"
	"github.com/tencent-connect/botgo/token"
	"os"
	"path"
)

const (
	DefaultConfigPath = "conf/qbot/"
)

//go:embed example_config.yaml
var ExampleConfig []byte

type Config struct {
	Debug       bool                   `yaml:"debug"`
	Credentials token.QQBotCredentials `yaml:"api"`
	Http        struct {
		Host    string `yaml:"host"`
		Port    uint   `yaml:"port"`
		Path    string `yaml:"path"`
		BaseURL string `yaml:"base_url"`
		TLS     struct {
			CertPath string `yaml:"cert_path"`
			KeyPath  string `yaml:"key_path"`
		} `yaml:"tls"`
	} `yaml:"http"`
}

func (s *Service) initConfig() error {
	c := s.conic
	c.SetConfigFile(path.Join(s.configPath, "config.yaml"))
	c.WatchConfig()
	c.BindRef("", &s.config)
	c.SetLogger(s.logger.Debug)

	if !util.FileExists(path.Join(s.configPath, "config.yaml")) {
		if err := util.MkdirIfNotExists(s.configPath); err != nil {
			return err
		}
		if err := os.WriteFile(s.configPath, ExampleConfig, 0644); err != nil {
			return fmt.Errorf("failed to create config file: %v", err)
		}
		s.logger.Warning("QBot config file created.")
		return fmt.Errorf("QBot config file not exist")
	}

	if err := c.ReadConfig(); err != nil {
		return err
	}

	return nil
}
