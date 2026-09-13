# pkg/qbot
用于 [Jel1ySpot/GoroBot](https://github.com/Jel1ySpot/GoroBot) 的 QQ 开放平台官方机器人适配器。
参考官方 [tencent-connect/openclaw-qqbot](https://github.com/tencent-connect/openclaw-qqbot) 规范重构，支持 Webhook 回调（带 Ed25519 签名验证与 op:13 校验）及 WebSocket Gateway 两种连接模式。

## 快速开始
1. 载入包
    - `import "github.com/Jel1ySpot/GoroBot/pkg/qbot"`
2. 创建实例并使用
    - `grb.Use(qbot.Create())`
3. [填写配置文件](#填写配置文件)

### 填写配置文件
首次运行后，会在 `conf/qbot` 目录下创建空配置文件。配置示例：
```yaml
debug: false
mode: "websocket" # 连接模式: "websocket" (默认), "webhook", 或 "both"
api:
   appid: "your app id"
   secret: "your secret key"

# 可选：仅在 mode 为 "webhook" 或需对外提供静态资源下载服务时需要配置
# webhook:
#    host: "0.0.0.0"
#    path: "/bot"
#    port: 8443
#    base_url: "http://your.domain.net:8443"
#    tls:
#       cert_path: "/path/to/server.crt"
#       key_path: "/path/to/server.key"
```
