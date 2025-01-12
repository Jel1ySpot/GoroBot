package ping

import (
	GoroBot "github.com/Jel1ySpot/GoroBot/pkg/core"
	"github.com/Jel1ySpot/GoroBot/pkg/core/command"
	"github.com/Jel1ySpot/GoroBot/pkg/core/message"
)

type Service struct {
	bot *GoroBot.Instant

	releaseFunc []func()
}

func (s *Service) Name() string {
	return "Ping"
}

func Create() *Service {
	return &Service{}
}

func (s *Service) Init(grb *GoroBot.Instant) error {
	s.bot = grb

	msgFn, _ := grb.On(GoroBot.MessageEvent(func(ctx GoroBot.BotContext, msg message.Context) error {
		if msg.String() == "ping" {
			_ = msg.ReplyText("🏓")
		}
		return nil
	}))

	cmdFn, err := grb.Command("ping", func(ctx GoroBot.BotContext, cmd *command.Context) {
		_ = cmd.ReplyText("🏓")
	}).Build()
	if err != nil {
		return err
	}

	s.releaseFunc = append(s.releaseFunc, msgFn, cmdFn)

	return nil
}

func (s *Service) Release(grb *GoroBot.Instant) error {
	for _, fn := range s.releaseFunc {
		fn()
	}
	return nil
}
