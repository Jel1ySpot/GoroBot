# 消息类型
消息相关的类型位于 `pkg/core/bot_context`，包含消息上下文和消息结构。

## MessageContext
消息的上下文，由适配器在触发消息事件时提供。

### ctx.Protocol() string
消息的协议，值可以为机器人所处的平台或所使用的协议。

### ctx.BotContext() BotContext
获取所在平台的适配器上下文。

### ctx.String() string
返回消息的文本预览。

### ctx.Message() *BaseMessage
返回消息结构。

### ctx.SenderID() string
返回发送者 ID。

### ctx.NewMessageBuilder() MessageBuilder
创建消息构建器。

### ctx.Reply(elements []*MessageElement) (*BaseMessage, error)
在当前上下文中回复消息。具体实现由适配器决定。

### ctx.ReplyText(a ...any) (*BaseMessage, error)
回复纯文本消息的便捷方法。

## BaseMessage
消息的基本结构。含有以下属性：
- MessageType `MessageType` 消息类型（`DirectMessage` / `GroupMessage`）
- ID `string` 消息 ID
- Content `string` 显示文本
- Elements `[]*MessageElement` 消息元素
- Sender `*entity.Sender` 消息发送者
- Time `time.Time` 消息发送(接收)时间

### *BaseMessage.Marshall() string
序列化消息实体，使其能被保存以及使用 `UnmarshallMessage(string)` 读取。

## MessageElement
消息元素，消息的基本组成部分。
- Type `ElementType` 元素类型
- Content `string` 显示文本
- Source `string` 多媒体索引

### TextElement
- Content: `显示文本`
- Source: `""`

### QuoteElement
- Content: `[回复]`
- Source: `被引用消息的序列化对象`

### MentionElement
- Content: `@Somebody`
- Source: `protocol:被提及用户的ID`

### ImageElement
- Content: `[图片]`
- Source: `资源ID`

### VideoElement
- Content: `[视频]`
- Source: `资源ID`

### FileElement
- Content: `[文件]`
- Source: `protocol:参数...`

### VoiceElement
- Content: `[语音]`
- Source: `资源ID`

### StickerElement
- Content: `[表情]`
- Source: `protocol:参数...`

### LinkElement
- Content: `百度一下，你就知道`
- Source: `https://www.baidu.com`

### OtherElement
- Content: `奇怪的东西`
- Source: `protocol:参数`

## MessageBuilder
构建和发送消息的链式 API。通过 `ctx.NewMessageBuilder()` 或 `ctx.BotContext().NewMessageBuilder()` 创建。

### builder.Text(text string) MessageBuilder
添加文本内容。

### builder.ImageFromFile(path string) MessageBuilder
从本地文件添加图片。

### builder.ImageFromUrl(url string) MessageBuilder
从 URL 添加图片。

### builder.ImageFromData(data []byte) MessageBuilder
从二进制数据添加图片。

### builder.Quote(msg *BaseMessage) MessageBuilder
引用一条消息。

### builder.Mention(id string) MessageBuilder
@某人。

### builder.ReplyTo(msg MessageContext) (*BaseMessage, error)
作为回复发送到消息来源。

### builder.Send(id string) (*BaseMessage, error)
发送到指定的聊天 ID。

## BaseBuilder
构建消息元素链的工具，用于适配器内部解析消息。

### NewBuilder() *BaseBuilder
创建构建器。

### *BaseBuilder.Append(elementType ElementType, content string, source string) *BaseBuilder
在消息链中添加元素。

### *BaseBuilder.Build() []*MessageElement
返回构造完成的元素链。
