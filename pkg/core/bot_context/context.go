package bot_context

type MessageContext interface {
	FeatureProvider
	Protocol() string
	BotContext() BotContext
	String() string
	Message() *BaseMessage
	SenderID() string
	NewMessageBuilder() MessageBuilder
	Reply(elements []*MessageElement) (*BaseMessage, error)
	ReplyText(a ...any) (*BaseMessage, error)
}
