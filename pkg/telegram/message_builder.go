package telegram

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"strings"

	botc "github.com/Jel1ySpot/GoroBot/pkg/core/bot_context"
	"github.com/Jel1ySpot/GoroBot/pkg/core/command"
	"github.com/Jel1ySpot/GoroBot/pkg/core/entity"
	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

type MessageBuilder struct {
	service  *Service
	elements []*botc.MessageElement
	err      error
}

func (m *MessageBuilder) Protocol() string {
	return "telegram"
}

func (m *MessageBuilder) Text(text string) botc.MessageBuilder {
	if m.err != nil {
		return m
	}
	m.elements = append(m.elements, &botc.MessageElement{
		Type:    botc.TextElement,
		Content: text,
	})
	return m
}

func (m *MessageBuilder) Markdown(content string) botc.MessageBuilder {
	if m.err != nil {
		return m
	}
	m.elements = append(m.elements, &botc.MessageElement{
		Type:    botc.MarkdownElement,
		Content: content,
	})
	return m
}

func (m *MessageBuilder) Quote(msg *botc.BaseMessage) botc.MessageBuilder {
	if m.err != nil {
		return m
	}
	m.elements = append(m.elements, &botc.MessageElement{
		Type:    botc.QuoteElement,
		Content: msg.ID,
	})
	return m
}

func (m *MessageBuilder) Mention(id string) botc.MessageBuilder {
	if m.err != nil {
		return m
	}
	m.elements = append(m.elements, &botc.MessageElement{
		Type:    botc.MentionElement,
		Content: id,
		Source:  id,
	})
	return m
}

func (m *MessageBuilder) ImageFromFile(path string) botc.MessageBuilder {
	if m.err != nil {
		return m
	}
	if _, err := os.Stat(path); err != nil {
		m.err = fmt.Errorf("读取图片失败: %w", err)
		return m
	}
	m.elements = append(m.elements, &botc.MessageElement{
		Type:   botc.ImageElement,
		Source: path,
	})
	return m
}

func (m *MessageBuilder) ImageFromUrl(url string) botc.MessageBuilder {
	if m.err != nil {
		return m
	}
	m.elements = append(m.elements, &botc.MessageElement{
		Type:   botc.ImageElement,
		Source: url,
	})
	return m
}

func (m *MessageBuilder) ImageFromData(data []byte) botc.MessageBuilder {
	if m.err != nil {
		return m
	}
	tmp, err := os.CreateTemp("", "telegram-image-*.bin")
	if err != nil {
		m.err = fmt.Errorf("创建临时文件失败: %w", err)
		return m
	}
	defer tmp.Close()

	if _, err := tmp.Write(data); err != nil {
		m.err = fmt.Errorf("写入临时图片失败: %w", err)
		return m
	}

	m.elements = append(m.elements, &botc.MessageElement{
		Type:   botc.ImageElement,
		Source: tmp.Name(),
	})
	return m
}

func (m *MessageBuilder) InlineKeyboard(kb *botc.InlineKeyboard) botc.MessageBuilder {
	if m.err != nil || kb == nil {
		return m
	}
	data, err := kb.ToJSON()
	if err != nil {
		m.err = fmt.Errorf("serialize inline keyboard: %w", err)
		return m
	}
	m.elements = append(m.elements, &botc.MessageElement{
		Type:    botc.InlineKeyboardElement,
		Content: data,
	})
	return m
}

func (m *MessageBuilder) ReplyTo(msgCtx botc.MessageContext) (*botc.BaseMessage, error) {
	if m.err != nil {
		return nil, m.err
	}

	if ctx, ok := msgCtx.(*command.Context); ok {
		msgCtx = ctx.MessageContext
	}

	msg := msgCtx.Message()
	if msg.MessageType == botc.GroupMessage && msg.Sender != nil && msg.Sender.From != nil {
		return m.service.SendGroupMessage(entity.Group{
			Base: &entity.Base{
				ID:   msg.Sender.From.ID,
				Name: msg.Sender.From.Name,
			},
		}, m.elements)
	}
	if msg.Sender != nil && msg.Sender.User != nil {
		return m.service.SendDirectMessage(entity.User{
			Base: &entity.Base{
				ID:   msg.Sender.User.ID,
				Name: msg.Sender.User.Name,
			},
		}, m.elements)
	}
	return nil, fmt.Errorf("无法确定回复目标")
}

func (m *MessageBuilder) Send(id string) (*botc.BaseMessage, error) {
	if m.err != nil {
		return nil, m.err
	}

	id = strings.TrimPrefix(id, "telegram:")

	chatID, err := ParseChatID(id)
	if err != nil {
		return nil, err
	}
	return m.service.sendToChat(chatID, m.elements)
}

