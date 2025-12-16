package tests

import (
	"encoding/json"

	"github.com/Jel1ySpot/GoroBot/pkg/core/command"
)

func (s *Service) AddCommand(name string, action command.Handler) error {
	delFn, _ := s.bot.Command(name).Action(action).Build()

	s.releaseFunc = append(s.releaseFunc, delFn)

	return nil
}

func (s *Service) CommandsRegistry() error {
	if err := s.AddCommand("repr", reprAction); err != nil {
		return err
	}

	if err := s.AddCommand("getOwner", s.getOwnerAction); err != nil {
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
