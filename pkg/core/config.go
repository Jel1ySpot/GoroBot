package GoroBot

import "github.com/Jel1ySpot/GoroBot/pkg/core/logger"

type Config struct {
	Owner        map[string]string `json:"owner"`
	LogLevel     logger.LogLevel   `json:"log_level"`
	ResourcePath string            `json:"resource_path"`
}
