package qbot

import (
	"context"
	GoroBot "github.com/Jel1ySpot/GoroBot/pkg/core"
	botc "github.com/Jel1ySpot/GoroBot/pkg/core/bot_context"
	"github.com/Jel1ySpot/GoroBot/pkg/core/logger"
	"github.com/Jel1ySpot/conic"
	"github.com/tencent-connect/botgo"
	"github.com/tencent-connect/botgo/openapi"
	"github.com/tencent-connect/botgo/token"
	"time"
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
