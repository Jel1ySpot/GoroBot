package GoroBot

import (
	_ "embed"
	"encoding/json"

	"github.com/Jel1ySpot/GoroBot/pkg/core/logger"
)

// StringList 支持从单字符串或字符串切片反序列化
type StringList []string

func (s *StringList) UnmarshalJSON(data []byte) error {
	if len(data) == 0 {
		return nil
	}
	if data[0] == '[' {
		var list []string
		if err := json.Unmarshal(data, &list); err != nil {
			return err
		}
		*s = list
		return nil
	}
	var single string
	if err := json.Unmarshal(data, &single); err != nil {
		return err
	}
	*s = []string{single}
	return nil
}

type Config struct {
	Owner        map[string]string     `json:"owner"`
	Admin        map[string]StringList `json:"admin"`
	LogLevel     logger.LogLevel       `json:"log_level"`
	ResourcePath string                `json:"resource_path"`
}

//go:embed config/default_conf.json
var DefaultConfig []byte
