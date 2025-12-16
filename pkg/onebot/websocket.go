package onebot

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gorilla/websocket"
)

// WebSocket server for reverse WebSocket mode
func (s *Service) startWebSocketServer() error {
	upgrader := websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true // Allow all origins for simplicity
		},
	}

	mux := http.NewServeMux()

	// Handle API connections
	mux.HandleFunc(s.config.ReverseWebSocket.Path, func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			s.logger.Error("Failed to upgrade WebSocket: %v", err)
			return
		}
		s.apiConn = conn
		s.eventConn = conn
		s.handleUniversalWebSocket(conn)
	})

	server := &http.Server{
		Addr:    fmt.Sprintf("%s:%d", s.config.ReverseWebSocket.Host, s.config.ReverseWebSocket.Port),
		Handler: mux,
	}

	go func() {
		s.logger.Info("Starting WebSocket server on %s", server.Addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			s.logger.Error("WebSocket server error: %v", err)
		}
	}()

	return nil
}

// Connect to OneBot WebSocket server (forward WebSocket mode)
func (s *Service) connectToWebSocketServer() error {
	host := s.config.WebSocket.Host
	port := s.config.WebSocket.Port

	// Connect to API endpoint
	apiURL := fmt.Sprintf("ws://%s:%d/api", host, port)
	s.logger.Info("Connecting to OneBot WebSocket API: %s", apiURL)

	headers := make(http.Header)
	if s.config.WebSocket.AccessToken != "" {
		headers.Set("Authorization", "Bearer "+s.config.WebSocket.AccessToken)
	}

	apiConn, resp, err := s.wsDialer.Dial(apiURL, headers)
	if err != nil {
		if resp != nil {
			return fmt.Errorf("failed to connect to API WebSocket (status: %d): %v", resp.StatusCode, err)
		}
		return fmt.Errorf("failed to connect to API WebSocket: %v", err)
	}
	s.apiConn = apiConn
	s.logger.Success("Connected to OneBot API WebSocket")

	// Connect to Event endpoint
	eventURL := fmt.Sprintf("ws://%s:%d/event", host, port)
	s.logger.Info("Connecting to OneBot WebSocket Event: %s", eventURL)

	eventConn, resp, err := s.wsDialer.Dial(eventURL, headers)
	if err != nil {
		apiConn.Close()
		if resp != nil {
			return fmt.Errorf("failed to connect to Event WebSocket (status: %d): %v", resp.StatusCode, err)
		}
		return fmt.Errorf("failed to connect to Event WebSocket: %v", err)
	}
	s.eventConn = eventConn
	s.logger.Success("Connected to OneBot Event WebSocket")

	// Start event handler
	go s.handleEventWebSocket(eventConn)

	// Get bot information
	s.logger.Debug("Retrieving bot information...")
	loginInfo, err := s.getLoginInfo()
	if err != nil {
		return fmt.Errorf("failed to get login info: %v", err)
	}

	s.selfID = loginInfo.UserID
	s.nickname = loginInfo.Nickname
	s.logger.Success("Bot information retrieved - ID: %d, nickname: %s", s.selfID, s.nickname)

	return nil
}

// WebSocket event handlers
func (s *Service) handleAPIWebSocket(conn *websocket.Conn) {
	defer conn.Close()

	s.logger.Info("API WebSocket connection established")

	for {
		select {
		case <-s.ctx.Done():
			return
		default:
			// Handle API requests/responses
			// This is primarily used for sending API requests in reverse WebSocket mode
			var message map[string]interface{}
			if err := conn.ReadJSON(&message); err != nil {
				if websocket.IsCloseError(err, websocket.CloseGoingAway, websocket.CloseNormalClosure) {
					s.logger.Info("API WebSocket connection closed")
					return
				}
				s.logger.Error("API WebSocket read error: %v", err)
				return
			}
			// Process API request if needed
		}
	}
}

func (s *Service) handleEventWebSocket(conn *websocket.Conn) {
	defer conn.Close()

	s.logger.Info("Event WebSocket connection established")

	for {
		select {
		case <-s.ctx.Done():
			return
		default:
			// Read event message
			_, message, err := conn.ReadMessage()
			if err != nil {
				if websocket.IsCloseError(err, websocket.CloseGoingAway, websocket.CloseNormalClosure) {
					s.logger.Info("Event WebSocket connection closed")
					return
				}
				s.logger.Error("Event WebSocket read error: %v", err)
				return
			}

			// Process event
			if err := s.processEvent(message); err != nil {
				s.logger.Error("Failed to process WebSocket event: %v", err)
			}
		}
	}
}

func (s *Service) handleUniversalWebSocket(conn *websocket.Conn) {
	defer conn.Close()

	s.logger.Info("Universal WebSocket connection established")

	for {
		select {
		case <-s.ctx.Done():
			return
		default:
			// Read message
			_, message, err := conn.ReadMessage()
			if err != nil {
				if websocket.IsCloseError(err, websocket.CloseGoingAway, websocket.CloseNormalClosure) {
					s.logger.Info("Universal WebSocket connection closed")
					return
				}
				s.logger.Error("Universal WebSocket read error: %v", err)
				return
			}

			// Try to determine if this is an event or API response
			var parsed map[string]interface{}
			if err := json.Unmarshal(message, &parsed); err != nil {
				s.logger.Error("Failed to parse WebSocket message: %v", err)
				continue
			}

			// Check if it's an API response (has 'echo' field) or an event (has 'post_type')
			if _, hasEcho := parsed["echo"]; hasEcho {
				// This is an API response, handle it accordingly
				// For now, we'll just log it
				s.logger.Debug("Received API response")
			} else if _, hasPostType := parsed["post_type"]; hasPostType {
				// This is an event
				if err := s.processEvent(message); err != nil {
					s.logger.Error("Failed to process WebSocket event: %v", err)
				}
			}
		}
	}
}
