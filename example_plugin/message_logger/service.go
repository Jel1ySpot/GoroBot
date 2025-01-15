package message_logger

import (
	GoroBot "github.com/Jel1ySpot/GoroBot/pkg/core"
	botc "github.com/Jel1ySpot/GoroBot/pkg/core/bot_context"
	"github.com/Jel1ySpot/GoroBot/pkg/core/logger"
)

type Service struct {
	bot    *GoroBot.Instant
	logger logger.Inst

	releaseFunc func()
}

func (s *Service) Name() string {
	return "MessageLogger"
}

func Create() *Service {
	return &Service{}
}

func (s *Service) Init(grb *GoroBot.Instant) error {
	s.bot = grb
	s.logger = grb.GetLogger()

	s.releaseFunc, _ = grb.On(GoroBot.MessageEvent(func(ctx botc.MessageContext) error {
		s.log(ctx)
		return nil
	}))

	return nil
}

func (s *Service) Release(grb *GoroBot.Instant) error {
	if s.releaseFunc != nil {
		s.releaseFunc()
	}
	return nil
}

func (s *Service) log(ctx botc.MessageContext) {
	log := s.logger
	msg := ctx.Message()

	switch msg.MessageType {
	case botc.GroupMessage:
		log.Info("Message from [%s]: [%s]%s(%s): %s", ctx.Protocol(), msg.Sender.From, msg.Sender.Nickname, msg.Sender.ID, msg.Content)
	case botc.DirectMessage:
		log.Info("Message from [%s]: %s(%s): %s", ctx.Protocol(), msg.Sender.Name, msg.Sender.ID, msg.Content)
	}
}
