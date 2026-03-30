package main

import (
	"context"

	"github.com/Jel1ySpot/GoroBot/example_plugin/message_logger"
	"github.com/Jel1ySpot/GoroBot/example_plugin/ping"
	"github.com/Jel1ySpot/GoroBot/example_plugin/tests"
	GoroBot "github.com/Jel1ySpot/GoroBot/pkg/core"
	botc "github.com/Jel1ySpot/GoroBot/pkg/core/bot_context"
	"github.com/Jel1ySpot/GoroBot/pkg/core/command"
	TelegramClient "github.com/Jel1ySpot/GoroBot/pkg/telegram"
	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
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
	_, _ = grb.Command("echo").
		Description("回声测试").
		Argument("content", command.String, true, "").
		Action(func(ctx *command.Context) error {
			_, _ = ctx.ReplyText(ctx.KvArgs["content"])
			return nil
		}).Build()

	// 关闭自定义键盘：/closekb
	_, _ = grb.Command("closekb").
		Description("关闭自定义键盘").
		Action(func(ctx *command.Context) error {
			msg := ctx.Message()
			if msg.Sender == nil {
				return nil
			}
			chatID := msg.Sender.From
			if chatID == nil {
				chatID = msg.Sender.Base
			}
			id, err := TelegramClient.ParseChatID(chatID.ID)
			if err != nil {
				return err
			}
			_, err = tg.Bot().SendMessage(context.Background(), &bot.SendMessageParams{
				ChatID:      id,
				Text:        "已关闭自定义键盘",
				ReplyMarkup: &models.ReplyKeyboardRemove{RemoveKeyboard: true},
			})
			return err
		}).Build()

	if err := grb.Run(); err != nil {
		panic(err)
	}
}
