# 插件系统

GoroBot 的一切功能都是通过插件实现的，适配器也是插件。编写一个插件其实很简单，只需要实现三个方法就行。

## 插件骨架
每个插件都要实现 `Service` 接口：
```go
package myplugin

import GoroBot "github.com/Jel1ySpot/GoroBot/pkg/core"

type Service struct {
	bot         *GoroBot.Instant
	releaseFunc []func()
}

func (s *Service) Name() string {
	return "MyPlugin"
}

func Create() *Service {
	return &Service{}
}

func (s *Service) Init(grb *GoroBot.Instant) error {
	s.bot = grb
	// 在这里注册命令、事件监听器、中间件等
	return nil
}

func (s *Service) Release(grb *GoroBot.Instant) error {
	for _, fn := range s.releaseFunc {
		fn()
	}
	return nil
}
```

就这么几行代码，一个空的插件就写好了。接下来往 `Init` 里面塞东西就行。

## 生命周期
1. `Create()` — 创建插件实例。这时候框架还没启动，别在这里搞事情
2. `grb.Use(plugin)` — 注册插件，告诉框架"我要用这个"
3. `Init()` — 框架启动时调用。在这里注册命令、事件、中间件
4. `Release()` — 框架关闭时调用。清理资源，注销注册过的东西

## 注册命令
```go
func (s *Service) Init(grb *GoroBot.Instant) error {
	s.bot = grb

	delFn, _ := grb.Command("ping").
		Description("Ping 测试").
		Action(func(ctx *command.Context) error {
			_, _ = ctx.ReplyText("🏓")
			return nil
		}).
		Build()

	s.releaseFunc = append(s.releaseFunc, delFn)
	return nil
}
```

命令系统的详细用法参见 [命令系统](command.md)。

## 监听事件
```go
del, _ := grb.On(GoroBot.MessageEvent(func(ctx botc.MessageContext) {
	log.Info("收到消息: %s", ctx.String())
}))
s.releaseFunc = append(s.releaseFunc, del)
```

事件系统的详细用法参见 [事件系统](event.md)。

## 使用中间件
```go
del := grb.Middleware(func(msg botc.MessageContext, next func(...GoroBot.MiddlewareCallback) error) error {
	// 在消息到达事件和命令之前做点什么
	return next()
})
s.releaseFunc = append(s.releaseFunc, del)
```

中间件的详细用法参见 [中间件系统](middleware.md)。

## 构建消息
如果你需要回复图片、引用、@某人之类的复杂消息：
```go
_, _ = ctx.BotContext().NewMessageBuilder().
	Text("看看这张图").
	ImageFromFile("/path/to/image.png").
	ReplyTo(ctx)
```

`MessageBuilder` 支持的方法：
- `Text(text)` — 文字
- `ImageFromFile(path)` / `ImageFromUrl(url)` / `ImageFromData(bytes)` — 图片
- `Quote(baseMsg)` — 引用消息
- `Mention(userID)` — @某人
- `ReplyTo(msgCtx)` — 作为回复发送
- `Send(chatID)` — 发送到指定聊天

## 使用数据库
框架提供了一个可选的 SQLite 数据库：
```go
db := grb.Database() // *sql.DB，如果没有打开数据库就是 nil
if db != nil {
	// 用标准 database/sql 操作
}
```

## 清理资源
**很重要**：在 `Init` 里注册的东西，一定要在 `Release` 里清理掉。所有 `grb.On()`、`grb.Command().Build()`、`grb.Middleware()` 都会返回一个注销函数，把它们收集起来，在 `Release` 里逐个调用就行了。

## 文件组织
把插件放在自己的包里就好。简单的插件一个 `service.go` 搞定，复杂一点的可以拆成多个文件：
```
example_plugin/myplugin/
├── service.go    # 生命周期（Init / Release）
├── commands.go   # 命令注册
└── ...
```

> 示例插件在 [`example_plugin/`](https://github.com/Jel1ySpot/GoroBot/tree/master/example_plugin)，是最好的参考。
