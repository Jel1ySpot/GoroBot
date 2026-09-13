package qbot

import (
	"context"
	"fmt"
	"io"
	"net/http"
	urlpkg "net/url"
	"os"
	"path"
	"path/filepath"
	"strings"

	GoroBot "github.com/Jel1ySpot/GoroBot/pkg/core"
	botc "github.com/Jel1ySpot/GoroBot/pkg/core/bot_context"
	"github.com/Jel1ySpot/GoroBot/pkg/core/entity"
	"github.com/Jel1ySpot/GoroBot/pkg/core/logger"
	"github.com/Jel1ySpot/conic"
	"github.com/google/uuid"
)

type Service struct {
	config     Config
	configPath string
	conic      *conic.Conic

	tokenManager *TokenManager
	api          *Client
	gateway      *GatewayConnection

	ctx       context.Context
	ctxCancel context.CancelFunc

	grb    *GoroBot.Instant
	status botc.LoginStatus
	logger logger.Inst
}

func Create() *Service {
	return &Service{
		configPath: DefaultConfigPath,
		conic:      conic.New(),
		status:     botc.Offline,
	}
}

func (s *Service) Name() string {
	return "QBot-adapter"
}

func (s *Service) Init(grb *GoroBot.Instant) error {
	s.grb = grb
	s.logger = grb.GetLogger()
	s.ctx, s.ctxCancel = context.WithCancel(context.Background())

	if err := s.initConfig(); err != nil {
		return err
	}

	if !s.config.IsConfigured() {
		s.logInfo("QBot API 凭据未配置，启动手机 QQ 扫码接入流程...")
		creds, err := s.QRConnect(s.ctx)
		if err != nil {
			return fmt.Errorf("qbot qr connect failed: %w", err)
		}
		s.config.Credentials.AppID = creds.AppID
		s.config.Credentials.Secret = creds.Secret
		if err := s.conic.WriteConfig(); err != nil {
			s.logWarning("QBot 凭据写入配置文件失败: %v", err)
		} else {
			s.logSuccess("QBot 扫码接入成功！凭据已保存至配置文件，AppID: %s", creds.AppID)
		}
	}

	s.tokenManager = NewTokenManager(DefaultTokenBaseURL, s.logger)
	s.tokenManager.StartBackgroundRefresh(s.ctx, s.config.Credentials.AppID, s.config.Credentials.ClientSecret())
	s.api = NewClient(DefaultAPIBaseURL, s.config.Credentials, s.tokenManager, s.logger, s.config.Debug)

	mode := strings.ToLower(s.config.Mode)
	switch mode {
	case "webhook":
		if err := s.runHttp(); err != nil {
			return err
		}
	case "both":
		if err := s.runHttp(); err != nil {
			return err
		}
		s.gateway = NewGatewayConnection(s)
		s.gateway.Start(s.ctx)
	default: // "websocket" 或留空（默认）
		s.gateway = NewGatewayConnection(s)
		s.gateway.Start(s.ctx)
		if wb := s.config.GetWebhookConfig(); wb.Port > 0 {
			if err := s.runHttp(); err != nil {
				s.logWarning("QBot run resource server error: %v", err)
			}
		}
	}

	s.status = botc.Online
	grb.AddContext(s)

	return nil
}

func (s *Service) Release(grb *GoroBot.Instant) error {
	if s.ctxCancel != nil {
		s.ctxCancel()
	}
	s.status = botc.Offline
	return nil
}

func (s *Service) API() *Client {
	return s.api
}

func (s *Service) ID() string {
	u, err := s.api.Me(context.Background())
	if err != nil {
		return fmt.Sprintf("%s:%s", s.Protocol(), s.config.Credentials.AppID)
	}
	return fmt.Sprintf("%s:%s", s.Protocol(), u.ID)
}

func (s *Service) logInfo(format string, args ...any) {
	if s.logger != nil {
		s.logger.Info(format, args...)
	}
}

func (s *Service) logWarning(format string, args ...any) {
	if s.logger != nil {
		s.logger.Warning(format, args...)
	}
}

func (s *Service) logError(format string, args ...any) {
	if s.logger != nil {
		s.logger.Error(format, args...)
	}
}

func (s *Service) logSuccess(format string, args ...any) {
	if s.logger != nil {
		s.logger.Success(format, args...)
	}
}

func (s *Service) logDebug(format string, args ...any) {
	if s.logger != nil {
		s.logger.Debug(format, args...)
	}
}

func (s *Service) Protocol() string {
	return "qbot"
}

func (s *Service) Status() botc.LoginStatus {
	return s.status
}

func (s *Service) NewMessageBuilder() botc.MessageBuilder {
	return NewMessageBuilder(&MessageContext{
		bot: s,
	})
}

func (s *Service) SendDirectMessage(target entity.User, elements []*botc.MessageElement) (*botc.BaseMessage, error) {
	builder := NewMessageBuilder(&MessageContext{bot: s})
	builder.ApplyElements(elements)
	return builder.Send(FormatID("user", target.ID))
}

func (s *Service) SendGroupMessage(target entity.Group, elements []*botc.MessageElement) (*botc.BaseMessage, error) {
	builder := NewMessageBuilder(&MessageContext{bot: s})
	builder.ApplyElements(elements)
	return builder.Send(FormatID("group", target.ID))
}

func (s *Service) Contacts() []entity.User {
	return nil
}

func (s *Service) Groups() []entity.Group {
	return nil
}

func (s *Service) DownloadResourceFromRefLink(refLink string) (string, error) {
	values, err := urlpkg.ParseQuery(refLink)
	if err != nil {
		return "", fmt.Errorf("invalid ref link: %w", err)
	}

	rawURL := values.Get("url")
	if rawURL == "" {
		return "", fmt.Errorf("ref link missing url")
	}

	target := values.Get("target")
	if target == "" {
		ext := values.Get("ext")
		if ext == "" {
			if u, err := urlpkg.Parse(rawURL); err == nil {
				ext = path.Ext(u.Path)
			}
		}
		if ext == "" {
			ext = ".dat"
		} else if !strings.HasPrefix(ext, ".") {
			ext = "." + ext
		}
		target = filepath.Join("resources", uuid.NewString()+ext)
	}

	resp, err := http.Get(rawURL)
	if err != nil {
		return "", fmt.Errorf("request resource failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read resource failed: %w", err)
	}

	if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
		return "", fmt.Errorf("create cache dir failed: %w", err)
	}

	if err := os.WriteFile(target, data, 0644); err != nil {
		return "", fmt.Errorf("write resource failed: %w", err)
	}

	return target, nil
}
