package main

import (
	"github.com/Jel1ySpot/GoroBot/example_plugin/dice"
	"github.com/Jel1ySpot/GoroBot/example_plugin/message_logger"
	"github.com/Jel1ySpot/GoroBot/example_plugin/ping"
	GoroBot "github.com/Jel1ySpot/GoroBot/pkg/core"
	botc "github.com/Jel1ySpot/GoroBot/pkg/core/bot_context"
	"github.com/Jel1ySpot/GoroBot/pkg/core/command"
	OneBotClient "github.com/Jel1ySpot/GoroBot/pkg/onebot"
	_ "github.com/mattn/go-sqlite3"
)

func onebot_example() {
	grb := GoroBot.Create()

	if err := grb.OpenDatabase("sqlite3", "bot.db"); err != nil {
		panic(err)
	}

	// Create OneBot service
	onebot := OneBotClient.Create()

	// Use the services
	grb.Use(onebot)
	grb.Use(message_logger.Create())
	grb.Use(ping.Create())
	grb.Use(dice.Create())

	// Example: Echo bot that responds to "ping" with "pong"
	var del func()
	del, _ = grb.On(GoroBot.MessageEvent(func(ctx botc.MessageContext) {
		if ctx.String() == "ping" {
			_, _ = ctx.ReplyText("🏓 pong")
			del() // Remove this handler after first use
		}
	}))

	// Example: Command-style handler using the correct API
	_command_del, _ := grb.Command("echo").
		Argument("content", command.String, true, "").
		Action(func(ctx *command.Context) {
			ctx.ReplyText(ctx.KvArgs["content"])
		}).Build()

	// Keep the command handler reference to avoid unused variable error
	_ = _command_del

	// Run the bot
	if err := grb.Run(); err != nil {
		panic(err)
	}
}
