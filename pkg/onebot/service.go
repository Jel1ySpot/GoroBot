package onebot

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	GoroBot "github.com/Jel1ySpot/GoroBot/pkg/core"
	botc "github.com/Jel1ySpot/GoroBot/pkg/core/bot_context"
	"github.com/Jel1ySpot/GoroBot/pkg/core/logger"
	"github.com/Jel1ySpot/conic"
	"github.com/gorilla/websocket"
)

const (
	DefaultConfigPath = "conf/onebot/"
	CacheUpdateInterval = 5 * time.Minute // Cache refresh interval
)

// Cache structures
type Cache struct {
	friendList       map[int64]Friend
	groupList        map[int64]Group
	friendListMu     sync.RWMutex
	groupListMu      sync.RWMutex
	lastFriendUpdate time.Time
	lastGroupUpdate  time.Time
}

type Service struct {
	config     Config
	configPath string
	conic      *conic.Conic

	// HTTP client for OneBot API calls
	httpClient *http.Client

	// WebSocket connections
	wsDialer  *websocket.Dialer
	apiConn   *websocket.Conn
	eventConn *websocket.Conn

	// Context and cancellation
	ctx       context.Context
	ctxCancel context.CancelFunc

	grb    *GoroBot.Instant
	status botc.LoginStatus
	logger logger.Inst

	// Bot information
	selfID   string
	nickname string

	// Cache
	cache Cache
}

func Create() *Service {
	return &Service{
		configPath: DefaultConfigPath,
		conic:      conic.New(),
		status:     botc.Offline,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		wsDialer: &websocket.Dialer{
			HandshakeTimeout: 10 * time.Second,
		},
		cache: Cache{
			friendList: make(map[int64]Friend),
			groupList:  make(map[int64]Group),
		},
	}
}

func (s *Service) Name() string {
	return "OneBot-adapter"
}

func (s *Service) Init(grb *GoroBot.Instant) error {
	s.grb = grb
	s.logger = grb.GetLogger()
	s.ctx, s.ctxCancel = context.WithCancel(context.Background())

	s.logger.Info("Initializing OneBot adapter...")

	if err := s.initConfig(); err != nil {
		s.logger.Fatal("OneBot configuration failed: %v", err)
		return err
	}

	// Initialize connection based on configuration
	s.logger.Info("Connecting to OneBot service (mode: %s)...", s.config.Mode)
	if err := s.connect(); err != nil {
		s.logger.Error("OneBot connection failed: %v", err)
		return fmt.Errorf("failed to connect to OneBot: %v", err)
	}

	// Initialize cache
	s.logger.Info("Initializing OneBot caches...")
	if err := s.initializeCache(); err != nil {
		s.logger.Warning("Failed to initialize cache: %v", err)
		// Don't fail initialization if cache fails, just log warning
	}

	// Register event handlers
	s.registerEventHandlers()

	// Start connection monitoring
	go s.connectionMonitor()

	// Start cache refresh routine
	go s.cacheRefreshRoutine()

	s.status = botc.Online
	grb.AddContext(s.getContext())

	s.logger.Success("OneBot adapter initialized successfully")
	return nil
}

func (s *Service) Release(grb *GoroBot.Instant) error {
	s.logger.Info("Releasing OneBot adapter...")
	s.status = botc.Offline
	s.ctxCancel()

	if s.apiConn != nil {
		s.logger.Debug("Closing API WebSocket connection")
		s.apiConn.Close()
	}
	if s.eventConn != nil {
		s.logger.Debug("Closing Event WebSocket connection")
		s.eventConn.Close()
	}

	s.logger.Success("OneBot adapter released successfully")
	return nil
}

func (s *Service) getContext() *Context {
	return &Context{service: s}
}

func (s *Service) connect() error {
	switch s.config.Mode {
	case "http":
		return s.connectHTTP()
	case "ws":
		return s.connectWebSocket()
	case "ws_reverse":
		return s.connectReverseWebSocket()
	default:
		return fmt.Errorf("unsupported communication mode: %s (supported: http, http_post, ws, ws_reverse)", s.config.Mode)
	}
}

func (s *Service) connectHTTP() error {
	// For HTTP mode, we just need to verify the connection
	s.logger.Debug("Testing HTTP connection to OneBot...")
	loginInfo, err := s.getLoginInfo()
	if err != nil {
		return fmt.Errorf("failed to connect via HTTP: %v", err)
	}

	s.selfID = fmt.Sprintf("%d", loginInfo.UserID)
	s.nickname = loginInfo.Nickname
	s.logger.Success("Connected to OneBot via HTTP, bot ID: %s, nickname: %s", s.selfID, s.nickname)

	s.logger.Debug("Starting HTTP POST server for OneBot events...")
	return s.startHTTPServer()
}

func (s *Service) connectWebSocket() error {
	// Connect to OneBot WebSocket server
	s.logger.Debug("Connecting to OneBot WebSocket server...")
	return s.connectToWebSocketServer()
}

func (s *Service) connectReverseWebSocket() error {
	// For reverse WebSocket, OneBot will connect to us
	s.logger.Debug("Starting reverse WebSocket server for OneBot...")
	return s.startWebSocketServer()
}

