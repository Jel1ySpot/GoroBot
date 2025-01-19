package bot_context

type MessageContext interface {
	Protocol() string
	BotContext() BotContext
	String() string
	Message() *BaseMessage
	NewMessageBuilder() MessageBuilder
	Reply(elements []*MessageElement) (*BaseMessage, error)
	ReplyText(text string) (*BaseMessage, error)
}
