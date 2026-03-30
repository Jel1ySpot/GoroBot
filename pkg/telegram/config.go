package telegram

import (
	"fmt"
	"path"

	"github.com/Jel1ySpot/GoroBot/pkg/util"
)

const DefaultConfigPath = "conf/telegram/"

type Config struct {
	Token     string `json:"token"`
	ServerURL string `json:"server_url"` // 自定义 API 地址，留空使用默认
}

var defaultConfig = Config{
	Token:     "",
	ServerURL: "",
}

func (s *Service) initConfig() error {
	c := s.conic
	configPath := path.Join(s.configPath, "config.json")
	c.SetConfigFile(configPath)
	c.WatchConfig()
	c.BindRef("", &s.config)
	c.SetLogger(s.logger.Debug)

	if !util.FileExists(configPath) {
		if err := util.MkdirIfNotExists(s.configPath); err != nil {
			return fmt.Errorf("failed to create config directory: %v", err)
		}

		s.config = defaultConfig

		if err := c.WriteConfig(); err != nil {
			return fmt.Errorf("failed to write default config: %v", err)
		}

		s.logger.Warning("Telegram 配置文件已生成于 %s，请填写 bot token 后重启", configPath)
		return fmt.Errorf("telegram config missing token, please configure %s", configPath)
	}

	if err := c.ReadConfig(); err != nil {
		return fmt.Errorf("failed to read Telegram config: %v", err)
	}

	if s.config.Token == "" {
		return fmt.Errorf("telegram bot token is empty")
	}

	return nil
}
