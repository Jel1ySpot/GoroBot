package lagrange

import (
	"fmt"
	"github.com/Jel1ySpot/GoroBot/pkg/util"
	"path"
)

type Account struct {
	Uin      uint32 `json:"uin"`
	Password string `json:"password"`
	SigPath  string `json:"sig_path"`
}

type Config struct {
	AppInfo            string  `json:"app_info"`
	SignServerUrl      string  `json:"sign_server_url"`
	MusicSignServerUrl string  `json:"music_sign_server_url"`
	CommandPrefix      string  `json:"command_prefix"`
	Account            Account `json:"account"`
	IgnoreSelf         bool    `json:"ignore_self"`
}

func (s *Service) InitConic() error {
	c := s.conic
	c.SetConfigFile(path.Join(s.ConfigPath, "config.json"))
	c.WatchConfig()
	c.BindRef("", &s.config)
	c.SetLogger(s.logger.Debug)

	if !util.FileExists(path.Join(s.ConfigPath, "config.json")) {
		if err := util.MkdirIfNotExists(s.ConfigPath); err != nil {
			return err
		}
		if err := c.WriteConfig(); err != nil {
			return err
		}
		s.logger.Failed("Lagrange config file created.")
		return fmt.Errorf("lagrange config file not exist")
	}

	if err := c.ReadConfig(); err != nil {
		return err
	}

	return nil
}
