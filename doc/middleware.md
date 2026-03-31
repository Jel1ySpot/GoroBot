# 中间件
中间件使用 `grb.Middleware(MiddlewareCallback)` 注册。跟事件一样，这个方法返回一个函数用于注销该中间件。

`MiddlewareCallback` 的签名为：
```go
func(msg botc.MessageContext, next func(...MiddlewareCallback) error) error
```

跟消息事件不同的是，中间件的回调函数参数中有一个额外的参数 `next`，只有调用了它才会进入接下来的流程。

### 使用中间件实现的 ping
```go
grb.Middleware(func(msg botc.MessageContext, next func(...GoroBot.MiddlewareCallback) error) error {
	if msg.String() == "ping" {
		_, _ = msg.ReplyText("🏓")
	}
	return next()
})
```

### prepare 模式
传入第二个参数 `true` 可以注册一个 prepare 中间件，它会在所有普通中间件之前执行：
```go
grb.Middleware(func(msg botc.MessageContext, next func(...GoroBot.MiddlewareCallback) error) error {
	// 这段逻辑会在所有普通中间件之前执行
	return next()
}, true)
```
