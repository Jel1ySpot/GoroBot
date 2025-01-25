package GoroBot

import (
	botc "github.com/Jel1ySpot/GoroBot/pkg/core/bot_context"
	"github.com/Jel1ySpot/GoroBot/pkg/core/command"
	"github.com/Jel1ySpot/GoroBot/pkg/core/event"
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
		return i.MessageEmit(args[0].(botc.MessageContext))
	}
	if eventName == "command" {
		i.CommandEmit(args[0].(*command.Context))
		return nil
	}
	return i.event.Emit(eventName, args...)
}

func (i *Instant) MessageEmit(msg botc.MessageContext) error {
	// 中间件
	return i.middleware.dispatch(msg, func() error {
		go func() {
			for _, cmdReg := range i.commands.Commands {
				cmdReg.CheckAlias(command.NewCommandContext(msg, msg.String()))
			}
		}()
		return i.event.Emit("message", msg)
	})
}

func (i *Instant) CommandEmit(cmd *command.Context) {
	// 中间件
	_ = i.middleware.dispatch(cmd, func() error {
		go i.event.Emit("message", cmd.MessageContext)
		go i.commands.Emit(cmd)
		return nil
	})
}

type MessageEventCallback func(ctx botc.MessageContext)

func MessageEvent(callback MessageEventCallback) EventHandler {
	return EventHandler{
		Name: "message",
		Callback: func(args ...interface{}) {
			callback(args[0].(botc.MessageContext))
		},
	}
}

type CommandEventCallback func(ctx *command.Context)

func CommandEvent(callback CommandEventCallback) EventHandler {
	return EventHandler{
		Name: "command",
		Callback: func(args ...interface{}) {
			callback(args[0].(*command.Context))
		},
	}
}
