package message

type Element struct {
	Type    ElementType
	Content string
	Source  string
}

type ElementType int

const (
	TextElement ElementType = iota
	QuoteElement
	MentionElement
	ImageElement
	VideoElement
	FileElement
	VoiceElement
	StickerElement
	LinkElement
	OtherElement
)
