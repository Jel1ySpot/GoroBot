package main

import (
	"github.com/Jel1ySpot/GoroBot/example_plugin/message_logger"
	GoroBot "github.com/Jel1ySpot/GoroBot/pkg/core"
	"github.com/Jel1ySpot/GoroBot/pkg/core/message"
	LgrClient "github.com/Jel1ySpot/GoroBot/pkg/lagrange"
)

func main() {
	grb := GoroBot.Create()

	lgr := LgrClient.Create()

	grb.Use(lgr)
	grb.Use(message_logger.Create())

	var del func()
	del, _ = grb.On(GoroBot.MessageEvent(func(ctx GoroBot.BotContext, msg message.Context) error {
		if msg.String() == "ping" {
			_ = msg.ReplyText("🏓")
			del()
		}
		return nil
	}))

	if err := grb.Run(); err != nil {
		panic(err)
	}
}
