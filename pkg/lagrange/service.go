package lagrange

import (
	GoroBot "github.com/Jel1ySpot/GoroBot/pkg/core"
	"github.com/Jel1ySpot/GoroBot/pkg/core/bot"
	"github.com/Jel1ySpot/GoroBot/pkg/core/logger"
	"github.com/Jel1ySpot/conic"
	"github.com/LagrangeDev/LagrangeGo/client"
	"os"
	"path"
	"strconv"
)

const (
	ConfigPath = "conf/lagrange/"
)

type Service struct {
	config   Config
	qqClient *client.QQClient
	bot      *GoroBot.Instant
	owner    uint32
	status   bot.LoginStatus

	conic  *conic.Conic
	logger logger.Inst
}

func (s *Service) Name() string {
	return "Lagrange"
}

func Create() *Service {
	return &Service{
		conic:  conic.New(),
		status: bot.Offline,
	}
}

func (s *Service) Init(bot *GoroBot.Instant) error {
	s.bot = bot
	s.logger = bot.GetLogger()
	if id, ok := bot.GetOwner("qq"); ok {
		if uin, err := strconv.ParseUint(id, 10, 32); err == nil {
			s.owner = uint32(uin)
		}
	}

	if err := s.InitConic(); err != nil {
		return err
	}

	s.qqClient = client.NewClient(0, "")

	if err := s.login(); err != nil {
		return err
	}

	return nil
}

func (s *Service) Release(bot *GoroBot.Instant) error {
	if s.qqClient != nil {
		if err := s.releaseQQClient(); err != nil {
			return err
		}
	}

	return nil
}

func (s *Service) releaseQQClient() error {
	s.qqClient.Release()

	func() {
		data, err := s.qqClient.Sig().Marshal()
		if err != nil {
			s.logger.Error("marshal sig.bin err: %s", err)
			return
		}
		err = os.WriteFile(path.Join(ConfigPath, s.config.Account.SigPath), data, 0644)
		if err != nil {
			s.logger.Error("write sig.bin err: %s", err)
			return
		}
		s.logger.Info("sig saved into %s", path.Join(ConfigPath, s.config.Account.SigPath))
	}()

	return nil
}
