# GoroBot
Go语言编写的跨平台聊天机器人框架

## [快速入门](https://jel1yspot.github.io/GoroBot/getting_started.html)

## 支持平台
- [x] QQ ([pkg/lagrange](https://github.com/Jel1ySpot/GoroBot/tree/master/pkg/lagrange))
- [x] OneBot
- [x] QQ Official ([pkg/qbot](https://github.com/Jel1ySpot/GoroBot/tree/master/pkg/qbot))
- [ ] Telegram

## 特性
- 高性能（我瞎说的反正 Go 怎么也比 Node 快）
- 高扩展性（插件化设计）
- 高自订性（插件想干啥都行.jpg）

## 能做什么
- 方便地编写自己的插件
- 所见即所得的命令格式
- 以通用的方法回复消息

## 注意事项
如果有能力，请**审查所有使用的代码！！！**
由于语言的特性，开发者拥有极高的权限，请在使用不可信来源的插件时多加注意！

## TODO
欢迎向本项目提交 Issue

### Roadmap
- [x] 完善指令系统

### Official Plugins
- [x] `ping`: bot 还在线吗？ping 一下看看
- [x] `message_logger`: 在控制台输出消息日志
- [x] `go_plugin`: 支持 go 风格热插拔式插件

### Bugs
- [ ] Onebot 适配器未稳定（只有反向Websocket工作正常）

