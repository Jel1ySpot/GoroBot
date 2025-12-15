package onebot

import (
	"fmt"
	"path"

	"github.com/Jel1ySpot/GoroBot/pkg/util"
)

type Config struct {
	// connection mode: "http", "http_post", "ws", "ws_reverse"
	Mode string `json:"mode"`

	// HTTP configuration
	HTTP *struct {
		Host        string `json:"host"`
		Port        int    `json:"port"`
		AccessToken string `json:"access_token"`
		PostURL     string `json:"post_url"`
		Secret      string `json:"secret"`
		Timeout     int    `json:"timeout,omitempty"` // seconds
	} `json:"http,omitempty"`

	// WebSocket configuration
	WebSocket *struct {
		Host        string `json:"host"`
		Port        int    `json:"port"`
		AccessToken string `json:"access_token,omitempty"`
	} `json:"ws,omitempty"`

	// Reverse WebSocket configuration
	ReverseWebSocket *struct {
		Host              string `json:"host"`
		Port              int    `json:"port"`
		AccessToken       string `json:"access_token,omitempty"`
		Path              string `json:"path,omitempty"`
		ReconnectInterval int    `json:"reconnect_interval,omitempty"` // milliseconds
	} `json:"ws_reverse,omitempty"`

	// Message format: "string" or "array"
	MessageFormat string `json:"message_format,omitempty"`

	// Whether to enable heartbeat
	Heartbeat *struct {
		Enable   bool `json:"enable"`
		Interval int  `json:"interval,omitempty"` // milliseconds
	} `json:"heartbeat,omitempty"`

	// API rate limiting
	RateLimit *struct {
		Enable   bool `json:"enable"`
		Interval int  `json:"interval"` // milliseconds
	} `json:"rate_limit,omitempty"`

	// Bot behavior settings
	IgnoreSelf    bool   `json:"ignore_self,omitempty"`
	Debug         bool   `json:"debug,omitempty"`
	CommandPrefix string `json:"command_prefix,omitempty"`
}

var defaultConfig = Config{
	Mode:             "",
	HTTP:             nil,
	WebSocket:        nil,
	ReverseWebSocket: nil,
	MessageFormat:    "",
	Heartbeat:        nil,
	RateLimit:        nil,
	IgnoreSelf:       true,
	Debug:            false,
	CommandPrefix:    "",
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

		s.logger.Warning("OneBot config file created at %s with default settings", configPath)
		s.logger.Warning("Please configure the OneBot connection settings in %s and restart", configPath)
		s.logger.Info("Available modes: http, ws, ws_reverse")

		return fmt.Errorf("OneBot configuration required - please edit %s and restart", configPath)
	}

	if err := c.ReadConfig(); err != nil {
		return fmt.Errorf("failed to read OneBot config from %s: %v", configPath, err)
	}

	// Validate configuration
	if err := s.validateConfig(); err != nil {
		s.logger.Error("OneBot configuration validation failed: %v", err)
		s.logger.Info("Please check your configuration in %s", configPath)
		return fmt.Errorf("OneBot configuration invalid: %v", err)
	}

	s.logger.Success("OneBot configuration loaded successfully (mode: %s)", s.config.Mode)
	return nil
}

func (s *Service) validateConfig() error {
	validModes := []string{"http", "ws", "ws_reverse"}
	isValidMode := false
	for _, mode := range validModes {
		if s.config.Mode == mode {
			isValidMode = true
			break
		}
	}
	if !isValidMode {
		return fmt.Errorf("unsupported connection mode: %s (supported: %v)", s.config.Mode, validModes)
	}

	// Validate mode-specific configuration
	switch s.config.Mode {
	case "http":
		if s.config.HTTP == nil {
			return fmt.Errorf("http configuration required")
		}
		if s.config.HTTP.Host == "" {
			return fmt.Errorf("HTTP host is required for HTTP mode")
		}
		if s.config.HTTP.Port <= 0 || s.config.HTTP.Port > 65535 {
			return fmt.Errorf("invalid HTTP port: %d (must be 1-65535)", s.config.HTTP.Port)
		}
		if s.config.HTTP.PostURL == "" {
			return fmt.Errorf("HTTP post url is required for HTTP mode")
		}
		if s.config.HTTP.Timeout <= 0 {
			s.config.HTTP.Timeout = 30 // Set default timeout
		}
	case "ws":
		if s.config.WebSocket == nil {
			return fmt.Errorf("ws configuration required")
		}
		if s.config.WebSocket.Host == "" {
			return fmt.Errorf("WebSocket host is required for ws mode")
		}
		if s.config.WebSocket.Port <= 0 || s.config.WebSocket.Port > 65535 {
			return fmt.Errorf("invalid WebSocket port: %d (must be 1-65535)", s.config.WebSocket.Port)
		}
	case "ws_reverse":
		if s.config.ReverseWebSocket == nil {
			return fmt.Errorf("ws_reverse configuration required")
		}
		if s.config.ReverseWebSocket.Host == "" {
			return fmt.Errorf("reverse WebSocket host is required for ws_reverse mode")
		}
		if s.config.ReverseWebSocket.Port <= 0 || s.config.ReverseWebSocket.Port > 65535 {
			return fmt.Errorf("invalid reverse WebSocket port: %d (must be 1-65535)", s.config.ReverseWebSocket.Port)
		}
		if s.config.ReverseWebSocket.ReconnectInterval <= 0 {
			s.config.ReverseWebSocket.ReconnectInterval = 300 // Set default reconnect interval
		}
	}

	// Validate message format
	if s.config.MessageFormat == "" {
		s.config.MessageFormat = "array"
	}
	if s.config.MessageFormat != "string" && s.config.MessageFormat != "array" {
		return fmt.Errorf("invalid message format: %s (supported: string, array)", s.config.MessageFormat)
	}

	// Validate heartbeat configuration
	if s.config.Heartbeat == nil {
		s.config.Heartbeat = &struct {
			Enable   bool `json:"enable"`
			Interval int  `json:"interval,omitempty"`
		}{
			Enable:   true,
			Interval: 30000,
		}
	}
	if s.config.Heartbeat.Enable && s.config.Heartbeat.Interval <= 0 {
		s.config.Heartbeat.Interval = 15000 // Set default heartbeat interval
	}

	// Validate rate limit configuration
	if s.config.RateLimit == nil {
		s.config.RateLimit = &struct {
			Enable   bool `json:"enable"`
			Interval int  `json:"interval"`
		}{Enable: false, Interval: 0}
	}
	if s.config.RateLimit.Enable && s.config.RateLimit.Interval <= 0 {
		s.config.RateLimit.Interval = 500 // Set default rate limit interval
	}

	// Validate command prefix
	if s.config.CommandPrefix == "" {
		s.config.CommandPrefix = "/" // Set default command prefix
		s.logger.Debug("Using default command prefix: /")
	} else {
		s.logger.Debug("Using command prefix: %s", s.config.CommandPrefix)
	}

	return nil
}
