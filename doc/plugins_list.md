# 插件列表

欢迎在 issue 中提供你所写的插件信息。信息将会在维护者查看后加入列表。

## 注意事项
重申：如果有能力，请**审查所有使用的代码！！！**
由于语言的特性，开发者拥有极高的权限，请在使用不可信来源的插件时多加注意！

恶意插件可以做到的事情有：
- 向恶意者发送消息
- 发送恶意消息
- 向恶意者传送账号信息以盗取账号
- 盗取你的本机账号

## 插件
### 消息日志
> `import "github.com/Jel1ySpot/GoroBot/example_plugin/message_logger"`

基础的消息事件监听。将消息预览打印到日志。

### Ping
> `import "github.com/Jel1ySpot/GoroBot/example_plugin/ping`

检测机器人连接状态，和一些调试命令

### 插件载入
> `import "github.com/Jel1ySpot/GoroBot/example_plugin/go_plugin`

载入外部插件（限类 Unix 系统）
