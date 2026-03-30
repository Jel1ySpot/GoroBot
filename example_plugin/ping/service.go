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

	delFn, _ := grb.Command("ping").
		Description("Ping 测试").
		Alias("^ping$", nil).
		Action(func(ctx *command.Context) error {
			_, _ = ctx.ReplyText("🏓")
			return nil
		}).
		Build()

	s.releaseFunc = append(s.releaseFunc, delFn)

	return nil
}

func (s *Service) Release(grb *GoroBot.Instant) error {
	for _, fn := range s.releaseFunc {
		fn()
	}
	return nil
}
