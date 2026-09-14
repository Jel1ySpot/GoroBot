package qbot

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// Gateway OpCodes
const (
	GatewayOpDispatch       = 0
	GatewayOpHeartbeat      = 1
	GatewayOpIdentify       = 2
	GatewayOpResume         = 6
	GatewayOpReconnect      = 7
	GatewayOpInvalidSession = 9
	GatewayOpHello          = 10
	GatewayOpHeartbeatAck   = 11
)

// Gateway Intents
const (
	IntentGuilds              = 1 << 0
	IntentGuildMembers         = 1 << 1
	IntentDirectMessage       = 1 << 12
	IntentGroupAndC2C         = 1 << 25
	IntentInteraction         = 1 << 26
	IntentPublicGuildMessages = 1 << 30

	FullIntents = IntentGuilds | IntentGuildMembers | IntentDirectMessage | IntentGroupAndC2C | IntentInteraction | IntentPublicGuildMessages
)

// GatewayConnection WebSocket 网关连接
type GatewayConnection struct {
	service   *Service
	wsConn    *websocket.Conn
	sessionID string
	lastSeq   *int
	mu        sync.Mutex
	isClosed  bool
}

// NewGatewayConnection 创建网关连接
func NewGatewayConnection(s *Service) *GatewayConnection {
	return &GatewayConnection{
		service: s,
	}
}

// Start 启动网关连接循环
func (g *GatewayConnection) Start(ctx context.Context) {
	go func() {
		delay := time.Second
		for {
			select {
			case <-ctx.Done():
				return
			default:
			}

			err := g.connect(ctx)
			if err != nil && ctx.Err() == nil {
				g.service.logWarning("QBot WebSocket gateway disconnected: %v, reconnecting in %v...", err, delay)
			}

			select {
			case <-ctx.Done():
				return
			case <-time.After(delay):
			}

			delay = delay * 2
			if delay > 60*time.Second {
				delay = 60 * time.Second
			}
		}
	}()
}

func (g *GatewayConnection) connect(ctx context.Context) error {
	gatewayURL, err := g.service.api.GetGatewayURL(ctx)
	if err != nil {
		return fmt.Errorf("get gateway url: %w", err)
	}

	g.service.logDebug("QBot 正在连接 WebSocket Gateway: %s", gatewayURL)
	dialer := websocket.Dialer{
		HandshakeTimeout: 10 * time.Second,
	}

	header := make(http.Header)
	header.Set("User-Agent", "GoroBot-QBot/2.0")

	conn, _, err := dialer.DialContext(ctx, gatewayURL, header)
	if err != nil {
		return fmt.Errorf("dial gateway: %w", err)
	}

	g.mu.Lock()
	g.wsConn = conn
	g.isClosed = false
	g.mu.Unlock()

	defer func() {
		g.mu.Lock()
		g.isClosed = true
		_ = conn.Close()
		g.mu.Unlock()
	}()

	g.service.logInfo("QBot connected to WebSocket gateway")

	heartbeatStop := make(chan struct{})
	defer close(heartbeatStop)

	for {
		select {
		case <-ctx.Done():
			return nil
		default:
		}

		_, msgBytes, err := conn.ReadMessage()
		if err != nil {
			return err
		}

		var payload WSPayload
		if err := json.Unmarshal(msgBytes, &payload); err != nil {
			continue
		}

		if payload.S != nil {
			g.mu.Lock()
			g.lastSeq = payload.S
			g.mu.Unlock()
		}

		seqInfo := "none"
		if payload.S != nil {
			seqInfo = fmt.Sprintf("%d", *payload.S)
		}
		g.service.logDebug("QBot Gateway <<< op: %d, t: %s, s: %s", payload.Op, payload.T, seqInfo)

		switch payload.Op {
		case GatewayOpHello:
			var helloData struct {
				HeartbeatInterval int `json:"heartbeat_interval"`
			}
			_ = json.Unmarshal(payload.D, &helloData)
			interval := helloData.HeartbeatInterval
			if interval <= 0 {
				interval = 30000
			}

			// 启动心跳协程
			go g.heartbeatLoop(ctx, time.Duration(interval)*time.Millisecond, heartbeatStop)

			// 发送鉴权或恢复
			if err := g.sendIdentifyOrResume(ctx); err != nil {
				return err
			}

		case GatewayOpDispatch:
			if payload.T == "READY" {
				var readyData struct {
					SessionID string `json:"session_id"`
					User      User   `json:"user"`
				}
				if err := json.Unmarshal(payload.D, &readyData); err == nil {
					g.mu.Lock()
					g.sessionID = readyData.SessionID
					g.mu.Unlock()
					g.service.logSuccess("QBot WebSocket gateway READY, session ID: %s", readyData.SessionID)
				}
			}

			// 分发业务事件
			go func(p WSPayload) {
				if err := g.service.handleWebhookDispatch(&p); err != nil {
					g.service.logError("QBot dispatch gateway event failed: %v", err)
				}
			}(payload)

		case GatewayOpHeartbeatAck:
			// 心跳应答

		case GatewayOpReconnect:
			g.service.logInfo("QBot server requested reconnect")
			return fmt.Errorf("server requested reconnect")

		case GatewayOpInvalidSession:
			var canResume bool
			_ = json.Unmarshal(payload.D, &canResume)
			if !canResume {
				g.mu.Lock()
				g.sessionID = ""
				g.lastSeq = nil
				g.mu.Unlock()
			}
			return fmt.Errorf("invalid session (can resume: %v)", canResume)
		}
	}
}

