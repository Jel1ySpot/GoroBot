package GoroBot

import (
	_ "embed"
	"github.com/Jel1ySpot/GoroBot/pkg/core/logger"
)

type Config struct {
	Owner        map[string]string `json:"owner"`
	LogLevel     logger.LogLevel   `json:"log_level"`
	ResourcePath string            `json:"resource_path"`
}

//go:embed config/default_conf.json
var DefaultConfig []byte
