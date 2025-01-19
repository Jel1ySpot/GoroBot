package ping

import (
	GoroBot "github.com/Jel1ySpot/GoroBot/pkg/core"
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

	cmdFn, _ := grb.Command("ping", func(cmd *command.Context) {
		_, _ = cmd.ReplyText("🏓")
	}).Alias("^ping$", nil).Build()

	_, _ = grb.Command("test", func(cmd *command.Context) {
		_, _ = cmd.NewMessageBuilder().ImageFromFile("./test.png").ReplyTo(cmd.MessageContext)
	}).Build()

	s.releaseFunc = append(s.releaseFunc, cmdFn)

	return nil
}

func (s *Service) Release(grb *GoroBot.Instant) error {
	for _, fn := range s.releaseFunc {
		fn()
	}
	return nil
}
