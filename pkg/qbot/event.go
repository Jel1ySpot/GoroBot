package qbot

import (
	"github.com/Jel1ySpot/GoroBot/pkg/core/command"
	"github.com/tencent-connect/botgo/dto"
	"github.com/tencent-connect/botgo/event"
	"strings"
)

func (s *Service) registerHandlers() {
	_ = event.RegisterHandlers(
		// ***********消息事件***********
		// 群@机器人消息事件
		event.ATMessageEventHandler(func(event *dto.WSPayload, data *dto.WSATMessageData) error {
			return s.emitMessage(event, (*dto.Message)(data))
		}),
		// C2C消息事件
		event.C2CMessageEventHandler(func(event *dto.WSPayload, data *dto.WSC2CMessageData) error {
			return s.emitMessage(event, (*dto.Message)(data))
		}),
		// 频道@机器人事件
		event.GroupATMessageEventHandler(func(event *dto.WSPayload, data *dto.WSGroupATMessageData) error {
			return s.emitMessage(event, (*dto.Message)(data))
		}),
	)
}

func (s *Service) emitMessage(event *dto.WSPayload, data *dto.Message) error {
	data.Content = strings.TrimSpace(data.Content)
	if strings.HasPrefix(data.Content, "/") {
		return s.emitCommand(event, data)
	}
	return s.grb.MessageEmit(&MessageContext{
		NewContext(s),
		&Message{event, data},
	})
}

func (s *Service) emitCommand(event *dto.WSPayload, data *dto.Message) error {
	s.grb.CommandEmit(command.NewCommandContext(&MessageContext{
		NewContext(s),
		&Message{event, data},
	}, data.Content[1:]))
	return nil
}
