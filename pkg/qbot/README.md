# pkg/lagrange
用于 [Jel1ySpot/GoroBot](https://github.com/Jel1ySpot/GoroBot) 的基于 [tencent-connect/botgo](https://github.com/tencent-connect/botgo) 的 QQBot 适配器
支持 Webhook 和基本的消息收发。

## 快速开始
1. 载入包
    - `import "github.com/Jel1ySpot/GoroBot/pkg/qbot"`
2. 创建实例并使用
    - `grb.Use(qbot.Create())`
3. [填写配置文件](#填写配置文件)

### 填写配置文件
首次运行后，会在 `conf/qbot` 目录下创建空配置文件。 配置示例：
```yaml
debug: false # botgo 库的 Debug 开关
api:
   appid: "your app id"
   secret: "your secret key"
http:
   host: "0.0.0.0"
   path: "/bot"
   port: 8443
   base_url: "http://your.domain.net:8443"
   tls:
      cert_path: "/path/to/server.crt"
      key_path: "/path/to/server.key"
```