// sendToChat 根据消息元素发送文本或图片消息
func (s *Service) sendToChat(chatID int64, elements []*botc.MessageElement) (*botc.BaseMessage, error) {
	text, isMarkdown := extractText(elements)
	photoSource := firstImageSource(s, elements)
	markup := extractReplyMarkup(elements)

	ctx := context.Background()

	if photoSource != "" {
		return s.sendPhotoWithMarkup(ctx, chatID, photoSource, text, markup, isMarkdown)
	}
	return s.sendTextWithMarkup(ctx, chatID, text, markup, isMarkdown)
}

func (s *Service) sendText(ctx context.Context, chatID int64, text string) (*botc.BaseMessage, error) {
	return s.sendTextWithMarkup(ctx, chatID, text, nil, false)
}

func (s *Service) sendTextWithMarkup(ctx context.Context, chatID int64, text string, markup models.ReplyMarkup, isMarkdown ...bool) (*botc.BaseMessage, error) {
	if text == "" {
		text = "(空消息)"
	}
	s.logger.Debug("Telegram 发送文本消息 >>> ChatID: %d, Text: %s", chatID, text)
	params := &bot.SendMessageParams{
		ChatID: chatID,
		Text:   text,
	}
	if len(isMarkdown) > 0 && isMarkdown[0] {
		params.ParseMode = models.ParseModeMarkdown
	}
	if markup != nil {
		params.ReplyMarkup = markup
	}
	msg, err := s.bot.SendMessage(ctx, params)
	if err != nil {
		s.logger.Debug("Telegram 发送文本消息失败: %v", err)
		return nil, err
	}
	return ParseMessage(msg, s), nil
}

func (s *Service) sendPhoto(ctx context.Context, chatID int64, source string, caption string) (*botc.BaseMessage, error) {
	return s.sendPhotoWithMarkup(ctx, chatID, source, caption, nil, false)
}

func (s *Service) sendPhotoWithMarkup(ctx context.Context, chatID int64, source string, caption string, markup models.ReplyMarkup, isMarkdown ...bool) (*botc.BaseMessage, error) {
	var photo models.InputFile

	if data, err := os.ReadFile(source); err == nil {
		s.logger.Debug("Telegram 读取本地图片文件成功: %s (%d 字节)", source, len(data))
		photo = &models.InputFileUpload{
			Filename: "image.jpg",
			Data:     bytes.NewReader(data),
		}
	} else {
		photo = &models.InputFileString{Data: source}
	}

	s.logger.Debug("Telegram 发送图片消息 >>> ChatID: %d, Source: %s", chatID, source)
	params := &bot.SendPhotoParams{
		ChatID:  chatID,
		Photo:   photo,
		Caption: caption,
	}
	if len(isMarkdown) > 0 && isMarkdown[0] {
		params.ParseMode = models.ParseModeMarkdown
	}
	if markup != nil {
		params.ReplyMarkup = markup
	}
	msg, err := s.bot.SendPhoto(ctx, params)
	if err != nil {
		return nil, err
	}
	return ParseMessage(msg, s), nil
}

func firstImageSource(s *Service, elements []*botc.MessageElement) string {
	for _, elem := range elements {
		if elem.Type != botc.ImageElement {
			continue
		}
		source := elem.Source
		if s.grb != nil {
			if p, err := s.grb.LoadResourceFromID(elem.Source); err == nil {
				source = p
			}
		}
		return strings.TrimPrefix(source, "file://")
	}
	return ""
}

func extractText(elements []*botc.MessageElement) (string, bool) {
	if len(elements) == 0 {
		return "", false
	}
	var builder strings.Builder
	isMarkdown := false
	for _, elem := range elements {
		switch elem.Type {
		case botc.TextElement, botc.MentionElement, botc.StickerElement, botc.QuoteElement:
			builder.WriteString(elem.Content)
		case botc.MarkdownElement:
			builder.WriteString(elem.Content)
			isMarkdown = true
		}
	}
	return builder.String(), isMarkdown
}

func extractReplyMarkup(elements []*botc.MessageElement) models.ReplyMarkup {
	for _, elem := range elements {
		if elem.Type != botc.InlineKeyboardElement || elem.Content == "" {
			continue
		}
		kb, err := botc.ParseInlineKeyboard(elem.Content)
		if err != nil || kb == nil || len(kb.Rows) == 0 {
			continue
		}

		var inlineRows [][]models.InlineKeyboardButton
		for _, row := range kb.Rows {
			var inlineRow []models.InlineKeyboardButton
			for _, btn := range row {
				tgBtn := models.InlineKeyboardButton{
					Text: btn.Text,
				}
				switch btn.Action {
				case botc.ActionURL:
					tgBtn.URL = btn.Data
				case botc.ActionCallback:
					tgBtn.CallbackData = btn.Data
				case botc.ActionCommand:
					// Telegram 内嵌键盘通过 SwitchInlineQueryCurrentChat 实现指令代填或回传
					data := btn.Data
					tgBtn.SwitchInlineQueryCurrentChat = &data
				}
				inlineRow = append(inlineRow, tgBtn)
			}
			inlineRows = append(inlineRows, inlineRow)
		}
		return &models.InlineKeyboardMarkup{
			InlineKeyboard: inlineRows,
		}
	}
	return nil
}
