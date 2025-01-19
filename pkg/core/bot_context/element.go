package bot_context

type MessageElement struct {
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

func ElemsToString(elems []*MessageElement) string {
	content := ""
	for _, elem := range elems {
		content += elem.Content
	}
	return content
}
