package GoroBot

import "github.com/Jel1ySpot/GoroBot/pkg/core/message"

type MessageBuilder interface {
	Protocol() string
	Text(text string) MessageBuilder
	Quote(msg *message.Base) MessageBuilder
	Mention(id string) MessageBuilder
	ImageFromFile(path string) MessageBuilder
	ImageFromUrl(url string) MessageBuilder
	ImageFromData(data []byte) MessageBuilder
	ReplyTo(msg message.Context) error
	Send(ctx BotContext, messageType message.Type, id string) error
}
