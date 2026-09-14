# 配置文件

GoroBot 采用模块化的配置管理方案。核心框架与各个适配器、插件独立管理各自的配置文件。配置管理基于 [conic](https://github.com/Jel1ySpot/conic) 库，支持 JSON / YAML 等格式、结构体动态绑定与配置热重载（WatchConfig）。

首次运行机器人时，各模块会在 `conf/` 目录下自动生成默认配置文件。

## 目录结构

典型的 `conf/` 配置目录结构如下：

```
conf/
├── config.json            # 框架核心主配置文件
├── qbot/                  # QQ 官方机器人适配器配置
│   └── config.yaml
├── onebot/                # OneBot 适配器配置
│   └── config.json
├── lagrange/              # Lagrange 适配器配置
│   └── config.json
└── telegram/              # Telegram 适配器配置
    └── config.json
```

---

## 框架核心配置文件 (`conf/config.json`)

主配置文件控制框架的全局行为，如日志输出级别、机器人所有者（管理员）映射以及资源文件路径等。

### 示例配置

```json
{
  "log_level": 1,
  "owner": {
    "qq": "100000000",
    "qbot:<your_bot_appid>": "qbot:user&<your_user_openid>",
    "telegram:<your_bot_id>": "telegram:user&<your_user_id>"
  },
  "admin": {
    "qq": ["100000001", "100000002"],
    "telegram:<your_bot_id>": ["telegram:user&<admin_user_id>"]
  },
  "resource_path": "./data/resources"
}
```

### 字段说明

| 字段 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| `log_level` | int | `1` | 全局日志输出级别，数值越大输出越详细（见下表） |
| `owner` | object | `{}` | 机器人最高所有者（等级 5）映射，键为机器人上下文 ID 或平台标识，值为 Universal ID 或用户 ID |
| `admin` | object | `{}` | 机器人管理员（等级 4）映射，键为上下文 ID 或平台标识，值为用户 ID（支持单字符串或字符串列表） |
| `resource_path` | string | `""` | 资源文件保存根路径（可选） |

### 日志级别 (`log_level`) 说明

| 数值 | 等级名称 | 说明 |
|:----:|---------|------|
| `-3` | Fatal | 仅输出致命错误 |
| `-2` | Error | 输出错误信息及以上 |
| `-1` | Announcement / Success | 输出成功与公告信息及以上 |
| `0` | Warning | 输出警告信息及以上 |
| `1` | Info | **默认级别**，输出常规运行信息及以上 |
| `2` | Debug | 输出详细调试日志及以上（包含接收/发送数据调试） |

---

## 适配器配置文件

### 1. QQ 官方机器人 (`conf/qbot/config.yaml`)

QBot 适配器用于接入 QQ 开放平台官方机器人。默认使用 WebSocket Gateway 长连接，也支持 Webhook 回调或双向并存模式。

#### 示例配置

```yaml
debug: false
mode: "websocket" # 连接模式: "websocket" (默认), "webhook", 或 "both"
intents: 0        # 订阅事件掩码，留空或 0 则使用框架内置默认值

api:
  appid: "your_app_id"
  secret: "your_secret_key"

# 可选：仅在 mode 为 "webhook" 或 "both"，或者需要对外提供图片等静态资源服务时配置
webhook:
  host: "0.0.0.0"
  port: 8443
  path: "/bot"
  base_url: "https://your.domain.net:8443"
  tls:
    cert_path: "/path/to/server.crt"
    key_path: "/path/to/server.key"
```

#### 字段说明

| 字段 | 类型 | 说明 |
|------|------|------|
| `debug` | bool | 是否启用 QBot 详细调试日志 |
| `mode` | string | 连接模式：`"websocket"`（推荐）、`"webhook"` 或 `"both"` |
| `intents` | int | 事件订阅 Intents 掩码，传 `0` 默认订阅群聊、频道和私聊常用事件 |
| `api.appid` | string | QQ 开放平台的机器人 AppID |
| `api.secret` | string | QQ 开放平台的机器人 AppSecret。**注：若留空且在终端运行，QBot 会在终端展示登录二维码，手机 QQ 扫码后自动获取凭证并加密写回配置文件** |
| `webhook.host` | string | Webhook 监听地址（默认 `0.0.0.0`） |
| `webhook.port` | uint | Webhook 监听端口（如 `8443`） |
| `webhook.path` | string | Webhook 回调 URL 路径（如 `/bot`） |
| `webhook.base_url` | string | 机器人的公网访问基础地址，用于回调校验及静态媒体回源 |
| `webhook.tls.cert_path` | string | TLS/SSL 证书文件路径（可选，若配置则开启 HTTPS） |
| `webhook.tls.key_path` | string | TLS/SSL 私钥文件路径 |

---

### 2. OneBot 适配器 (`conf/onebot/config.json`)

OneBot 适配器支持 OneBot v11 协议标准。支持 HTTP、WebSocket 客户端（正向 WS）以及反向 WebSocket（ws_reverse）三种通信模式。

#### 示例配置（正向 WebSocket 模式）

```json
{
  "mode": "ws",
  "ws": {
    "host": "127.0.0.1",
    "port": 3001,
    "access_token": "your_token"
  },
  "message_format": "array",
  "heartbeat": {
    "enable": true,
    "interval": 30000
  },
  "ignore_self": true,
  "command_prefix": "/",
  "debug": false
}
```

#### 示例配置（反向 WebSocket 模式）

```json
{
  "mode": "ws_reverse",
  "ws_reverse": {
    "host": "0.0.0.0",
    "port": 8080,
    "path": "/onebot/v11/ws",
    "access_token": "",
    "reconnect_interval": 300
  },
  "message_format": "array",
  "ignore_self": true,
  "command_prefix": "/"
}
```

#### 示例配置（HTTP 模式）

```json
{
  "mode": "http",
  "http": {
    "host": "127.0.0.1",
    "port": 5700,
    "post_url": "http://127.0.0.1:5701",
    "access_token": "",
    "secret": "",
    "timeout": 30
  },
  "message_format": "array",
  "ignore_self": true,
  "command_prefix": "/"
}
```

#### 字段说明

| 字段 | 类型 | 说明 |
|------|------|------|
| `mode` | string | **必填**。连接模式：`"ws"`、`"ws_reverse"` 或 `"http"` |
| `ws` | object | 正向 WebSocket 客户端配置（当 `mode` 为 `"ws"` 时必填） |
| `ws.host` | string | OneBot 服务端的主机地址 |
| `ws.port` | int | OneBot 服务端的 WebSocket 端口 |
| `ws.access_token` | string | 鉴权令牌（可选） |
| `ws_reverse` | object | 反向 WebSocket 服务端配置（当 `mode` 为 `"ws_reverse"` 时必填） |
| `ws_reverse.host` | string | 本地监听地址 |
| `ws_reverse.port` | int | 本地监听端口 |
| `ws_reverse.path` | string | WebSocket 路径（如 `"/onebot/v11/ws"`） |
| `ws_reverse.access_token` | string | 客户端连接时的鉴权 Token |
| `ws_reverse.reconnect_interval` | int | 重连超时时间（毫秒，默认 300） |
| `http` | object | HTTP 交互模式配置（当 `mode` 为 `"http"` 时必填） |
| `http.host` | string | HTTP 服务端地址 |
| `http.port` | int | HTTP 服务端端口 |
| `http.post_url` | string | 事件上报地址 |
| `http.access_token` | string | 鉴权 Token |
| `http.secret` | string | HTTP 上报事件签名密钥 |
| `http.timeout` | int | HTTP 请求超时时间（秒，默认 30） |
| `message_format` | string | 消息格式：`"array"`（推荐，CQ 码数组）或 `"string"`（纯字符串） |
| `heartbeat.enable` | bool | 是否启用心跳保活机制（默认 `true`） |
| `heartbeat.interval` | int | 心跳上报间隔（毫秒，默认 `30000`） |
| `rate_limit.enable` | bool | 是否开启 API 频率限制（默认 `false`） |
| `rate_limit.interval` | int | 限速间隔（毫秒） |
| `ignore_self` | bool | 是否忽略自身发送的消息（默认 `true`） |
| `command_prefix` | string | 命令前缀（默认 `"/"`） |
| `debug` | bool | 是否输出 OneBot 调试日志 |

---

### 3. Lagrange 适配器 (`conf/lagrange/config.json`)

Lagrange 适配器基于 LagrangeGo 实现 QQ 客户端协议支持。

#### 示例配置

```json
{
  "app_info": "linux 9.0.90",
  "sign_server_url": "https://sign.example.com",
  "music_sign_server_url": "",
  "command_prefix": "/",
  "account": {
    "uin": 100000000,
    "password": "",
    "sig_path": "keystore.json"
  },
  "ignore_self": true
}
```

#### 字段说明

| 字段 | 类型 | 说明 |
|------|------|------|
| `app_info` | string | 模拟客户端协议版本信息（如 `"linux 9.0.90"`） |
| `sign_server_url` | string | Lagrange 签名服务器地址 |
| `music_sign_server_url` | string | 音乐卡片签名服务地址（可选） |
| `command_prefix` | string | 命令前缀（默认 `"/"`） |
| `account.uin` | uint32 | QQ 账号 |
| `account.password` | string | QQ 密码（支持留空并通过扫码验证登录） |
| `account.sig_path` | string | 登录凭证保存相对路径（保存在 `conf/lagrange/` 目录下） |
| `ignore_self` | bool | 是否忽略自身消息 |

---

### 4. Telegram 适配器 (`conf/telegram/config.json`)

Telegram 适配器用于接入 Telegram Bot API。

#### 示例配置

```json
{
  "token": "your_bot_token_from_botfather",
  "server_url": ""
}
```

#### 字段说明

| 字段 | 类型 | 说明 |
|------|------|------|
| `token` | string | **必填**。Telegram Bot Token，可向 [@BotFather](https://t.me/botfather) 创建获取 |
| `server_url` | string | 自定义 Telegram API 地址（留空则使用官方默认 `https://api.telegram.org`，适合配置本地代理或反代加速） |

---

## Universal ID (统一实体标识) 说明

为了统一跨平台的用户、群组与消息寻址，GoroBot 采用了 Universal ID 机制。

格式定义为：
```
protocol:type&arg1&arg2...
```

### 常见格式示例

| 平台 | 实体类型 | Universal ID 示例 |
|------|---------|------------------|
| **QBot** | 用户 | `qbot:user&<openid>` |
| **QBot** | 群聊 | `qbot:group&<group_openid>` |
| **QBot** | 频道 | `qbot:guild&<guild_id>` |
| **Lagrange / OneBot** | 用户 | `lagrange:user&<uin>` / `onebot:user&<uin>` |
| **Lagrange / OneBot** | 群聊 | `lagrange:group&<group_id>` / `onebot:group&<group_id>` |
| **Telegram** | 用户 / 群聊 | `telegram:user&<chat_id>` / `telegram:group&<chat_id>` |

在 `conf/config.json` 的 `owner` 配置中，可通过配置该 ID 赋予特定用户管理员权限。

---

## 在自定义插件中管理配置

开发者编写插件时，也可以方便地利用 `conic` 库为自己的插件提供独立的配置文件：

```go
package myplugin

import (
	"path"
	"github.com/Jel1ySpot/conic"
	GoroBot "github.com/Jel1ySpot/GoroBot/pkg/core"
	"github.com/Jel1ySpot/GoroBot/pkg/util"
)

type Config struct {
	Greeting string `json:"greeting"`
	Limit    int    `json:"limit"`
}

type Service struct {
	conic  *conic.Conic
	config Config
}

func (s *Service) Init(grb *GoroBot.Instant) error {
	configDir := "conf/myplugin/"
	configFile := path.Join(configDir, "config.json")

	s.conic = conic.New()
	s.conic.SetConfigFile(configFile)
	s.conic.WatchConfig()
	s.conic.BindRef("", &s.config)

	if !util.FileExists(configFile) {
		_ = util.MkdirIfNotExists(configDir)
		s.config = Config{Greeting: "你好", Limit: 10}
		_ = s.conic.WriteConfig() // 自动写出默认配置
	}

	return s.conic.ReadConfig()
}
```
