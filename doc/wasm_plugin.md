# Wasm 插件系统

`pkg/wasm_plugin` 是 GoroBot 基于 [Extism](https://extism.org/)（底层为纯 Go 的 [wazero](https://github.com/tetratelabs/wazero)）构建的 WebAssembly 沙箱插件引擎。

相比于原生的 Go `.so` 动态库，Wasm 插件具备以下优势：
- **多语言开发**：支持使用 Rust、Go (TinyGo)、C/C++、Zig、AssemblyScript、TypeScript 等几乎所有能编译为 Wasm 的语言编写插件。
- **真正的热重载与完全卸载**：Wasm 模块实例可在不重启主程序的情况下随时创建与销毁，内存立即释放，彻底解决 Go 原生 `plugin` 无法卸载的问题。
- **强安全沙箱隔离**：插件运行在严格受控的沙箱环境中，默认无法访问未授权的主机文件与系统调用。
- **纯 Go 零 CGO 依赖**：全平台通用（Windows、Linux、macOS 均可编译运行）。

---

## 快速启用

在你的入口文件（如 `main.go`）中注册 `wasm_plugin` 服务：

```go
package main

import (
	GoroBot "github.com/Jel1ySpot/GoroBot/pkg/core"
	"github.com/Jel1ySpot/GoroBot/pkg/wasm_plugin"
)

func main() {
	grb := GoroBot.Create()

	// 注册 Wasm 插件管理器
	grb.Use(wasm_plugin.Create())

	if err := grb.Run(); err != nil {
		panic(err)
	}
}
```

---

## 插件存放与数据隔离

### 1. 插件搜索目录
默认在运行目录下的 `plugin/` 中搜索 `.wasm` 文件。
- `plugin/dice.wasm` 对应的插件 ID 为 `dice`。
- `plugin/games/guess.wasm` 对应的插件 ID 为 `games/guess`。

### 2. 独立数据目录（文件系统沙箱）
每个插件拥有专属的数据存储目录：`data/<plugin_id>/`。
- 宿主自动为每个插件将该目录挂载至容器内部的 `/` 与 `/data`。
- 插件在 Wasm 代码中执行标准文件操作（如 Rust 的 `std::fs` 或 TinyGo 的 `os`）时，所有读写均被严格限制在 `data/<plugin_id>/` 内，无法越权访问宿主系统的其他文件。

---

## 消息管理指令

管理指令默认仅机器人的 `Owner` 拥有执行权限（采用 Fail-Closed 默认拒绝鉴权）。

| 指令 | 说明 |
| :--- | :--- |
| `/wasm lookup` | 扫描 `plugin/` 目录，发现新增的 `.wasm` 插件 |
| `/wasm list` | 列出所有已识别的插件及其运行状态（✅ 已启用 / ❎ 未启用） |
| `/wasm load <id\|all>` | 加载或重新加载指定的插件（或全部插件），实现即时热重载 |
| `/wasm enable <id\|all>` | 启用已发现但尚未启用的插件 |
| `/wasm disable <id\|all>` | 禁用指定插件，注销所有命令、关闭监听端口并彻底释放内存 |

---

## 宿主 API (Host Functions)

Wasm 插件可以通过导入函数与 GoroBot 宿主进行双向交互。接口同时挂载在 `extism:host/user`（Extism PDK 默认命名空间）和 `gorobot` 命名空间下。

### 1. 日志输出
- **函数名**：`gorobot_log(level, msg)`
- **说明**：接入 GoroBot 统一日志系统输出。`level` 可为 `debug`、`info`、`warn`、`error`。

### 2. 注册消息指令
- **函数名**：`gorobot_register_command(def_json)`
- **说明**：向 GoroBot 注册机器人的消息指令。
- **入参格式**：
  ```json
  {
    "name": "dice",
    "description": "掷骰子插件",
    "arguments": [
      {"name": "max", "type": "number", "required": false, "help": "最大点数"}
    ],
    "options": [
      {"name": "times", "short": "t", "type": "number", "required": false, "help": "投掷次数"}
    ],
    "handler": "on_command" // 触发时回调的 Wasm 导出函数名，默认 "on_command"
  }
  ```
- **回调上下文**：指令触发时，宿主将包含用户输入、群聊/私聊类型与环境信息的 `CommandEvent` JSON 传给插件。若插件的导出函数直接返回非空文本，宿主会自动将其作为回复发送给用户。
  ```json
  {
    "command": "dice",
    "message_type": "group",       // "group"（群聊）或 "direct"（私聊）
    "group_id": "group_12345",     // 群聊 ID（私聊为空）
    "group": {                     // 群聊信息
      "id": "group_12345",
      "name": "交流群"
    },
    "sender_id": "user_67890",     // 发送者 ID
    "sender": {                    // 发送者详情
      "id": "user_67890",
      "name": "张三",
      "nickname": "三哥",
      "authority": 1               // 0=Banned, 1=Member, 2=GroupAdmin, 3=GroupOwner, 4=Admin, 5=Owner
    },
    "protocol": "qbot",            // 协议平台（qbot, telegram, onebot 等）
    "features": ["text", "image", "inline_keyboard"],
    "message_id": "msg_abc",       // 消息唯一 ID
    "raw": "/dice 6",              // 完整输入原始文本
    "context_token": "..."         // 上下文 Token，用于回传
  }
  ```

### 3. 消息回复与主动发送（支持内嵌键盘）
- **回复文本消息**：`gorobot_reply_text(context_token, text)`
  - 在指令或事件处理上下文中使用收到的 `context_token` 进行即时纯文本回复。
- **回复富消息（含内嵌键盘）**：`gorobot_reply_message(req_json)`
  - 支持向当前上下文回复携带内嵌按钮（Inline Keyboard）的消息。
  ```json
  {
    "context_token": "...",
    "text": "请选择你要执行的操作：",
    "keyboard": {
      "rows": [
        [
          { "text": "打开官网", "action": 0, "data": "https://example.com" },
          { "text": "点击签到", "action": 2, "data": "/sign", "direct_send": true }
        ]
      ]
    }
  }
  ```
- **主动发消息**：`gorobot_send_message(req_json)`
  - 主动向指定用户或群聊发送消息，同样支持可选的 `keyboard` 结构。
  ```json
  {
    "context_id": "telegram",  // 机器人上下文 ID，留空默认首个可用适配器
    "target_id": "12345678",   // 接收者 User ID 或 Group ID
    "text": "你好，这是来自 Wasm 插件的主动推送",
    "keyboard": {
      "rows": [
        [
          { "text": "赞", "action": 1, "data": "like_event" }
        ]
      ]
    }
  }
  ```

> **内嵌按钮 (Inline Keyboard) 字段说明**：
> - `action`: 按钮动作类型。`0`=打开 URL（`ActionURL`），`1`=回调事件（`ActionCallback`，如 Telegram callback_data / QQ INTERACTION_CREATE），`2`=指令型（`ActionCommand`）。
> - `data`: 动作对应的数据（URL 地址、回调 payload 或待执行指令）。
> - `direct_send`: 布尔值，仅指令型按钮支持（为 `true` 时在支持平台直接自动发送该指令）。

### 4. 查询适配器支持特性 (Feature Discovery)
- **事件上下文中内置**：传给插件的 `CommandEvent` 与 `MessageEventPayload` 已内置 `features` 字段数组（例如 `["text", "image", "markdown", "voice", "file", "inline_keyboard"]`），插件收到事件即可直接判断是否能发送内嵌按钮或 Markdown。
- **主动查询函数**：`gorobot_get_features(req_json)`
  - 支持通过当前会话 `context_token` 或机器人 `context_id` 主动探测平台支持特性。
  ```json
  // 请求示例
  { "context_token": "..." }

  // 响应示例
  {
    "success": true,
    "features": ["text", "image", "markdown", "voice", "file", "inline_keyboard"]
  }
  ```

### 5. 监听聊天事件
- **函数名**：`gorobot_subscribe_event(req_json)`
- **说明**：订阅通用事件（如 `message`）。每当收到聊天消息时回调插件指定的导出函数。
  ```json
  {
    "event": "message",
    "handler": "on_message"
  }
  ```
- **回调负载 (`MessageEventPayload`)**：
  ```json
  {
    "message_type": "group",       // "group" 或 "direct"
    "group_id": "group_12345",     // 群聊 ID（私聊为空）
    "group": { "id": "group_12345", "name": "交流群" },
    "sender_id": "user_67890",
    "sender": { "id": "user_67890", "nickname": "三哥", "authority": 1 },
    "protocol": "telegram",
    "features": ["text", "image", "inline_keyboard"],
    "text": "测试消息内容",
    "timestamp": 1726272000,
    "context_token": "..."
  }
  ```

### 6. 网络访问（Outgoing HTTP）
- **函数名**：`gorobot_http_request(req_json)`
- **说明**：发起对外网络请求（支持 GET、POST、PUT、DELETE 等）。由于默认开放了 `AllowedHosts: ["*"]`，插件也可以直接使用 Extism PDK 内置的 HTTP 客户端。
- **入参示例**：
  ```json
  {
    "url": "https://api.weather.com/v1/today",
    "method": "GET",
    "headers": {"User-Agent": "GoroBot-Wasm"},
    "timeout_ms": 5000
  }
  ```
- **返回响应**：包含 `status_code`、`headers` 与 `body` 的 JSON 字符串。

### 7. 网络监听（Incoming HTTP / Webhook）
- **函数名**：`gorobot_listen_http(req_json)`
- **说明**：由宿主代理监听指定的 HTTP 端口与路由。当外部 Webhook 或客户端请求到达时，由宿主转发给 Wasm 插件处理，插件返回的结果自动写回 HTTP 客户端。
- **入参示例**：
  ```json
  {
    "addr": ":8080",
    "path": "/webhook",
    "handler": "on_http_request"
  }
  ```
- **请求/响应协议**：
  - 外部请求送入插件导出函数格式：
    ```json
    {
      "method": "POST",
      "path": "/webhook",
      "query": "sig=123",
      "headers": {"Content-Type": "application/json"},
      "body": "..."
    }
    ```
  - 插件返回响应格式：
    ```json
    {
      "status_code": 200,
      "headers": {"Content-Type": "application/json"},
      "body": "{\"success\": true}"
    }
    ```
- **生命周期绑定**：当插件被重载或禁用时，宿主自动清理路由；若该端口上无其他活跃路由，对应的 HTTP 服务将自动优雅停止。

### 8. 辅助文件系统操作
- `gorobot_fs_read(path)`：读取 `data/<plugin_id>/` 内的文件内容。
- `gorobot_fs_write(path, data)`：写入文件到 `data/<plugin_id>/`。
- 内置防目录穿越逻辑，拦截所有试图逃逸数据目录的相对或绝对路径。

---

## 插件编写示例（Rust 版）

使用 Rust 配合 `extism-pdk` 编写一个完整的 Wasm 插件：

```rust
use extism_pdk::*;
use serde::{Deserialize, Serialize};

#[host_fn]
extern "ExtismHost" {
    fn gorobot_log(level: &str, msg: &str);
    fn gorobot_register_command(def_json: &str) -> String;
}

#[derive(Serialize)]
struct CommandDef {
    name: String,
    description: String,
    handler: String,
}

#[derive(Deserialize)]
struct CommandContext {
    sender_id: String,
    raw: String,
}

// 插件加载时自动执行的初始化钩子
#[plugin_fn]
pub fn on_init() -> FnResult<()> {
    unsafe {
        gorobot_log("info", "Dice 插件正在加载...")?;

        let def = CommandDef {
            name: "dice".to_string(),
            description: "掷骰子".to_string(),
            handler: "on_dice".to_string(),
        };
        let _ = gorobot_register_command(&serde_json::to_string(&def)?)?;
    }
    Ok(())
}

// 指令触发时的回调函数
#[plugin_fn]
pub fn on_dice(Json(ctx): Json<CommandContext>) -> FnResult<String> {
    // 读取与写入插件独占的数据目录文件
    let counter_file = "/counter.txt";
    let count: u32 = std::fs::read_to_string(counter_file)
        .unwrap_or_else(|_| "0".to_string())
        .trim()
        .parse()
        .unwrap_or(0) + 1;
    let _ = std::fs::write(counter_file, count.to_string());

    // 返回的文本会自动回复给发送者
    Ok(format!("用户 {} 掷出了 6 点！本插件已累计被调用 {} 次。", ctx.sender_id, count))
}
```

编译生成 `.wasm` 文件并放置到 GoroBot 的 `plugin/` 目录：
```bash
cargo build --target wasm32-wasip1 --release
cp target/wasm32-wasip1/release/dice.wasm path/to/GoroBot/plugin/dice.wasm
```

在聊天窗口发送 `/wasm lookup` 与 `/wasm enable dice` 即可立即启用！
