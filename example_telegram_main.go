package main

import (
	"github.com/Jel1ySpot/GoroBot/example_plugin/message_logger"
	"github.com/Jel1ySpot/GoroBot/example_plugin/ping"
	"github.com/Jel1ySpot/GoroBot/example_plugin/tests"
	GoroBot "github.com/Jel1ySpot/GoroBot/pkg/core"
	botc "github.com/Jel1ySpot/GoroBot/pkg/core/bot_context"
	"github.com/Jel1ySpot/GoroBot/pkg/core/command"
	TelegramClient "github.com/Jel1ySpot/GoroBot/pkg/telegram"
	_ "github.com/mattn/go-sqlite3"
)

// telegram_example 展示了如何运行 Telegram 适配器
func telegram_example() {
	grb := GoroBot.Create()

	if err := grb.OpenDatabase("sqlite3", "bot.db"); err != nil {
		panic(err)
	}

	tg := TelegramClient.Create()

	grb.Use(tg)
	grb.Use(message_logger.Create())
	grb.Use(ping.Create())
	grb.Use(tests.Create())

	// 简单的 ping 测试
	var del func()
	del, _ = grb.On(GoroBot.MessageEvent(func(ctx botc.MessageContext) {
		if ctx.String() == "ping" {
			_, _ = ctx.ReplyText("🏓 pong (telegram)")
			del()
		}
	}))

	// 示例命令：/echo <content>
	_command_del, _ := grb.Command("echo").
		Argument("content", command.String, true, "").
		Action(func(ctx *command.Context) error {
			_, _ = ctx.ReplyText(ctx.KvArgs["content"])
			return nil
		}).Build()

	_ = _command_del

	if err := grb.Run(); err != nil {
		panic(err)
	}
}
