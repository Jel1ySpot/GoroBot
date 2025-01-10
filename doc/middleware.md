# 中间件
中间件使用 `grb.Middleware(MiddlewareCallback)` 注册。跟事件一样，这个方法返回一个函数用于注销该中间件。`MiddlewareCallbakc` 的签名为 `func(bot BotContext, msg message.Context, next func(...MiddlewareCallback) error) error`，跟消息事件不同的是，中间件的回调函数参数中有一个额外的参数 `next`，只有调用了它才会进入接下来的流程。

### 使用中间件实现的 ping
```go
grb.Middleware(func(bot BotContext, msg message.Context, next func(...MiddlewareCallback) error) error {
	if msg.String() == "ping" {
		msg.ReplyText("🏓")
	}
	next()
})
```


