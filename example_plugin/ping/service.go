package ping

import (
	GoroBot "github.com/Jel1ySpot/GoroBot/pkg/core"
	botc "github.com/Jel1ySpot/GoroBot/pkg/core/bot_context"
	"github.com/Jel1ySpot/GoroBot/pkg/core/command"
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

	msgFn, _ := grb.On(GoroBot.MessageEvent(func(msg botc.MessageContext) error {
		if msg.String() == "ping" {
			_, _ = msg.ReplyText("🏓")
		}
		return nil
	}))

	cmdFn, err := grb.Command("ping", func(cmd *command.Context) {
		_, _ = cmd.ReplyText("🏓")
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
