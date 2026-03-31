# 命令系统

命令系统是 GoroBot 中处理用户指令的核心。它提供了一个链式 API 来注册命令，支持参数、选项、子命令和别名。

## 注册一个命令
最简单的命令长这样：
```go
import "github.com/Jel1ySpot/GoroBot/pkg/core/command"

delFn, _ := grb.Command("hello").
	Action(func(ctx *command.Context) error {
		_, _ = ctx.ReplyText("你好！")
		return nil
	}).
	Build()
```
用户发送 `/hello` 时，机器人会回复 "你好！"。`Build()` 返回一个注销函数，调用它可以移除这个命令。

## 命令描述
给命令加上描述，Telegram 适配器会自动同步到客户端的命令菜单：
```go
grb.Command("hello").
	Description("打个招呼").
	Action(handler).
	Build()
```

## 参数
参数是命令后面跟着的值，比如 `/echo 你好` 中的 "你好"：
```go
grb.Command("echo").
	Argument("content", command.String, true, "要回复的内容").
	Action(func(ctx *command.Context) error {
		_, _ = ctx.ReplyText(ctx.KvArgs["content"])
		return nil
	}).
	Build()
```
- 第一个参数是名称，在 `ctx.KvArgs` 中用它来取值
- 第二个参数是类型：`command.String`、`command.Number`、`command.Boolean`
- 第三个参数表示是否必填
- 第四个参数是帮助文本

取类型化的值可以用 `command.GetInt()`、`command.GetFloat()`。

## 选项
选项就是 `--flag` 或 `-f` 这种写法：
```go
grb.Command("search").
	Option("count", "n", command.Number, false, "10", "结果数量").
	Argument("keyword", command.String, true, "关键词").
	Action(func(ctx *command.Context) error {
		count := command.GetInt(ctx.KvArgs["count"])
		keyword := ctx.KvArgs["keyword"]
		// ...
		return nil
	}).
	Build()
```
用户可以这样用：`/search --count 5 golang` 或 `/search -n 5 golang`。

## 子命令
大的命令可以拆分成子命令：
```go
cmd := grb.Command("plugin")

cmd.SubCommand("list").
	Action(func(ctx *command.Context) error {
		// /plugin list
		return nil
	}).Build()

cmd.SubCommand("load").
	Argument("name", command.String, true, "插件名").
	Action(func(ctx *command.Context) error {
		// /plugin load myplugin
		return nil
	}).Build()

cmd.Build()
```
注意最后要调用根命令的 `Build()` 来完成注册。

## 别名
别名让命令可以通过正则表达式匹配触发。比如骰子插件用 `d6`、`d20` 这样的写法：
```go
grb.Command("dice").
	Argument("upper_bound", command.Number, false, "骰子点数上限").
	Alias(`^d(\d+)$`, func(ctx *command.Context) *command.Context {
		_ = ctx.AppendArg(ctx.String()[1:])
		return ctx
	}).
	Action(func(ctx *command.Context) error {
		limit := command.GetInt(ctx.KvArgs["upper_bound"])
		// ...
		return nil
	}).
	Build()
```
别名的第一个参数是正则表达式，第二个参数是一个转换函数，可以在里面用 `ctx.AppendArg()` 把匹配到的内容追加为参数。转换函数可以传 `nil`，这样匹配到后会直接触发命令。

## 命令上下文
`command.Context` 嵌入了 `botc.MessageContext`，所以消息上下文的方法都能用。额外提供了以下内容：
- `ctx.KvArgs` — 命名参数的键值对（`map[string]string`）
- `ctx.Arguments` — 位置参数列表（`[]string`）
- `ctx.Commands` — 匹配到的命令/子命令路径
- `ctx.ReplyText(...)` — 回复文本
- `ctx.Message()` — 获取原始消息

## 在插件中使用
在插件的 `Init` 中注册命令，在 `Release` 中清理：
```go
func (s *Service) Init(grb *GoroBot.Instant) error {
	s.bot = grb

	delFn, _ := grb.Command("mycommand").
		Description("我的命令").
		Action(func(ctx *command.Context) error {
			_, _ = ctx.ReplyText("收到！")
			return nil
		}).
		Build()

	s.releaseFunc = append(s.releaseFunc, delFn)
	return nil
}
```

> 更完整的示例可以参考 [`example_plugin/`](https://github.com/Jel1ySpot/GoroBot/tree/master/example_plugin) 中的代码。
