package lagrange

import (
	"github.com/Jel1ySpot/GoroBot/pkg/core/command"
	"github.com/LagrangeDev/LagrangeGo/client"
	LgrMessage "github.com/LagrangeDev/LagrangeGo/message"
	"strings"
)

func (s *Service) eventSubscribe() error {
	qqClient := s.qqClient

	qqClient.DisconnectedEvent.Subscribe(func(client *client.QQClient, event *client.DisconnectedEvent) {
		s.logger.Warning("连接已断开：%v", event.Message)
	})

	qqClient.GroupMessageEvent.Subscribe(func(client *client.QQClient, event *LgrMessage.GroupMessage) {
		go s.messageEventHandler(event)
	})

	qqClient.PrivateMessageEvent.Subscribe(func(client *client.QQClient, event *LgrMessage.PrivateMessage) {
		go s.messageEventHandler(event)
	})

	return nil
}

func (s *Service) messageEventHandler(event any) {
	msg := NewMessageContext(event, s)
	if strings.HasPrefix(msg.String(), s.config.CommandPrefix) {
		text := msg.String()[len(s.config.CommandPrefix):]
		s.bot.CommandEmit(
			s.getContext(),
			command.NewCommandContext(msg, text),
		)
	} else {
		_ = s.bot.MessageEmit(
			s.getContext(),
			msg,
		)
	}
}
