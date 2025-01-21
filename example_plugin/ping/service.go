package ping

import (
	"fmt"
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

	cmdFn, _ := grb.Command("ping").
		Alias("^ping$", nil).
		Action(func(cmd *command.Context) {
			_, _ = cmd.ReplyText("🏓")
		}).
		Build()

	testCmd := grb.Command("test")

	_, _ = testCmd.SubCommand("image").
		Action(func(cmd *command.Context) {
			_, _ = cmd.NewMessageBuilder().ImageFromFile("./test.png").ReplyTo(cmd.MessageContext)
		}).Build()

	_, _ = testCmd.SubCommand("args <some> [text:text]").
		Action(func(cmd *command.Context) {
			_, _ = cmd.NewMessageBuilder().Text(fmt.Sprintf("%#v", cmd.Args)).ReplyTo(cmd.MessageContext)
		}).Build()

	_, _ = testCmd.SubCommand("opt").
		Option("-s [str:string]=default").
		Option("-t [txt:text]=default").
		Action(func(ctx *command.Context) {
			_, _ = ctx.NewMessageBuilder().Text(fmt.Sprintf("%#v", ctx.Options)).ReplyTo(ctx.MessageContext)
		}).Build()

	_, _ = testCmd.Build()

	s.releaseFunc = append(s.releaseFunc, cmdFn)

	return nil
}

func (s *Service) Release(grb *GoroBot.Instant) error {
	for _, fn := range s.releaseFunc {
		fn()
	}
	return nil
}
