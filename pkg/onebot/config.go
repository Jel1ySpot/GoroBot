package onebot

import (
	"fmt"
	"path"

	"github.com/Jel1ySpot/GoroBot/pkg/util"
)

type Config struct {
	// Communication mode: "http", "http_post", "ws", "ws_reverse"
	Mode string `json:"mode"`

	// HTTP configuration
	HTTP struct {
		Host        string `json:"host"`
		Port        int    `json:"port"`
		AccessToken string `json:"access_token"`
		PostURL     string `json:"post_url"`
		Secret      string `json:"secret"`
		Timeout     int    `json:"timeout"` // seconds
	} `json:"http"`

	// WebSocket configuration
	WebSocket struct {
		Host        string `json:"host"`
		Port        int    `json:"port"`
		AccessToken string `json:"access_token"`
		Path        string `json:"path"`
	} `json:"ws"`

	// Reverse WebSocket configuration
	ReverseWebSocket struct {
		Host              string `json:"host"`
		Port              int    `json:"port"`
		AccessToken       string `json:"access_token"`
		APIPath           string `json:"api_path,omitempty"`
		EventPath         string `json:"event_path,omitempty"`
		UniversalPath     string `json:"universal_path"`
		UseUniversal      bool   `json:"use_universal"`
		ReconnectInterval int    `json:"reconnect_interval"` // milliseconds
	} `json:"ws_reverse"`

	// Message format: "string" or "array"
	MessageFormat string `json:"message_format"`

	// Whether to enable heartbeat
	Heartbeat struct {
		Enable   bool `json:"enable"`
		Interval int  `json:"interval"` // milliseconds
	} `json:"heartbeat"`

	// API rate limiting
	RateLimit struct {
		Enable   bool `json:"enable"`
		Interval int  `json:"interval"` // milliseconds
	} `json:"rate_limit"`

	// Bot behavior settings
	IgnoreSelf    bool   `json:"ignore_self"`
	Debug         bool   `json:"debug"`
	CommandPrefix string `json:"command_prefix"`
}

var defaultConfig = Config{
	Mode: "http",
	HTTP: struct {
		Host        string `json:"host"`
		Port        int    `json:"port"`
		AccessToken string `json:"access_token"`
		PostURL     string `json:"post_url"`
		Secret      string `json:"secret"`
		Timeout     int    `json:"timeout"`
	}{
		Host:        "127.0.0.1",
		Port:        8080,
		AccessToken: "",
		PostURL:     "http://127.0.0.1:3000",
		Secret:      "",
		Timeout:     30,
	},
	WebSocket: struct {
		Host        string `json:"host"`
		Port        int    `json:"port"`
		AccessToken string `json:"access_token"`
		Path        string `json:"path"`
	}{
		Host:        "127.0.0.1",
		Port:        3001,
		AccessToken: "",
		Path:        "/",
	},
	ReverseWebSocket: struct {
		Host              string `json:"host"`
		Port              int    `json:"port"`
		AccessToken       string `json:"access_token"`
		APIPath           string `json:"api_path,omitempty"`
		EventPath         string `json:"event_path,omitempty"`
		UniversalPath     string `json:"universal_path"`
		UseUniversal      bool   `json:"use_universal"`
		ReconnectInterval int    `json:"reconnect_interval"`
	}{
		Host:              "127.0.0.1",
		Port:              8082,
		AccessToken:       "",
		UniversalPath:     "/",
		UseUniversal:      true,
		ReconnectInterval: 30000,
	},
	MessageFormat: "array",
	Heartbeat: struct {
		Enable   bool `json:"enable"`
		Interval int  `json:"interval"`
	}{
		Enable:   false,
		Interval: 30000,
	},
	RateLimit: struct {
		Enable   bool `json:"enable"`
		Interval int  `json:"interval"`
	}{
		Enable:   false,
		Interval: 500,
	},
	IgnoreSelf:    true,
	Debug:         false,
	CommandPrefix: "/",
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

		// Set default config
		s.config = defaultConfig

		if err := c.WriteConfig(); err != nil {
			return fmt.Errorf("failed to write default config: %v", err)
		}

		s.logger.Warning("OneBot config file created at %s with default settings", configPath)
		s.logger.Warning("Please configure the OneBot connection settings in %s and restart", configPath)
		s.logger.Info("Available modes: http, http_post, ws, ws_reverse")
		s.logger.Info("Example HTTP mode: set host and port for OneBot server")
		s.logger.Info("Example WebSocket: set host and port for OneBot WebSocket server")

		// Don't return error immediately, let user configure first
		return fmt.Errorf("OneBot configuration required - please edit %s and restart", configPath)
	}

	if err := c.ReadConfig(); err != nil {
		return fmt.Errorf("failed to read OneBot config from %s: %v", configPath, err)
	}

	// Validate configuration
	if err := s.validateConfig(); err != nil {
		s.logger.Error("OneBot configuration validation failed: %v", err)
		s.logger.Info("Please check your configuration in %s", configPath)
		s.logger.Info("Available communication modes: http, http_post, ws, ws_reverse")
		s.logger.Info("Available message formats: string, array")
		return fmt.Errorf("OneBot configuration invalid: %v", err)
	}

	s.logger.Success("OneBot configuration loaded successfully (mode: %s)", s.config.Mode)
	return nil
}

func (s *Service) validateConfig() error {
	// Validate communication mode
	validModes := []string{"http", "http_post", "ws", "ws_reverse"}
	isValidMode := false
	for _, mode := range validModes {
		if s.config.Mode == mode {
			isValidMode = true
			break
		}
	}
	if !isValidMode {
		return fmt.Errorf("unsupported communication mode: %s (supported: %v)", s.config.Mode, validModes)
	}

	// Validate mode-specific configuration
	switch s.config.Mode {
	case "http":
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
		if s.config.WebSocket.Host == "" {
			return fmt.Errorf("WebSocket host is required for ws mode")
		}
		if s.config.WebSocket.Port <= 0 || s.config.WebSocket.Port > 65535 {
			return fmt.Errorf("invalid WebSocket port: %d (must be 1-65535)", s.config.WebSocket.Port)
		}
		if s.config.WebSocket.Path == "" {
			s.config.WebSocket.Path = "/" // Set default path
		}
	case "ws_reverse":
		if s.config.ReverseWebSocket.Host == "" {
			return fmt.Errorf("reverse WebSocket host is required for ws_reverse mode")
		}
		if s.config.ReverseWebSocket.Port <= 0 || s.config.ReverseWebSocket.Port > 65535 {
			return fmt.Errorf("invalid reverse WebSocket port: %d (must be 1-65535)", s.config.ReverseWebSocket.Port)
		}
		if s.config.ReverseWebSocket.ReconnectInterval <= 0 {
			s.config.ReverseWebSocket.ReconnectInterval = 3000 // Set default reconnect interval
		}
	}

	// Validate message format
	if s.config.MessageFormat != "string" && s.config.MessageFormat != "array" {
		return fmt.Errorf("invalid message format: %s (supported: string, array)", s.config.MessageFormat)
	}

	// Validate heartbeat configuration
	if s.config.Heartbeat.Enable && s.config.Heartbeat.Interval <= 0 {
		s.config.Heartbeat.Interval = 15000 // Set default heartbeat interval
	}

	// Validate rate limit configuration
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
