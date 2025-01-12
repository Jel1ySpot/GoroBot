package GoroBot

import (
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

func (i *Instant) EventUnregister(eventName string) {
	i.event.Unregister(eventName)
}

func (i *Instant) EventEmit(eventName string, args ...interface{}) error {
	if eventName == "message" {
		return i.MessageEmit(args[0].(BotContext), args[1].(message.Context))
	}
	if eventName == "command" {
		i.CommandEmit(args[0].(BotContext), args[1].(*command.Context))
		return nil
	}
	return i.event.Emit(eventName, args...)
}

func (i *Instant) MessageEmit(ctx BotContext, msg message.Context) error {
	// 中间件
	return i.middleware.dispatch(ctx, msg, func() error {
		go func() {
			for _, cmdReg := range i.commands.Commands {
				cmdReg.CheckAlias(ctx, command.NewCommandContext(msg, msg.String()))
			}
		}()
		return i.event.Emit("message", ctx, msg)
	})
}

func (i *Instant) CommandEmit(ctx BotContext, cmd *command.Context) {
	// 中间件
	_ = i.middleware.dispatch(ctx, cmd, func() error {
		go i.event.Emit("message", ctx, cmd.Context)
		go i.commands.Emit(ctx, cmd)
		return nil
	})
}

type MessageEventCallback func(BotContext, message.Context) error

func MessageEvent(callback MessageEventCallback) EventHandler {
	return EventHandler{
		Name: "message",
		Callback: func(args ...interface{}) error {
			return callback(args[0].(BotContext), args[1].(message.Context))
		},
	}
}

type CommandEventCallback func(BotContext, *command.Context) error

func CommandEvent(callback CommandEventCallback) EventHandler {
	return EventHandler{
		Name: "command",
		Callback: func(args ...interface{}) error {
			return callback(args[0].(BotContext), args[1].(*command.Context))
		},
	}
}
