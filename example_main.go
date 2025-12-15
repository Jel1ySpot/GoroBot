package main

import (
	"github.com/Jel1ySpot/GoroBot/example_plugin/message_logger"
	GoroBot "github.com/Jel1ySpot/GoroBot/pkg/core"
	botc "github.com/Jel1ySpot/GoroBot/pkg/core/bot_context"
	"github.com/Jel1ySpot/GoroBot/pkg/core/command"
	LgrClient "github.com/Jel1ySpot/GoroBot/pkg/lagrange"
)

func example_main() {
	grb := GoroBot.Create()

	lgr := LgrClient.Create()

	grb.Use(lgr)
	grb.Use(message_logger.Create())

	var del func()
	del, _ = grb.On(GoroBot.MessageEvent(func(ctx botc.MessageContext) {
		if ctx.String() == "ping" {
			_, _ = ctx.ReplyText("🏓")
			del()
		}
	}))

	_command_del, _ := grb.Command("echo <content:text>").
		Action(func(ctx *command.Context) {
			ctx.ReplyText(ctx.KvArgs["content"])
		}).
		Build()

	_ = _command_del

	if err := grb.Run(); err != nil {
		panic(err)
	}
}
