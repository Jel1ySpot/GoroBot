# message
message 包位于 pkg/core/message ，包含消息示例和上下文结构。

## message.Context
消息的上下文，由适配器在触发消息事件时提供。

### Context.Protocol() string
消息的协议，与 BotContext.Protocol() 相同，值可以为机器人所处的平台或所使用的协议。

### Context.String() string
返回消息的文本预览。

### Context.Message() *message.Base
返回消息结构。

### Reply(message []*message.Element) error
在当前上下文中回复消息。具体实现由适配器决定。

### ReplyText(text string) error
使用 `Reply()` 回复纯文本消息的便捷方法。

## message.Base
消息的基本结构。含有以下属性：
- MessageType `message.Type` 消息类型
- ID `string` 消息 ID
- Content `string` 显示文本
- Elements `[]*Element` 消息元素
- Sender `*entity.Sender` 消息发送者
- Time `time.Time` 消息发送(接收)时间

### *Base.Marshall() string
序列化消息实体，使其能被保存以及使用 `message.UnmarshallMessage(string)` 读取。

## message.Element
消息元素，消息的基本组成部分。
- Type `message.ElementType` 元素类型
- Content `string` 显示文本
- Source `string` 多媒体索引，在资源一节中会提到

### message.Text
- Content: `显示文本`
- Source: `""`

### message.Quote
- Content: `[回复]`
- Source: `被引用消息的序列化对象`

### message.Mention
- Content: `@Somebody`
- Source: `protocol:被提及用户的ID`

### message.Image
- Content: `[图片]`
- Source: `资源ID`

### message.Video
- Content: `[视频]`
- Source: `资源ID`

### message.File
- Content: `[文件]`
- Source: `protocol:参数...`

### message.Voice
- Content: `[语音]`
- Source: `资源ID`

### message.Sticker
- Content: `[贴纸]（[表情]）`
- Source: `protocol:参数...`

### message.Link
- Content: `百度一下，你就知道`
- Source: `https://www.baidu.com`

### message.Other
- Content: `奇怪的东西`
- Source: `protocol:参数`

## message.Builder
构建消息元素链。

### message.NewBuilder() *Builder
创建构建器。

### *Builder.Append(elementType ElementType, content string, source string) *Builder
在消息链中添加元素。

### *Builder.Build() []*message.Element
返回构造完成的元素链。
