package bot_context

type MessageBuilder interface {
	Protocol() string
	Text(text string) MessageBuilder
	Quote(msg *BaseMessage) MessageBuilder
	Mention(id string) MessageBuilder
	ImageFromFile(path string) MessageBuilder
	ImageFromUrl(url string) MessageBuilder
	ImageFromData(data []byte) MessageBuilder
	ReplyTo(msg MessageContext) (*BaseMessage, error)
	Send(id string) (*BaseMessage, error)
}