func (g *GatewayConnection) heartbeatLoop(ctx context.Context, interval time.Duration, stop <-chan struct{}) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-stop:
			return
		case <-ticker.C:
			g.mu.Lock()
			if g.isClosed || g.wsConn == nil {
				g.mu.Unlock()
				return
			}
			hb := struct {
				Op int  `json:"op"`
				D  *int `json:"d"`
			}{
				Op: GatewayOpHeartbeat,
				D:  g.lastSeq,
			}
			seqStr := "none"
			if g.lastSeq != nil {
				seqStr = fmt.Sprintf("%d", *g.lastSeq)
			}
			g.service.logDebug("QBot Gateway >>> 发送心跳包 (seq: %s)", seqStr)
			_ = g.wsConn.WriteJSON(hb)
			g.mu.Unlock()
		}
	}
}

func (g *GatewayConnection) sendIdentifyOrResume(ctx context.Context) error {
	token, err := g.service.tokenManager.GetAccessToken(ctx, g.service.config.Credentials.AppID, g.service.config.Credentials.ClientSecret())
	if err != nil {
		return fmt.Errorf("get token for identify: %w", err)
	}

	g.mu.Lock()
	defer g.mu.Unlock()

	if g.sessionID != "" && g.lastSeq != nil {
		g.service.logDebug("QBot Gateway >>> 发送 Resume (sessionId: %s, seq: %d)", g.sessionID, *g.lastSeq)
		resumePayload := struct {
			Op int `json:"op"`
			D  struct {
				Token     string `json:"token"`
				SessionID string `json:"session_id"`
				Seq       int    `json:"seq"`
			} `json:"d"`
		}{
			Op: GatewayOpResume,
			D: struct {
				Token     string `json:"token"`
				SessionID string `json:"session_id"`
				Seq       int    `json:"seq"`
			}{
				Token:     fmt.Sprintf("QQBot %s", token),
				SessionID: g.sessionID,
				Seq:       *g.lastSeq,
			},
		}
		return g.wsConn.WriteJSON(resumePayload)
	}

	intents := g.service.config.Intents
	if intents == 0 {
		intents = FullIntents
	}

	g.service.logDebug("QBot Gateway >>> 发送 Identify (intents: %d)", intents)
	identifyPayload := struct {
		Op int `json:"op"`
		D  struct {
			Token   string `json:"token"`
			Intents int    `json:"intents"`
			Shard   [2]int `json:"shard"`
		} `json:"d"`
	}{
		Op: GatewayOpIdentify,
		D: struct {
			Token   string `json:"token"`
			Intents int    `json:"intents"`
			Shard   [2]int `json:"shard"`
		}{
			Token:   fmt.Sprintf("QQBot %s", token),
			Intents: intents,
			Shard:   [2]int{0, 1},
		},
	}
	return g.wsConn.WriteJSON(identifyPayload)
}
