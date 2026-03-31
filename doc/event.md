# 事件系统

GoroBot 通过事件系统实现了消息监听等功能。这一设计允许插件开发者和用户编写简洁、通用的代码来扩展和实现各种功能。

## 以消息事件举例
以下示例代码展示了如何使用事件系统注册和注销一个监听器：
```go
import botc "github.com/Jel1ySpot/GoroBot/pkg/core/bot_context"

var del func()
del, _ = grb.On(GoroBot.MessageEvent(func(ctx botc.MessageContext) {
	if ctx.String() == "ping" {
		_, _ = ctx.ReplyText("🏓")
		del()
	}
}))
```

在这段代码中，通过调用 `grb.On` 方法，注册了一个监听消息事件的监听器。监听器的回调函数接收一个 `ctx`（消息内容上下文）参数。当接收到的消息内容为 "ping" 时，调用 `ctx.ReplyText` 方法回复 "🏓"，并通过 `del` 函数注销该监听器，从而实现了单次响应特定消息的功能。

`MessageEvent()` 函数接受一个回调函数作为参数，并返回一个 `Name` 属性为 `"message"` 的 `EventHandler` 实例。框架通过将用户提供的回调函数转换为 `func(args ...any) error` 的通用签名形式，简化了事件注册的流程。这种设计使开发者无需关心底层实现的细节，便可直观地使用框架或插件提供的事件注册器。

## 常用的消息上下文方法
- `ctx.String()` — 消息文本预览
- `ctx.Message()` — 获取完整消息结构（`*botc.BaseMessage`）
- `ctx.Protocol()` — 消息来源平台（`"lagrange"`、`"telegram"` 等）
- `ctx.SenderID()` — 发送者 ID
- `ctx.ReplyText(...)` — 回复纯文本
- `ctx.Reply(elements)` — 回复消息元素
- `ctx.BotContext()` — 获取所在平台的适配器上下文

> 如果你需要的是对特定格式的消息做出回复（比如 `/command arg1 arg2`），那你应该看看[命令系统](command.md)。
