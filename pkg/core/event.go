package GoroBot

import (
	"github.com/Jel1ySpot/GoroBot/pkg/core/bot"
	"github.com/Jel1ySpot/GoroBot/pkg/core/command"
	"github.com/Jel1ySpot/GoroBot/pkg/core/event"
	"github.com/Jel1ySpot/GoroBot/pkg/core/message"
)

type EventHandler struct {
	Name     string
	Callback event.Callback
}

func (i *Instant) On(event EventHandler) (func(), error) {
	return i.event.On(event.Name, event.Callback)
}

func (i *Instant) EventRegister(eventName string) {
	i.event.Register(eventName)
}

func (i *Instant) EventEmit(eventName string, args ...interface{}) error {
	if eventName == "message" {
		return i.MessageEmit(args[0].(bot.Context), args[1].(message.Context))
	}
	if eventName == "command" {
		return i.CommandEmit(args[0].(bot.Context), args[1].(*command.Context))
	}
	return i.event.Emit(eventName, args...)
}

func (i *Instant) MessageEmit(ctx bot.Context, msg message.Context) error {
	// 中间件
	return i.middleware.dispatch(ctx, msg, func() error {
		return i.event.Emit("message", ctx, msg)
	})
}

func (i *Instant) CommandEmit(ctx bot.Context, cmd *command.Context) error {
	// 中间件
	return i.middleware.dispatch(ctx, cmd, func() error {
		return i.event.Emit("command", ctx, cmd)
	})
}

type MessageEventCallback func(bot.Context, message.Context) error

func MessageEvent(callback MessageEventCallback) EventHandler {
	return EventHandler{
		Name: "message",
		Callback: func(args ...interface{}) error {
			return callback(args[0].(bot.Context), args[1].(message.Context))
		},
	}
}

type CommandEventCallback func(bot.Context, *command.Context) error

func CommandEvent(callback CommandEventCallback) EventHandler {
	return EventHandler{
		Name: "command",
		Callback: func(args ...interface{}) error {
			return callback(args[0].(bot.Context), args[1].(*command.Context))
		},
	}
}
