package lagrange

import (
	"github.com/Jel1ySpot/GoroBot/pkg/core/command"
	"github.com/Jel1ySpot/GoroBot/pkg/core/message"
	"github.com/LagrangeDev/LagrangeGo/client"
	lgrMessage "github.com/LagrangeDev/LagrangeGo/message"
	"github.com/google/shlex"
	"strings"
)

func (s *Service) eventSubscribe() error {
	qqClient := s.qqClient

	qqClient.DisconnectedEvent.Subscribe(func(client *client.QQClient, event *client.DisconnectedEvent) {
		s.logger.Warning("连接已断开：%v", event.Message)
	})

	qqClient.GroupMessageEvent.Subscribe(func(client *client.QQClient, event *lgrMessage.GroupMessage) {
		ctx := MessageContext{
			messageType: message.GroupMessage,
			service:     s,
			groupMsg:    event,
		}
		if strings.HasPrefix(event.ToString(), s.config.CommandPrefix) {
			tokens, _ := shlex.Split(ctx.String()[len(s.config.CommandPrefix):])
			_ = s.bot.CommandEmit(
				&Context{service: s},
				&command.Context{
					Context: &ctx,
					Tokens:  tokens,
				},
			)
		} else {
			_ = s.bot.MessageEmit(
				&Context{service: s},
				&ctx,
			)
		}
	})

	qqClient.PrivateMessageEvent.Subscribe(func(client *client.QQClient, event *lgrMessage.PrivateMessage) {
		ctx := MessageContext{
			messageType: message.DirectMessage,
			service:     s,
			privateMsg:  event,
		}
		if strings.HasPrefix(event.ToString(), s.config.CommandPrefix) {
			tokens, _ := shlex.Split(ctx.String()[len(s.config.CommandPrefix):])
			_ = s.bot.CommandEmit(
				&Context{service: s},
				&command.Context{
					Context: &ctx,
					Tokens:  tokens,
				},
			)
		} else {
			_ = s.bot.MessageEmit(
				&Context{service: s},
				&ctx,
			)
		}
	})

	return nil
}