func (s *Service) connectionMonitor() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	s.logger.Debug("Connection monitor started")

	for {
		select {
		case <-s.ctx.Done():
			s.logger.Debug("Connection monitor stopped")
			return
		case <-ticker.C:
			if s.status == botc.Online {
				// Ping to check connection health
				if err := s.ping(); err != nil {
					s.logger.Warning("Connection health check failed: %v", err)
					// Attempt to reconnect
					s.logger.Info("Attempting to reconnect...")
					if err := s.reconnect(); err != nil {
						s.logger.Error("Failed to reconnect: %v", err)
						s.status = botc.Offline
					} else {
						s.logger.Success("Reconnected successfully")
						s.status = botc.Online
					}
				} else {
					s.logger.Debug("Connection health check passed")
				}
			}
		}
	}
}

func (s *Service) ping() error {
	// Implementation depends on the communication mode
	switch s.config.Mode {
	case "http":
		_, err := s.getStatus()
		return err
	case "ws", "ws_reverse":
		// Send ping frame for WebSocket connections
		if s.apiConn != nil {
			return s.apiConn.WriteMessage(websocket.PingMessage, nil)
		}
		return nil
	default:
		return nil
	}
}

func (s *Service) reconnect() error {
	s.logger.Info("Attempting to reconnect to OneBot...")

	// Close existing connections
	if s.apiConn != nil {
		s.apiConn.Close()
		s.apiConn = nil
	}
	if s.eventConn != nil {
		s.eventConn.Close()
		s.eventConn = nil
	}

	// Attempt to reconnect
	return s.connect()
}

// Cache management methods

func (s *Service) initializeCache() error {
	// Initialize friend list cache
	if err := s.refreshFriendCache(); err != nil {
		s.logger.Warning("Failed to initialize friend cache: %v", err)
	}

	// Initialize group list cache
	if err := s.refreshGroupCache(); err != nil {
		s.logger.Warning("Failed to initialize group cache: %v", err)
	}

	s.logger.Success("OneBot caches initialized")
	return nil
}

func (s *Service) refreshFriendCache() error {
	s.logger.Debug("Refreshing friend list cache...")

	friends, err := s.fetchFriendList()
	if err != nil {
		return err
	}

	// Convert slice to map
	friendMap := make(map[int64]Friend)
	for _, friend := range friends {
		friendMap[friend.UserID] = friend
	}

	s.cache.friendListMu.Lock()
	s.cache.friendList = friendMap
	s.cache.lastFriendUpdate = time.Now()
	s.cache.friendListMu.Unlock()

	s.logger.Debug("Friend list cache updated (%d friends)", len(friends))
	return nil
}

func (s *Service) refreshGroupCache() error {
	s.logger.Debug("Refreshing group list cache...")

	groups, err := s.fetchGroupList()
	if err != nil {
		return err
	}

	// Convert slice to map
	groupMap := make(map[int64]Group)
	for _, group := range groups {
		groupMap[group.GroupID] = group
	}

	s.cache.groupListMu.Lock()
	s.cache.groupList = groupMap
	s.cache.lastGroupUpdate = time.Now()
	s.cache.groupListMu.Unlock()

	s.logger.Debug("Group list cache updated (%d groups)", len(groups))
	return nil
}

func (s *Service) cacheRefreshRoutine() {
	ticker := time.NewTicker(CacheUpdateInterval)
	defer ticker.Stop()

	s.logger.Debug("Cache refresh routine started (interval: %v)", CacheUpdateInterval)

	for {
		select {
		case <-s.ctx.Done():
			s.logger.Debug("Cache refresh routine stopped")
			return
		case <-ticker.C:
			if s.status == botc.Online {
				// Refresh friend cache if it's getting old
				if time.Since(s.cache.lastFriendUpdate) > CacheUpdateInterval {
					if err := s.refreshFriendCache(); err != nil {
						s.logger.Warning("Failed to refresh friend cache: %v", err)
					}
				}

				// Refresh group cache if it's getting old
				if time.Since(s.cache.lastGroupUpdate) > CacheUpdateInterval {
					if err := s.refreshGroupCache(); err != nil {
						s.logger.Warning("Failed to refresh group cache: %v", err)
					}
				}
			}
		}
	}
}

// invalidateCache clears all cached data
func (s *Service) invalidateCache() {
	s.cache.friendListMu.Lock()
	s.cache.friendList = make(map[int64]Friend)
	s.cache.lastFriendUpdate = time.Time{}
	s.cache.friendListMu.Unlock()

	s.cache.groupListMu.Lock()
	s.cache.groupList = make(map[int64]Group)
	s.cache.lastGroupUpdate = time.Time{}
	s.cache.groupListMu.Unlock()

	s.logger.Debug("OneBot caches invalidated")
}

// invalidateFriendCache clears only friend list cache
func (s *Service) invalidateFriendCache() {
	s.cache.friendListMu.Lock()
	s.cache.friendList = make(map[int64]Friend)
	s.cache.lastFriendUpdate = time.Time{}
	s.cache.friendListMu.Unlock()

	s.logger.Debug("Friend list cache invalidated")
}

// invalidateGroupCache clears only group list cache
func (s *Service) invalidateGroupCache() {
	s.cache.groupListMu.Lock()
	s.cache.groupList = make(map[int64]Group)
	s.cache.lastGroupUpdate = time.Time{}
	s.cache.groupListMu.Unlock()

	s.logger.Debug("Group list cache invalidated")
}

// forceCacheRefresh immediately refreshes all caches
func (s *Service) forceCacheRefresh() error {
	s.logger.Info("Force refreshing all OneBot caches...")

	if err := s.refreshFriendCache(); err != nil {
		s.logger.Error("Failed to refresh friend cache: %v", err)
	}

	if err := s.refreshGroupCache(); err != nil {
		s.logger.Error("Failed to refresh group cache: %v", err)
	}

	s.logger.Success("All OneBot caches refreshed")
	return nil
}
