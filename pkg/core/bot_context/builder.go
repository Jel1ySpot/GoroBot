package bot_context

import (
	"fmt"
)

type BaseBuilder struct {
	elements []*MessageElement
}

func (b *BaseBuilder) Text(text string) *BaseBuilder {
	return b.Append(TextElement, text, "")
}

func (b *BaseBuilder) Quote(msg *BaseMessage) *BaseBuilder {
	return b.Append(QuoteElement, "[回复]", msg.Marshall())
}

func (b *BaseBuilder) Mention(id string) *BaseBuilder {
	return b.Append(MentionElement, fmt.Sprintf("@%s", id), id)
}

func NewBuilder() *BaseBuilder {
	return &BaseBuilder{}
}

func (b *BaseBuilder) Append(elementType ElementType, content string, source string) *BaseBuilder {
	b.elements = append(b.elements, &MessageElement{
		Type:    elementType,
		Content: content,
		Source:  source,
	})
	return b
}

func (b *BaseBuilder) Build() []*MessageElement {
	return b.elements
}
