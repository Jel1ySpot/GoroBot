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
	"time"

	GoroBot "github.com/Jel1ySpot/GoroBot/pkg/core"
	botc "github.com/Jel1ySpot/GoroBot/pkg/core/bot_context"
	"github.com/Jel1ySpot/GoroBot/pkg/core/logger"
	"github.com/Jel1ySpot/conic"
	"github.com/google/uuid"
	"github.com/tencent-connect/botgo"
	"github.com/tencent-connect/botgo/openapi"
	"github.com/tencent-connect/botgo/token"
)

type Service struct {
	config     Config
	configPath string
	conic      *conic.Conic

	api       openapi.OpenAPI
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
	_, s.ctxCancel = context.WithCancel(context.Background())

	if err := s.initConfig(); err != nil {
		return err
	}

	tokenSource := token.NewQQBotTokenSource(&s.config.Credentials)
	if err := token.StartRefreshAccessToken(context.Background(), tokenSource); err != nil {
		return err
	}
	s.api = botgo.NewOpenAPI(s.config.Credentials.AppID, tokenSource).WithTimeout(5 * time.Second).SetDebug(s.config.Debug)

	s.registerHandlers()

	if err := s.runHttp(); err != nil {
		return err
	}

	grb.AddContext(NewContext(s))

	return nil
}

func (s *Service) Release(grb *GoroBot.Instant) error {
	s.ctxCancel()
	return nil
}

func (s *Service) Protocol() string {
	return "qbot"
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
