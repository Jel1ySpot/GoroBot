package message_logger

import (
	GoroBot "github.com/Jel1ySpot/GoroBot/pkg/core"
	"github.com/Jel1ySpot/GoroBot/pkg/core/bot"
	"github.com/Jel1ySpot/GoroBot/pkg/core/logger"
	"github.com/Jel1ySpot/GoroBot/pkg/core/message"
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

	s.releaseFunc, _ = grb.On(GoroBot.MessageEvent(func(ctx bot.Context, msg message.Context) error {
		s.log(msg)
		return nil
	}))

	return nil
}

func (s *Service) Release(grb *GoroBot.Instant) error {
	s.releaseFunc()
	return nil
}

func (s *Service) log(ctx message.Context) {
	log := s.logger
	msg := ctx.Message()

	switch msg.MessageType {
	case message.GroupMessage:
		log.Info("Message from [%s]: [%s]%s(%s): %s", ctx.Protocol(), msg.Sender.From, msg.Sender.Nickname, msg.Sender.ID, msg.Content)
	case message.DirectMessage:
		log.Info("Message from [%s]: %s(%s): %s", ctx.Protocol(), msg.Sender.Name, msg.Sender.ID, msg.Content)
	}
}
