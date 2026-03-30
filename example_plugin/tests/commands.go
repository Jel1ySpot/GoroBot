package tests

import (
	"encoding/json"

	"github.com/Jel1ySpot/GoroBot/pkg/core/command"
)

func (s *Service) AddCommand(name string, desc string, action command.Handler) error {
	delFn, _ := s.bot.Command(name).Description(desc).Action(action).Build()

	s.releaseFunc = append(s.releaseFunc, delFn)

	return nil
}

func (s *Service) CommandsRegistry() error {
	if err := s.AddCommand("repr", "输出消息的 JSON 结构", reprAction); err != nil {
		return err
	}

	if err := s.AddCommand("getOwner", "获取当前上下文的 owner", s.getOwnerAction); err != nil {
		return err
	}

	if err := s.AddCommand("getResourceFromID", "根据 ID 获取资源路径", s.getResourceFromIDAction); err != nil {
		return err
	}

	if err := s.AddCommand("sendImage", "发送指定路径的图片", s.sendImageAction); err != nil {
		return err
	}

	return nil
}

func reprAction(ctx *command.Context) error {
	msg := ctx.Message()
	data, err := json.MarshalIndent(msg, "", "  ")
	if err != nil {
		_, _ = ctx.ReplyText("marshal message error: ", err.Error())
		return err
	}
	_, _ = ctx.ReplyText(string(data))

	return nil
}

func (s *Service) getOwnerAction(ctx *command.Context) error {
	owner, ok := s.bot.GetOwner(ctx.BotContext().ID())
	if !ok {
		_, _ = ctx.ReplyText("owner not configured for context ", ctx.BotContext().ID())
		return nil
	}
	_, _ = ctx.ReplyText("owner for ", ctx.BotContext().ID(), ": ", owner, ctx.SenderID())
	return nil
}

func (s *Service) getResourceFromIDAction(ctx *command.Context) error {
	if len(ctx.Arguments) == 0 {
		_, _ = ctx.ReplyText("usage: getResourceFromID <id>")
		return nil
	}
	id := ctx.Arguments[0]
	path, err := s.bot.LoadResourceFromID(id)
	if err != nil {
		return err
	}
	_, _ = ctx.ReplyText("resource path: ", path)
	return nil
}

func (s *Service) sendImageAction(ctx *command.Context) error {
	if len(ctx.Arguments) == 0 {
		_, _ = ctx.ReplyText("usage: sendImage <path>")
		return nil
	}

	path := ctx.Arguments[0]
	_, err := ctx.BotContext().NewMessageBuilder().ImageFromFile(path).ReplyTo(ctx)
	if err != nil {
		_, _ = ctx.ReplyText("send image error: ", err.Error())
		return err
	}
	return nil
}
