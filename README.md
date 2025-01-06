# GoroBot
Go语言编写的跨平台聊天机器人框架

## 支持平台
- [x] QQ ([pkg/lagrange](https://github.com/Jel1ySpot/GoroBot/tree/master/pkg/lagrange))
- [ ] Telegram

## 快速入门

```go
package main // main.go

import (
	"github.com/Jel1ySpot/GoroBot/example_plugin/message_logger"
	GoroBot "github.com/Jel1ySpot/GoroBot/pkg/core"
	"github.com/Jel1ySpot/GoroBot/pkg/core/bot"
	"github.com/Jel1ySpot/GoroBot/pkg/core/message"
	LgrClient "github.com/Jel1ySpot/GoroBot/pkg/lagrange"
)

func main() {
	grb := GoroBot.Create()

	lgr := LgrClient.Create()

	grb.Use(lgr)
	grb.Use(message_logger.Create())

	var del func()
	del, _ = grb.On(GoroBot.MessageEvent(func(ctx bot.Context, msg message.Context) error {
		if msg.String() == "ping" {
			_ = msg.ReplyText("🏓")
			del()
		}
		return nil
	}))

	if err := grb.Run(); err != nil {
		panic(err)
	}
}

```

这段代码中首先创建了一个 `GoroBot` 实例，之后创建了 `Lagrange` 与 `MessageLogger` 实例，并在 `GoroBot` 实例中调用。
然后在主函数中监听了一个消息事件，它返回一个用于删除自身实例的函数，并在实例被触发后调用，实现了一个单次事件监听器。
在函数的末尾则调用了 `GoroBot` 实例的 `Run` 方法以启动实例。

执行这段代码，在正常运行之前会让用户填写配置文件。

通过这段代码可以发现：

- 插件等实例使用 `Create` 函数创建
- 使用 `Use` 方法调用实例
- 使用 `On` 方法监听事件

更多用法可以在 `example_plugin` 中查看