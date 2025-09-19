# 通信

目前包括四种通信方式：

- **HTTP**：OneBot 作为 HTTP 服务端，提供 API 调用服务
- **HTTP POST**：OneBot 作为 HTTP 客户端，向用户配置的 URL 推送事件，并处理用户返回的响应
- **正向 WebSocket**：OneBot 作为 WebSocket 服务端，接受用户连接，提供 API 调用和事件推送服务
- **反向 WebSocket**：OneBot 作为 WebSocket 客户端，主动连接用户配置的 URL，提供 API 调用和事件推送服务

所有通信方式传输的数据都使用 UTF-8 编码。

> **注意**
>
> 在原 CQHTTP 插件中，HTTP POST 没有被列为一种独立的通信方式，而是单纯由 `post_url` 指定上报地址然后上报，但严格来说它也是一种通信方式，只用于推送事件，并且用法不同于 HTTP。这里为了清晰起见，把它增列为一种通信方式。

# HTTP

- [请求](#请求)
- [响应](#响应)
- [相关配置](#相关配置)

OneBot 在启动时开启一个 HTTP 服务器，监听配置文件指定的 IP 和端口，接受路径为 `/:action` 的 API 请求（或 `/:action/`），如 `/send_private_msg`，请求可以使用 GET 或 POST 方法，可以通过 query 参数（`?arg1=111&arg2=222`）、urlencoded 表单（`arg1=111&arg2=222`）或 JSON（`{"arg1": "111", "arg2": "222"}`）传递参数。

参数可能有不同的类型，当用户通过 query 参数或 urlencoded 表单传参，或在 JSON 中使用字符串作为参数值时，OneBot 实现需要从字符串解析出对应类型的数据。

## 请求

假设配置中指定了 IP 和端口分别为 `127.0.0.1` 和 `5700`，则在浏览器中访问 `http://127.0.0.1:5700/send_private_msg?user_id=1000010000&message=hello` 即可给 QQ 号为 `1000010000` 的好友发送私聊消息 `hello`。

如需使用 JSON 传递参数，则请求如下：

```http
POST /send_private_msg HTTP/1.1
Host: 127.0.0.1:5700
Content-Type: application/json

{
    "user_id": 1000010000,
    "message": "hello"
}
```

> **注意**
>
> - 当使用 query 参数或 urlencoded 表单传递参数时，参数值必须进行 urlencode。
> - 当使用 urlencoded 表单或 JSON 传递参数时，请求头中的 `Content-Type` 必须对应的为 `application/x-www-form-urlencoded` 或 `application/json`。

上例中调用的 API（即 action）为 `send_private_msg`，其它 API 及它们的参数和响应内容，见 [API](../api/)。

## 响应

收到 API 请求并处理后，OneBot 会返回一个 HTTP 响应，根据具体情况不同，HTTP 状态码不同：

- 如果 access token 未提供，状态码为 401（关于 access token，见 [鉴权](authorization.md)）
- 如果 access token 不符合，状态码为 403
- 如果 POST 请求的 Content-Type 不支持，状态码为 406
- 如果 POST 请求的正文格式不正确，状态码为 400
- 如果 API 不存在，状态码为 404
- 剩下的所有情况，无论操作实际成功与否，状态码都是 200

状态码为 200 时，响应内容为 JSON 格式，基本结构如下：

```json
{
    "status": "ok",
    "retcode": 0,
    "data": {
        "id": 123456,
        "nickname": "滑稽"
    }
}
```

`status` 字段表示请求的状态：

- `ok` 表示操作成功，同时 `retcode` （返回码）会等于 0
- `async` 表示请求已提交异步处理，此时 `retcode` 为 1，具体成功或失败将无法获知
- `failed` 表示操作失败，此时 `retcode` 既不是 0 也不是 1，具体错误信息应参考 OneBot 实现的日志

`data` 字段为 API 返回数据的内容，对于踢人、禁言等不需要返回数据的操作，这里为 null，对于获取群成员信息这类操作，这里为所获取的数据的对象，具体的数据内容将会在相应的 API 描述中给出。注意，异步版本的 API，`data` 永远是 null，即使其相应的同步接口本身是有数据。

## 相关配置

> **提示**
>
> 配置项名称仅供参考，不同 OneBot 实现可以自行定义名称，只需在样例配置中给出注释即可。

| 配置项 | 默认值 | 说明 |
| -------- | ------ | --- |
| `http.enable` | `true` | 是否启用 HTTP |
| `http.host` | `0.0.0.0` | HTTP 服务器监听的 IP |
| `http.port` | `5700` | HTTP 服务器监听的端口 |

# HTTP POST

- [上报](#上报)
- [签名](#签名)
- [快速操作](#快速操作)
- [相关配置](#相关配置)

OneBot 在收到事件后，向配置指定的事件上报 URL 通过 POST 请求发送事件数据，事件数据以 JSON 格式表示。请求结束后，OneBot 处理用户返回的响应中的「快速操作」，如快速回复、快速禁言等。

## 上报

假设配置指定的上报 URL 为 `http://127.0.0.1:8080/`，以私聊消息为例，事件上报的 POST 请求如下：

```http
POST / HTTP/1.1
Host: 127.0.0.1:8080
Content-Type: application/json
X-Self-ID: 10001000

{
    "time": 1515204254,
    "self_id": 10001000,
    "post_type": "message",
    "message_type": "private",
    "sub_type": "friend",
    "message_id": 12,
    "user_id": 12345678,
    "message": "你好～",
    "raw_message": "你好～",
    "font": 456,
    "sender": {
        "nickname": "小不点",
        "sex": "male",
        "age": 18
    }
}
```

请求头中的 `X-Self-ID` 表示当前正在上报的机器人 QQ 号，和请求正文 JSON 中的 `self_id` 一致。

上例中的事件为私聊消息事件，其它事件及它们的上报内容和支持的响应数据，见 [事件](../event/)。

## 签名

如果配置中给出了 `secret`，即签名密钥，则会在每次上报的请求头中加入 HMAC 签名，即 `X-Signature` 头，如：

```http
POST / HTTP/1.1
Host: 127.0.0.1:8080
Content-Type: application/json
X-Signature: sha1=f9ddd4863ace61e64f462d41ca311e3d2c1176e2
X-Self-ID: 10001000

...
```

签名以 `secret` 作为密钥，HTTP 正文作为消息，进行 HMAC SHA1 哈希，用户后端可以通过该哈希值来验证上报的数据确实来自 OneBot，而不是第三方伪造的。HMAC 介绍见 [密钥散列消息认证码](https://zh.wikipedia.org/zh-cn/%E9%87%91%E9%91%B0%E9%9B%9C%E6%B9%8A%E8%A8%8A%E6%81%AF%E9%91%91%E5%88%A5%E7%A2%BC)。

### HMAC SHA1 校验的示例

#### Python + Flask

```python
import hmac
from flask import Flask, request

app = Flask(__name__)

@app.route('/', methods=['POST'])
def receive():
    sig = hmac.new(b'<your-key>', request.get_data(), 'sha1').hexdigest()
    received_sig = request.headers['X-Signature'][len('sha1='):]
    if sig == received_sig:
        # 请求确实来自于 OneBot
        pass
    else:
        # 假的上报
        pass
```

#### Node.js + Koa

```js
const crypto = require('crypto');
const secret = 'some-secret';

// 在 Koa 的请求 context 中
ctx.assert(ctx.request.headers['x-signature'] !== undefined, 401);
const hmac = crypto.createHmac('sha1', secret);
hmac.update(ctx.request.rawBody);
const sig = hmac.digest('hex');
ctx.assert(ctx.request.headers['x-signature'] === `sha1=${sig}`, 403);
```

## 快速操作

事件上报的后端可以在上报请求的响应中直接指定一些简单的操作，称为「快速操作」，如快速回复、快速禁言等。如果不需要使用这个特性，返回 HTTP 响应状态码 204，或保持响应正文内容为空；如果需要，则使用 JSON 作为响应正文，`Content-Type` 响应头任意（目前不会进行判断），但设置为 `application/json` 最好，以便减少不必要的升级成本，因为如果以后有需求，可能会加入判断。

> **注意**：无论是否需要使用快速操作，事件上报后端都应该在处理完毕后返回 HTTP 响应，否则 OneBot 将一直等待直到超时。

响应的 JSON 数据中，支持的操作随事件的不同而不同，会在事件列表中的「快速操作」标题下给出。需要指出的是，**响应数据中的每个字段都是可选的**，只有在字段存在（明确要求进行操作）时，才会触发相应的操作，否则将保持对机器人整体运行状态影响最小的行为（比如默认不回复消息、不处理请求）。

以私聊消息为例，事件上报后端若返回如下 JSON 作为响应正文：

```json
{
    "reply": "嗨～"
}
```

则会回复 `嗨～`。

## 相关配置

| 配置项名称 | 默认值 | 说明 |
| -------- | ------ | --- |
| `http_post.enable` | `true` | 是否启用 HTTP POST |
| `http_post.url` | 空 | 事件上报 URL |
| `http_post.timeout` | `0` | HTTP 上报超时时间，单位秒，0 表示不设置超时 |
| `http_post.secret` | 空 | 上报数据签名密钥 |

# 正向 WebSocket

- [`/api` 接口](#api-接口)
- [`/event` 接口](#event-接口)
- [`/` 接口](#-接口)
- [相关配置](#相关配置)

OneBot 在启动时开启一个 WebSocket 服务器，监听配置文件指定的 IP 和端口，接受路径为 `/api`（或 `/api/`）、`/event`（或 `/event/`）、`/` 的连接请求。连接建立后，将一直保持连接（用户可主动断开连接），并根据路径的不同，提供 API 调用或事件推送服务。通过 WebSocket 消息发送的数据全部使用 JSON 格式。

## `/api` 接口

连接此接口后，向 OneBot 发送如下结构的 JSON 对象，即可调用相应的 API：

```json
{
    "action": "send_private_msg",
    "params": {
        "user_id": 10001000,
        "message": "你好"
    },
    "echo": "123"
}
```

这里的 `action` 参数用于指定要调用的 API，具体参考 [API](../api/)；`params` 用于传入参数，如果要调用的 API 不需要参数，则可以不加；`echo` 字段是可选的，类似于 [JSON RPC](https://www.jsonrpc.org/specification) 的 `id` 字段，用于唯一标识一次请求，可以是任何类型的数据，OneBot 将会在调用结果中原样返回。

客户端向 OneBot 发送 JSON 之后，OneBot 会往回发送一个调用结果，结构和 [HTTP 的响应](http.md#响应) 相似，（除了包含请求中传入的 `echo` 字段外）唯一的区别在于，通过 HTTP 调用 API 时，HTTP 状态码反应的错误情况被移动到响应 JSON 的 `retcode` 字段，例如，HTTP 返回 404 的情况，对应到 WebSocket 的回复，是：

```json
{
    "status": "failed",
    "retcode": 1404,
    "data": null,
    "echo": "123"
}
```

下面是 `retcode` 和 HTTP 状态码的对照：

| `retcode` | HTTP 接口中的状态码 |
| --------- | ----------------- |
| 1400 | 400 |
| 1401 | 401 |
| 1403 | 403 |
| 1404 | 404 |

实际上 `1401` 和 `1403` 并不会真的返回，因为如果建立连接时鉴权失败，连接会直接断开，根本不可能进行到后面的 API 调用阶段。

## `/event` 接口

连接此接口后，OneBot 会在收到事件后推送至客户端，推送的格式和 [HTTP POST 的上报](http-post.md#上报) 完全一致，事件的具体内容见 [事件](../event/)。

与 HTTP POST 不同的是，WebSocket 推送不会对数据进行签名（即 HTTP POST 中的 `X-Signature` 请求头在这里没有等价的东西），并且也不会处理响应数据。如果对事件进行处理的时候需要调用接口，请使用 [`/api` 接口](#api-接口) 或使用 HTTP 调用 API。

此外，这个接口和 HTTP POST 不冲突，如果启用了正向 WebSocket，同时也配置了 HTTP POST 的上报地址，OneBot 会先通过 HTTP POST 上报，在处理完它的响应后，向所有已连接了 `/event` 的 WebSocket 客户端推送事件。

## `/` 接口

在一条连接上同时提供 `/api` 和 `/event` 的服务，使用方式参考上面。

## 相关配置

| 配置项名称 | 默认值 | 说明 |
| -------- | ------ | --- |
| `ws.enable` | `false` | 是否启用正向 WebSocket |
| `ws.host` | `0.0.0.0` | WebSocket 服务器监听的 IP |
| `ws.port` | `6700` | WebSocket 服务器监听的端口 |

# 反向 WebSocket

- [连接请求](#连接请求)
- [断线重连](#断线重连)
- [相关配置](#相关配置)

OneBot 启动后，作为客户端向用户配置的反向 WebSocket URL 建立连接。连接建立后，将一直保持连接，并根据连接的 URL 不同，提供 API 调用或事件推送服务。通过 WebSocket 消息发送的数据全部使用 JSON 格式。

## 连接请求

根据配置的不同，连接用户提供的 URL 的客户端有三种：API 客户端、Event 客户端和 Universal 客户端。API 客户端提供 API 调用服务；Event 客户端提供事件推送服务；Universal 客户端**在一条连接上**同时提供两种服务。

> **注意**
>
> 只要服务器能够正确区分，API 客户端和 Event 客户端可以向同一个 URL 建立连接，但这是两条连接，和 Universal 客户端不同。

各客户端建立连接的方式相同，以 API 客户端为例，假设设置了 API URL 为 `ws://127.0.0.1:8080/ws/api`，则连接请求如下：

```http
GET /ws/api HTTP/1.1
Host: 127.0.0.1:8080
Connection: Upgrade
Upgrade: websocket
X-Self-ID: 10001000
X-Client-Role: API
...
```

请求头中的 `X-Self-ID` 表示当前正在建立连接的机器人 QQ 号；`X-Client-Role` 表示当前正在建立连接的客户端类型，对于 Event 客户端和 Universal 客户端，这里分别是 `Event` 和 `Universal`。

连接建立后，使用方式同 [正向 WebSocket](ws.md)。

## 断线重连

当由于各种意外情况，连接断开时，OneBot 将以配置中指定的时间间隔不断尝试重连，直到再次连接成功。

## 相关配置

| 配置项 | 默认值 | 说明 |
| -------- | ------ | --- |
| `ws_reverse.enable` | `true` | 是否启用反向 WebSocket |
| `ws_reverse.url` | 空 | 反向 WebSocket API、Event、Universal 共用 URL |
| `ws_reverse.api_url` | 空 | 反向 WebSocket API URL，如果为空，则使用 `ws_reverse.url` 指定的值 |
| `ws_reverse.event_url` | 空 | 反向 WebSocket Event URL，如果为空，则使用 `ws_reverse.url` 指定的值 |
| `ws_reverse.use_universal_client` | `false` | 是否使用 Universal 客户端 |
| `ws_reverse.reconnect_interval` | `3000` | 反向 WebSocket 客户端断线重连间隔，单位毫秒 |

# 鉴权

- [HTTP 和正向 WebSocket](#http-和正向-websocket)
- [反向 WebSocket](#反向-websocket)
- [相关配置](#相关配置)

在 HTTP POST 通信方式中，OneBot 提供了 [签名](http-post.md#签名) 来保证安全性，而在 HTTP、正向 WebSocket、反向 WebSocket 通信方式中，通过对 access token 进行验证来保证安全性。

## HTTP 和正向 WebSocket

如果配置文件中填写了 access token，则每次客户端向 OneBot 发送请求时需要在请求头中加入 `Authorization` 头，如：

```http
GET /get_friend_list HTTP/1.1
...
Authorization: Bearer kSLuTF2GC2Q4q4ugm3
```

`Bearer` 后面需给出和 OneBot 配置中相同的 access token。

在某些特殊情况下，可能无法修改请求头，则可以通过 query 参数传入 access token，例如：

```http
GET /get_friend_list?access_token=kSLuTF2GC2Q4q4ugm3 HTTP/1.1
```

## 反向 WebSocket

如果配置文件中填写了 access token，则每次 OneBot 的反向 WebSocket 客户端在向用户配置的 URL 建立连接的时候，会在请求头中加入 `Authorization` 头，如：

```http
GET /ws/api HTTP/1.1
...
Authorization: Bearer kSLuTF2GC2Q4q4ugm3
```

## 相关配置

| 配置项名称 | 默认值 | 说明 |
| -------- | ------ | --- |
| `auth.access_token` | 空 | access token |

# 消息

消息是 OneBot 标准中一个重要的数据类型，在发送消息的 API 和接收消息的事件中都有相关字段。目前消息的格式分为两种：字符串（string）和数组（array）。

# 字符串格式

- [CQ 码格式](#cq-码格式)
- [转义](#转义)

字符串格式是 CKYU 原生所使用的消息格式，在本标准中将继续使用以保持兼容。在字符串格式中，无论纯文本还是图片、表情、链接分享等多媒体内容都放在同一个字符串里，即，一条消息对应一个字符串。以下是一个字符串格式消息的例子：

```
[CQ:face,id=178]看看我刚拍的照片[CQ:image,file=123.jpg]
```

在调用 API 发送这段消息时，JSON 如下：

```json
{
    "user_id": 10001000,
    "message": "[CQ:face,id=178]看看我刚拍的照片[CQ:image,file=123.jpg]"
}
```

## CQ 码格式

消息中的多媒体内容使用 CQ 码来表示，形如 `[CQ:face,id=178]`。其中，`[CQ:]` 是固定格式；`face` 是「功能名」，除了 `face` 还有许多不同的功能名；`id=178` 是「参数」，某些功能不需要参数，而另一些需要多个参数，当有多个参数时，参数间使用逗号分隔。

一些 CQ 码的例子如下：

```
[CQ:shake]
[CQ:face,id=178]
[CQ:share,title=标题,url=http://baidu.com]
```

更多 CQ 码功能请参考 [消息段类型](segment.md)。

> **注意**
>
> CQ 码中不应有多余的空格，例如不应该使用 `[CQ:face, id=178]`。
>
> CQ 码的参数值可以包含空格、换行、除 `[],&` 之外的特殊符号等。在解析时，应直接取 `[CQ:` 后、第一个 `,` 或 `]` 前的部分为功能名，第一个 `,` 之后到 `]` 之间的部分为参数，按 `,` 分割后，每个部分第一个 `=` 前的内容为参数名，之后的部分为参数值。例如 `[CQ:share,title=标题中有=等号,url=http://baidu.com]` 中，功能名为 `share`，`title` 参数值为 `标题中有=等号`，`url` 参数值为 `http://baidu.com`。

## 转义

CQ 码中包含一些特殊字符：`[`、`]`、`,` 等，而 CQ 码又是可能混杂在纯文本内容之中的，因此消息中的纯文本内容需要对特殊字符进行转义，以避免歧义。具体的转义规则如下：

| 转义前 | 转义后 |
| --- | --- |
| `&` | `&amp;` |
| `[` | `&#91;` |
| `]` | `&#93;` |

例如，一个纯文本消息转义前内容如下：

```
- [x] 使用 `&data` 获取地址
```

转义后如下：

```
- &#91;x&#93; 使用 `&amp;data` 获取地址
```

另一方面，CQ 码内部的参数值也可能出现特殊字符，也是需要转义的。由于 `,`（半角逗号）在 CQ 码中用于分隔参数，因此除了上面的转义规则，还需要对 `,` 进行转义，如下：

| 转义前 | 转义后 |
| --- | --- |
| `&` | `&amp;` |
| `[` | `&#91;` |
| `]` | `&#93;` |
| `,` | `&#44;` |

例如，一个链接分享消息的 CQ 码可能如下：

```
[CQ:share,title=震惊&#44;小伙睡觉前居然...,url=http://baidu.com/?a=1&amp;b=2]
```

# 数组格式

- [消息段](#消息段)
- [消息段数组](#消息段数组)

数组格式将消息表示为一系列消息段对象的数组，在基本语义上与字符串格式等价，可以相互转换，但数组格式的表达能力更强，例如可以嵌套、规定参数数据类型等。

## 消息段

在字符串格式中，使用 CQ 码表示多媒体内容，例如 `[CQ:image,file=123.jpg]`，这里 CQ 码功能名为 `image`，参数为 `file=123.jpg`，也即一个键值对。

消息段是 CQ 码在数组格式中的表示形式，基本格式如下：

```json
{
    "type": "image",
    "data": {
        "file": "123.jpg"
    }
}
```

其中 `type` 字段的类型为字符串，对应 CQ 码中的「功能名」；`data` 字段的类型为对象，对应 CQ 码的「参数」，此字段可为 null。**目前，除了合并转发相关的特殊消息段外，几乎所有消息段参数值类型均为字符串，以支持与 CQ 码的相互转换**。

**由于消息段不会将纯文本和多媒体内容放在一起，也就意味着任意一个字段的值都是真实值，而不再需要转义。**

为了使用消息段表示纯文本，引入一个特殊消息段类型 `text`，并在 `data` 中使用 `text` 字段来指示纯文本内容，例如：

```json
{
    "type": "text",
    "data": {
        "text": "这是一段纯文本"
    }
}
```

在将上面的消息段转成 CQ 码时，将会直接变成纯文本字符串，而不是真的转成 CQ 码。

更多消息段类型请参考 [消息段类型](segment.md)。

## 消息段数组

了解了消息段概念之后，就不难理解消息的数组格式了，即消息段组成的数组。

例如，字符串格式下的这样一条消息：

```
&#91;第一部分&#93;[CQ:image,file=123.jpg]图片之后的部分，表情：[CQ:face,id=123]
```

表示成数组格式即为：

```json
[
    {
        "type": "text",
        "data": {
            "text": "[第一部分]"
        }
    },
    {
        "type": "image",
        "data": {
            "file": "123.jpg"
        }
    },
    {
        "type": "text",
        "data": {
            "text": "图片之后的部分，表情："
        }
    },
    {
        "type": "face",
        "data": {
            "id": "123"
        }
    }
]
```

# 消息段类型

- [纯文本](#纯文本)
- [QQ 表情](#qq-表情)
- [图片](#图片)
- [语音](#语音)
- [短视频](#短视频)
- [@某人](#某人)
- [猜拳魔法表情](#猜拳魔法表情)
- [掷骰子魔法表情](#掷骰子魔法表情)
- [窗口抖动（戳一戳） <Badge text="发"/>](#窗口抖动戳一戳-badge-text发)
- [戳一戳](#戳一戳)
- [匿名发消息 <Badge text="发"/>](#匿名发消息-badge-text发)
- [链接分享](#链接分享)
- [推荐好友](#推荐好友)
- [推荐群](#推荐群)
- [位置](#位置)
- [音乐分享 <Badge text="发"/>](#音乐分享-badge-text发)
- [音乐自定义分享 <Badge text="发"/>](#音乐自定义分享-badge-text发)
- [回复](#回复)
- [合并转发 <Badge text="收"/>](#合并转发-badge-text收)
- [合并转发节点 <Badge text="发"/>](#合并转发节点-badge-text发)
- [合并转发自定义节点](#合并转发自定义节点)
- [XML 消息](#xml-消息)
- [JSON 消息](#json-消息)

对于每一种消息段类型，将分别给出消息段格式和 CQ 码格式的例子，然后解释各参数的含义。

下面所有可能的值为 `0` 和 `1` 的参数，也可以使用 `no` 和 `yes`、`false` 和 `true`。

## 纯文本

```json
{
    "type": "text",
    "data": {
        "text": "纯文本内容"
    }
}
```

```
纯文本内容
```

| 参数名| 收 | 发 | 可能的值 | 说明 |
| --- | --- | --- | --- | --- |
| `text` | ✓ | ✓ | - | 纯文本内容 |

## QQ 表情

```json
{
    "type": "face",
    "data": {
        "id": "123"
    }
}
```

```
[CQ:face,id=123]
```

| 参数名 | 收 | 发 | 可能的值 | 说明 |
| --- | --- | --- | --- | --- |
| `id` | ✓ | ✓ | 见 [QQ 表情 ID 表](https://github.com/richardchien/coolq-http-api/wiki/%E8%A1%A8%E6%83%85-CQ-%E7%A0%81-ID-%E8%A1%A8) | QQ 表情 ID |

## 图片

```json
{
    "type": "image",
    "data": {
        "file": "http://baidu.com/1.jpg"
    }
}
```

```
[CQ:image,file=http://baidu.com/1.jpg]
```

| 参数名 | 收 | 发 | 可能的值 | 说明 |
| --- | --- | --- | --- | --- |
| `file` | ✓ | ✓<sup>[1]</sup> | - | 图片文件名 |
| `type` | ✓ | ✓ | `flash` | 图片类型，`flash` 表示闪照，无此参数表示普通图片 |
| `url` | ✓ |  | - | 图片 URL |
| `cache` |  | ✓ | `0` `1` | 只在通过网络 URL 发送时有效，表示是否使用已缓存的文件，默认 `1` |
| `proxy` |  | ✓ | `0` `1` | 只在通过网络 URL 发送时有效，表示是否通过代理下载文件（需通过环境变量或配置文件配置代理），默认 `1` |
| `timeout` |  | ✓ | - | 只在通过网络 URL 发送时有效，单位秒，表示下载网络文件的超时时间，默认不超时 |

[1] 发送时，`file` 参数除了支持使用收到的图片文件名直接发送外，还支持：

- 绝对路径，例如 `file:///C:\\Users\Richard\Pictures\1.png`，格式使用 [`file` URI](https://tools.ietf.org/html/rfc8089)
- 网络 URL，例如 `http://i1.piimg.com/567571/fdd6e7b6d93f1ef0.jpg`
- Base64 编码，例如 `base64://iVBORw0KGgoAAAANSUhEUgAAABQAAAAVCAIAAADJt1n/AAAAKElEQVQ4EWPk5+RmIBcwkasRpG9UM4mhNxpgowFGMARGEwnBIEJVAAAdBgBNAZf+QAAAAABJRU5ErkJggg==`

## 语音

```json
{
    "type": "record",
    "data": {
        "file": "http://baidu.com/1.mp3"
    }
}
```

```
[CQ:record,file=http://baidu.com/1.mp3]
```

| 参数名 | 收 | 发 | 可能的值 | 说明 |
| --- | --- | --- | --- | --- |
| `file` | ✓ | ✓<sup>[1]</sup> | - | 语音文件名 |
| `magic` | ✓ | ✓ | `0` `1` | 发送时可选，默认 `0`，设置为 `1` 表示变声 |
| `url` | ✓ |  | - | 语音 URL |
| `cache` |  | ✓ | `0` `1` | 只在通过网络 URL 发送时有效，表示是否使用已缓存的文件，默认 `1` |
| `proxy` |  | ✓ | `0` `1` | 只在通过网络 URL 发送时有效，表示是否通过代理下载文件（需通过环境变量或配置文件配置代理），默认 `1` |
| `timeout` |  | ✓ | - | 只在通过网络 URL 发送时有效，单位秒，表示下载网络文件的超时时间 ，默认不超时|

[1] 发送时，`file` 参数除了支持使用收到的语音文件名直接发送外，还支持其它形式，参考 [图片](#图片)。

## 短视频

```json
{
    "type": "video",
    "data": {
        "file": "http://baidu.com/1.mp4"
    }
}
```

```
[CQ:video,file=http://baidu.com/1.mp4]
```

| 参数名 | 收 | 发 | 可能的值 | 说明 |
| --- | --- | --- | --- | --- |
| `file` | ✓ | ✓<sup>[1]</sup> | - | 视频文件名 |
| `url` | ✓ |  | - | 视频 URL |
| `cache` |  | ✓ | `0` `1` | 只在通过网络 URL 发送时有效，表示是否使用已缓存的文件，默认 `1` |
| `proxy` |  | ✓ | `0` `1` | 只在通过网络 URL 发送时有效，表示是否通过代理下载文件（需通过环境变量或配置文件配置代理），默认 `1` |
| `timeout` |  | ✓ | - | 只在通过网络 URL 发送时有效，单位秒，表示下载网络文件的超时时间 ，默认不超时|

[1] 发送时，`file` 参数除了支持使用收到的视频文件名直接发送外，还支持其它形式，参考 [图片](#图片)。

## @某人

```json
{
    "type": "at",
    "data": {
        "qq": "10001000"
    }
}
```

```
[CQ:at,qq=10001000]
```

| 参数名 | 收 | 发 | 可能的值 | 说明 |
| --- | --- | --- | --- | --- |
| `qq` | ✓ | ✓ | QQ 号、`all` | @的 QQ 号，`all` 表示全体成员 |

## 猜拳魔法表情

```json
{
    "type": "rps",
    "data": {}
}
```

```
[CQ:rps]
```

## 掷骰子魔法表情

```json
{
    "type": "dice",
    "data": {}
}
```

```
[CQ:dice]
```

## 窗口抖动（戳一戳） <Badge text="发"/>

> **提示**
>
> 相当于戳一戳最基本类型的快捷方式。

```json
{
    "type": "shake",
    "data": {}
}
```

```
[CQ:shake]
```

## 戳一戳

```json
{
    "type": "poke",
    "data": {
        "type": "126",
        "id": "2003"
    }
}
```

```
[CQ:poke,type=126,id=2003]
```

| 参数名 | 收 | 发 | 可能的值 | 说明 |
| --- | --- | --- | --- | --- |
| `type` | ✓ | ✓ | 见 [Mirai 的 PokeMessage 类](https://github.com/mamoe/mirai/blob/f5eefae7ecee84d18a66afce3f89b89fe1584b78/mirai-core/src/commonMain/kotlin/net.mamoe.mirai/message/data/HummerMessage.kt#L49) | 类型 |
| `id` | ✓ | ✓ | 同上 | ID |
| `name` | ✓ |  | 同上 | 表情名 |

## 匿名发消息 <Badge text="发"/>

> **提示**
>
> 当收到匿名消息时，需要通过 [消息事件的群消息](../event/message.md#群消息) 的 `anonymous` 字段判断。

```json
{
    "type": "anonymous",
    "data": {}
}
```

```
[CQ:anonymous]
```

| 参数名 | 收 | 发 | 可能的值 | 说明 |
| --- | --- | --- | --- | --- |
| `ignore` |  | ✓ | `0` `1` | 可选，表示无法匿名时是否继续发送 |

## 链接分享

```json
{
    "type": "share",
    "data": {
        "url": "http://baidu.com",
        "title": "百度"
    }
}
```

```
[CQ:share,url=http://baidu.com,title=百度]
```

| 参数名 | 收 | 发 | 可能的值 | 说明 |
| --- | --- | --- | --- | --- |
| `url` | ✓ | ✓ | - | URL |
| `title` | ✓ | ✓ | - | 标题 |
| `content` | ✓ | ✓ | - | 发送时可选，内容描述 |
| `image` | ✓ | ✓ | - | 发送时可选，图片 URL |

## 推荐好友

```json
{
    "type": "contact",
    "data": {
        "type": "qq",
        "id": "10001000"
    }
}
```

```
[CQ:contact,type=qq,id=10001000]
```

| 参数名 | 收 | 发 | 可能的值 | 说明 |
| --- | --- | --- | --- | --- |
| `type` | ✓ | ✓ | `qq` | 推荐好友 |
| `id` | ✓ | ✓ | - | 被推荐人的 QQ 号 |

## 推荐群

```json
{
    "type": "contact",
    "data": {
        "type": "group",
        "id": "100100"
    }
}
```

```
[CQ:contact,type=group,id=100100]
```

| 参数名 | 收 | 发 | 可能的值 | 说明 |
| --- | --- | --- | --- | --- |
| `type` | ✓ | ✓ | `group` | 推荐群 |
| `id` | ✓ | ✓ | - | 被推荐群的群号 |

## 位置

```json
{
    "type": "location",
    "data": {
        "lat": "39.8969426",
        "lon": "116.3109099"
    }
}
```

```
[CQ:location,lat=39.8969426,lon=116.3109099]
```

| 参数名 | 收 | 发 | 可能的值 | 说明 |
| --- | --- | --- | --- | --- |
| `lat` | ✓ | ✓ | - | 纬度 |
| `lon` | ✓ | ✓ | - | 经度 |
| `title` | ✓ | ✓ | - | 发送时可选，标题 |
| `content` | ✓ | ✓ | - | 发送时可选，内容描述 |

## 音乐分享 <Badge text="发"/>

```json
{
    "type": "music",
    "data": {
        "type": "163",
        "id": "28949129"
    }
}
```

```
[CQ:music,type=163,id=28949129]
```

| 参数名 | 收 | 发 | 可能的值 | 说明 |
| --- | --- | --- | --- | --- |
| `type` |  | ✓ | `qq` `163` `xm` | 分别表示使用 QQ 音乐、网易云音乐、虾米音乐 |
| `id` |  | ✓ | - | 歌曲 ID |

## 音乐自定义分享 <Badge text="发"/>

```json
{
    "type": "music",
    "data": {
        "type": "custom",
        "url": "http://baidu.com",
        "audio": "http://baidu.com/1.mp3",
        "title": "音乐标题"
    }
}
```

```
[CQ:music,type=custom,url=http://baidu.com,audio=http://baidu.com/1.mp3,title=音乐标题]
```

| 参数名 | 收 | 发 | 可能的值 | 说明 |
| --- | --- | --- | --- | --- |
| `type` |  | ✓ | `custom` | 表示音乐自定义分享 |
| `url` |  | ✓ | - | 点击后跳转目标 URL |
| `audio` |  | ✓ | - | 音乐 URL |
| `title` |  | ✓ | - | 标题 |
| `content` |  | ✓ | - | 发送时可选，内容描述 |
| `image` |  | ✓ | - | 发送时可选，图片 URL |

## 回复

```json
{
    "type": "reply",
    "data": {
        "id": "123456"
    }
}
```

```
[CQ:reply,id=123456]
```

| 参数名 | 收 | 发 | 可能的值 | 说明 |
| --- | --- | --- | --- | --- |
| `id` | ✓ | ✓ | - | 回复时引用的消息 ID |

## 合并转发 <Badge text="收"/>

```json
{
    "type": "forward",
    "data": {
        "id": "123456"
    }
}
```

```
[CQ:forward,id=123456]
```

| 参数名 | 收 | 发 | 可能的值 | 说明 |
| --- | --- | --- | --- | --- |
| `id` | ✓ |  | - | 合并转发 ID，需通过 [`get_forward_msg` API](../api/public.md#get_forward_msg-获取合并转发消息) 获取具体内容 |

## 合并转发节点 <Badge text="发"/>

```json
{
    "type": "node",
    "data": {
        "id": "123456"
    }
}
```

```
[CQ:node,id=123456]
```

| 参数名 | 收 | 发 | 可能的值 | 说明 |
| --- | --- | --- | --- | --- |
| `id` |  | ✓ | - | 转发的消息 ID |

## 合并转发自定义节点

> **注意**
>
> 接收时，此消息段不会直接出现在消息事件的 `message` 中，需通过 [`get_forward_msg` API](../api/public.md#get_forward_msg-获取合并转发消息) 获取。

**例 1**

```json
{
    "type": "node",
    "data": {
        "user_id": "10001000",
        "nickname": "某人",
        "content": "[CQ:face,id=123]哈喽～"
    }
}
```

```
[CQ:node,user_id=10001000,nickname=某人,content=&#91;CQ:face&#44;id=123&#93;哈喽～]
```

**例 2**

```json
{
    "type": "node",
    "data": {
        "user_id": "10001000",
        "nickname": "某人",
        "content": [
            {"type": "face", "data": {"id": "123"}},
            {"type": "text", "data": {"text": "哈喽～"}}
        ]
    }
}
```

| 参数名 | 收 | 发 | 可能的值 | 说明 |
| --- | --- | --- | --- | --- |
| `user_id` | ✓ | ✓ | - | 发送者 QQ 号 |
| `nickname` | ✓ | ✓ | - | 发送者昵称 |
| `content` | ✓ | ✓ | - | 消息内容，支持发送消息时的 `message` 数据类型，见 [API 的参数](../api/#参数) |

## XML 消息

```json
{
    "type": "xml",
    "data": {
        "data": "<?xml ..."
    }
}
```

```
[CQ:xml,data=<?xml ...]
```

| 参数名 | 收 | 发 | 可能的值 | 说明 |
| --- | --- | --- | --- | --- |
| `data` | ✓ | ✓ | - | XML 内容 |

## JSON 消息

```json
{
    "type": "json",
    "data": {
        "data": "{\"app\": ..."
    }
}
```

```
[CQ:json,data={"app": ...]
```

| 参数名 | 收 | 发 | 可能的值 | 说明 |
| --- | --- | --- | --- | --- |
| `data` | ✓ | ✓ | - | JSON 内容 |


# API

- [参数](#参数)
- [响应](#响应)
- [异步调用](#异步调用)
- [限速调用](#限速调用)
- [相关配置](#相关配置)

API 是 OneBot 向用户提供的操作接口，用户可通过 HTTP 请求或 WebSocket 消息等方式调用 API。

## 参数

API 调用需要指定 action（要进行的动作）和动作所需的参数。

在后面的 API 描述中，action 在标题中给出，如 `send_private_msg`；参数在「参数」小标题下给出，其中「数据类型」使用 JSON 中的名字，例如 `string`、`number` 等。

特别地，数据类型 `message` 表示该参数是一个消息类型的参数，在调用 API 时，`message` 类型的参数允许接受字符串、消息段数组、单个消息段对象三种类型的数据，关于消息格式的更多细节请查看 [消息](../message/)。

## 响应

OneBot 会对每个 API 调用返回一个 JSON 响应（除非是 HTTP 状态码不为 200 的情况），响应中的 `data` 字段包含 API 调用返回的数据内容。在后面的 API 描述中，将只给出 `data` 字段的内容，放在「响应数据」小标题下，而不再赘述 `status`、`retcode` 字段。

## 异步调用

所有 API 都可以通过给 action 附加后缀 `_async` 来进行异步调用，例如 `send_private_msg_async`、`send_msg_async`、`clean_data_dir_async`。

异步调用的响应中，`status` 字段为 `async`。

需要注意的是，虽然说以 `get_` 开头的那些接口也可以进行异步调用，但实际上客户端没有办法得知最终的调用结果，所以对这部分接口进行异步调用是没有意义的；另外，有一些接口本身就是异步执行的（返回的 `status` 为 `async`），此时使用 `_async` 后缀来调用不会产生本质上的区别。

## 限速调用

所有 API 都可以通过给 action 附加后缀 `_rate_limited` 来进行限速调用，例如 `send_private_msg_rate_limited`、`send_msg_rate_limited`，不过主要还是用在发送消息接口上，以避免消息频率过快导致腾讯封号。所有限速调用将会以指定速度**排队执行**，这个速度可在配置中指定。

限速调用的响应中，`status` 字段为 `async`。

## 相关配置

| 配置项 | 默认值 | 说明 |
| -------- | ------ | --- |
| `api.rate_limit_interval` | `500` | 限速 API 调用的排队间隔时间，单位毫秒 |

# 公开 API

- [`send_private_msg` 发送私聊消息](#send_private_msg-发送私聊消息)
- [`send_group_msg` 发送群消息](#send_group_msg-发送群消息)
- [`send_msg` 发送消息](#send_msg-发送消息)
- [`delete_msg` 撤回消息](#delete_msg-撤回消息)
- [`get_msg` 获取消息](#get_msg-获取消息)
- [`get_forward_msg` 获取合并转发消息](#get_forward_msg-获取合并转发消息)
- [`send_like` 发送好友赞](#send_like-发送好友赞)
- [`set_group_kick` 群组踢人](#set_group_kick-群组踢人)
- [`set_group_ban` 群组单人禁言](#set_group_ban-群组单人禁言)
- [`set_group_anonymous_ban` 群组匿名用户禁言](#set_group_anonymous_ban-群组匿名用户禁言)
- [`set_group_whole_ban` 群组全员禁言](#set_group_whole_ban-群组全员禁言)
- [`set_group_admin` 群组设置管理员](#set_group_admin-群组设置管理员)
- [`set_group_anonymous` 群组匿名](#set_group_anonymous-群组匿名)
- [`set_group_card` 设置群名片（群备注）](#set_group_card-设置群名片群备注)
- [`set_group_name` 设置群名](#set_group_name-设置群名)
- [`set_group_leave` 退出群组](#set_group_leave-退出群组)
- [`set_group_special_title` 设置群组专属头衔](#set_group_special_title-设置群组专属头衔)
- [`set_friend_add_request` 处理加好友请求](#set_friend_add_request-处理加好友请求)
- [`set_group_add_request` 处理加群请求／邀请](#set_group_add_request-处理加群请求邀请)
- [`get_login_info` 获取登录号信息](#get_login_info-获取登录号信息)
- [`get_stranger_info` 获取陌生人信息](#get_stranger_info-获取陌生人信息)
- [`get_friend_list` 获取好友列表](#get_friend_list-获取好友列表)
- [`get_group_info` 获取群信息](#get_group_info-获取群信息)
- [`get_group_list` 获取群列表](#get_group_list-获取群列表)
- [`get_group_member_info` 获取群成员信息](#get_group_member_info-获取群成员信息)
- [`get_group_member_list` 获取群成员列表](#get_group_member_list-获取群成员列表)
- [`get_group_honor_info` 获取群荣誉信息](#get_group_honor_info-获取群荣誉信息)
- [`get_cookies` 获取 Cookies](#get_cookies-获取-cookies)
- [`get_csrf_token` 获取 CSRF Token](#get_csrf_token-获取-csrf-token)
- [`get_credentials` 获取 QQ 相关接口凭证](#get_credentials-获取-qq-相关接口凭证)
- [`get_record` 获取语音](#get_record-获取语音)
- [`get_image` 获取图片](#get_image-获取图片)
- [`can_send_image` 检查是否可以发送图片](#can_send_image-检查是否可以发送图片)
- [`can_send_record` 检查是否可以发送语音](#can_send_record-检查是否可以发送语音)
- [`get_status` 获取运行状态](#get_status-获取运行状态)
- [`get_version_info` 获取版本信息](#get_version_info-获取版本信息)
- [`set_restart` 重启 OneBot 实现](#set_restart-重启-onebot-实现)
- [`clean_cache` 清理缓存](#clean_cache-清理缓存)

## `send_private_msg` 发送私聊消息

### 参数

| 字段名 | 数据类型 | 默认值 | 说明 |
| ----- | ------- | ----- | --- |
| `user_id` | number | - | 对方 QQ 号 |
| `message` | message | - | 要发送的内容 |
| `auto_escape` | boolean | `false` | 消息内容是否作为纯文本发送（即不解析 CQ 码），只在 `message` 字段是字符串时有效 |

### 响应数据

| 字段名 | 数据类型 | 说明 |
| ----- | ------- | --- |
| `message_id` | number (int32) | 消息 ID |

## `send_group_msg` 发送群消息

### 参数

| 字段名 | 数据类型 | 默认值 | 说明 |
| ----- | ------- | ----- | --- |
| `group_id` | number | - | 群号 |
| `message` | message | - | 要发送的内容 |
| `auto_escape` | boolean | `false` | 消息内容是否作为纯文本发送（即不解析 CQ 码），只在 `message` 字段是字符串时有效 |

### 响应数据

| 字段名 | 数据类型 | 说明 |
| ----- | ------- | --- |
| `message_id` | number (int32) | 消息 ID |

## `send_msg` 发送消息

### 参数

| 字段名 | 数据类型 | 默认值 | 说明 |
| ----- | ------- | ----- | --- |
| `message_type` | string | - | 消息类型，支持 `private`、`group`，分别对应私聊、群组，如不传入，则根据传入的 `*_id` 参数判断 |
| `user_id` | number | - | 对方 QQ 号（消息类型为 `private` 时需要） |
| `group_id` | number | - | 群号（消息类型为 `group` 时需要） |
| `message` | message | - | 要发送的内容 |
| `auto_escape` | boolean | `false` | 消息内容是否作为纯文本发送（即不解析 CQ 码），只在 `message` 字段是字符串时有效 |

### 响应数据

| 字段名 | 数据类型 | 说明 |
| ----- | ------- | --- |
| `message_id` | number (int32) | 消息 ID |

## `delete_msg` 撤回消息

### 参数

| 字段名 | 数据类型 | 默认值 | 说明 |
| ----- | ------- | ----- | --- |
| `message_id` | number (int32) | - | 消息 ID |

### 响应数据

无

## `get_msg` 获取消息

### 参数

| 字段名         | 数据类型  | 说明   |
| ------------ | ----- | ------ |
| `message_id` | number (int32) | 消息 ID |

### 响应数据

| 字段名         | 数据类型    | 说明       |
| ------------ | ------- | ---------- |
| `time`       | number (int32) | 发送时间   |
| `message_type` | string | 消息类型，同 [消息事件](../event/message.md) |
| `message_id` | number (int32) | 消息 ID     |
| `real_id` | number (int32) | 消息真实 ID     |
| `sender`     | object  | 发送人信息，同 [消息事件](../event/message.md) |
| `message`    | message | 消息内容   |

## `get_forward_msg` 获取合并转发消息

### 参数

| 字段名         | 数据类型   | 说明   |
| ------------ | ------ | ------ |
| `id` | string | 合并转发 ID |

### 响应数据

| 字段名 | 类型 | 说明 |
| --- | --- | --- |
| `message` | message | 消息内容，使用 [消息的数组格式](../message/array.md) 表示，数组中的消息段全部为 [`node` 消息段](../message/segment.md#合并转发自定义节点) |

## `send_like` 发送好友赞

### 参数

| 字段名 | 数据类型 | 默认值 | 说明 |
| ----- | ------- | ----- | --- |
| `user_id` | number | - | 对方 QQ 号 |
| `times` | number | 1 | 赞的次数，每个好友每天最多 10 次 |

### 响应数据

无

## `set_group_kick` 群组踢人

### 参数

| 字段名 | 数据类型 | 默认值 | 说明 |
| ----- | ------- | ----- | --- |
| `group_id` | number | - | 群号 |
| `user_id` | number | - | 要踢的 QQ 号  |
| `reject_add_request` | boolean | `false` | 拒绝此人的加群请求 |

### 响应数据

无

## `set_group_ban` 群组单人禁言

### 参数

| 字段名 | 数据类型 | 默认值 | 说明 |
| ----- | ------- | ----- | --- |
| `group_id` | number | - | 群号 |
| `user_id` | number | - | 要禁言的 QQ 号 |
| `duration` | number | `30 * 60` | 禁言时长，单位秒，0 表示取消禁言 |

### 响应数据

无

## `set_group_anonymous_ban` 群组匿名用户禁言

### 参数

| 字段名 | 数据类型 | 默认值 | 说明 |
| ----- | ------- | ----- | --- |
| `group_id` | number | - | 群号 |
| `anonymous` | object | - | 可选，要禁言的匿名用户对象（群消息上报的 `anonymous` 字段） |
| `anonymous_flag` 或 `flag` | string | - | 可选，要禁言的匿名用户的 flag（需从群消息上报的数据中获得） |
| `duration` | number | `30 * 60` | 禁言时长，单位秒，无法取消匿名用户禁言 |

上面的 `anonymous` 和 `anonymous_flag` 两者任选其一传入即可，若都传入，则使用 `anonymous`。

### 响应数据

无

## `set_group_whole_ban` 群组全员禁言

### 参数

| 字段名 | 数据类型 | 默认值 | 说明 |
| ----- | ------- | ----- | --- |
| `group_id` | number | - | 群号 |
| `enable` | boolean | `true` | 是否禁言 |

### 响应数据

无

## `set_group_admin` 群组设置管理员

### 参数

| 字段名 | 数据类型 | 默认值 | 说明 |
| ----- | ------- | ----- | --- |
| `group_id` | number | - | 群号 |
| `user_id` | number | - | 要设置管理员的 QQ 号 |
| `enable` | boolean | `true` | true 为设置，false 为取消 |

### 响应数据

无

## `set_group_anonymous` 群组匿名

### 参数

| 字段名 | 数据类型 | 默认值 | 说明 |
| ----- | ------- | ----- | --- |
| `group_id` | number | - | 群号 |
| `enable` | boolean | `true` | 是否允许匿名聊天 |

### 响应数据

无

## `set_group_card` 设置群名片（群备注）

### 参数

| 字段名 | 数据类型 | 默认值 | 说明 |
| ----- | ------- | ----- | --- |
| `group_id` | number | - | 群号 |
| `user_id` | number | - | 要设置的 QQ 号 |
| `card` | string | 空 | 群名片内容，不填或空字符串表示删除群名片 |

### 响应数据

无

## `set_group_name` 设置群名

### 参数

| 字段名   | 数据类型 | 说明 |
| -------- | ------ | ---- |
| `group_id` | number (int64) | 群号 |
| `group_name` | string | 新群名 |

### 响应数据

无

## `set_group_leave` 退出群组

### 参数

| 字段名 | 数据类型 | 默认值 | 说明 |
| ----- | ------- | ----- | --- |
| `group_id` | number | - | 群号 |
| `is_dismiss` | boolean | `false` | 是否解散，如果登录号是群主，则仅在此项为 true 时能够解散 |

### 响应数据

无

## `set_group_special_title` 设置群组专属头衔

### 参数

| 字段名 | 数据类型 | 默认值 | 说明 |
| ----- | ------- | ----- | --- |
| `group_id` | number | - | 群号 |
| `user_id` | number | - | 要设置的 QQ 号 |
| `special_title` | string | 空 | 专属头衔，不填或空字符串表示删除专属头衔 |
| `duration` | number | `-1` | 专属头衔有效期，单位秒，-1 表示永久，不过此项似乎没有效果，可能是只有某些特殊的时间长度有效，有待测试 |

### 响应数据

无

## `set_friend_add_request` 处理加好友请求

### 参数

| 字段名 | 数据类型 | 默认值 | 说明 |
| ----- | ------- | ----- | --- |
| `flag` | string | - | 加好友请求的 flag（需从上报的数据中获得） |
| `approve` | boolean | `true` | 是否同意请求 |
| `remark` | string | 空 | 添加后的好友备注（仅在同意时有效） |

### 响应数据

无

## `set_group_add_request` 处理加群请求／邀请

### 参数

| 字段名 | 数据类型 | 默认值 | 说明 |
| ----- | ------- | ----- | --- |
| `flag` | string | - | 加群请求的 flag（需从上报的数据中获得） |
| `sub_type` 或 `type` | string | - | `add` 或 `invite`，请求类型（需要和上报消息中的 `sub_type` 字段相符） |
| `approve` | boolean | `true` | 是否同意请求／邀请 |
| `reason` | string | 空 | 拒绝理由（仅在拒绝时有效） |

### 响应数据

无

## `get_login_info` 获取登录号信息

### 参数

无

### 响应数据

| 字段名 | 数据类型 | 说明 |
| ----- | ------- | --- |
| `user_id` | number (int64) | QQ 号 |
| `nickname` | string | QQ 昵称 |

## `get_stranger_info` 获取陌生人信息

### 参数

| 字段名 | 数据类型 | 默认值 | 说明 |
| ----- | ------- | ----- | --- |
| `user_id` | number | - | QQ 号 |
| `no_cache` | boolean | `false` | 是否不使用缓存（使用缓存可能更新不及时，但响应更快） |

### 响应数据

| 字段名 | 数据类型 | 说明 |
| ----- | ------- | --- |
| `user_id` | number (int64) | QQ 号 |
| `nickname` | string | 昵称 |
| `sex` | string | 性别，`male` 或 `female` 或 `unknown` |
| `age` | number (int32) | 年龄 |

## `get_friend_list` 获取好友列表

### 参数

无

### 响应数据

响应内容为 JSON 数组，每个元素如下：

| 字段名 | 数据类型 | 说明 |
| ----- | ------- | --- |
| `user_id` | number (int64) | QQ 号 |
| `nickname` | string | 昵称 |
| `remark` | string | 备注名 |

## `get_group_info` 获取群信息

### 参数

| 字段名 | 数据类型 | 默认值 | 说明 |
| ----- | ------- | ----- | --- |
| `group_id` | number | - | 群号 |
| `no_cache` | boolean | `false` | 是否不使用缓存（使用缓存可能更新不及时，但响应更快） |

### 响应数据

| 字段名 | 数据类型 | 说明 |
| ----- | ------- | --- |
| `group_id` | number (int64) | 群号 |
| `group_name` | string | 群名称 |
| `member_count` | number (int32) | 成员数 |
| `max_member_count` | number (int32) | 最大成员数（群容量） |

## `get_group_list` 获取群列表

### 参数

无

### 响应数据

响应内容为 JSON 数组，每个元素和上面的 `get_group_info` 接口相同。

## `get_group_member_info` 获取群成员信息

### 参数

| 字段名 | 数据类型 | 默认值 | 说明 |
| ----- | ------- | ----- | --- |
| `group_id` | number | - | 群号 |
| `user_id`  | number | - | QQ 号 |
| `no_cache` | boolean | `false` | 是否不使用缓存（使用缓存可能更新不及时，但响应更快） |

### 响应数据

| 字段名 | 数据类型 | 说明 |
| ----- | ------- | --- |
| `group_id` | number (int64) | 群号 |
| `user_id` | number (int64) | QQ 号 |
| `nickname` | string | 昵称 |
| `card` | string | 群名片／备注 |
| `sex` | string | 性别，`male` 或 `female` 或 `unknown` |
| `age` | number (int32) | 年龄 |
| `area` | string | 地区 |
| `join_time` | number (int32) | 加群时间戳 |
| `last_sent_time` | number (int32) | 最后发言时间戳 |
| `level` | string | 成员等级 |
| `role` | string | 角色，`owner` 或 `admin` 或 `member` |
| `unfriendly` | boolean | 是否不良记录成员 |
| `title` | string | 专属头衔 |
| `title_expire_time` | number (int32) | 专属头衔过期时间戳 |
| `card_changeable` | boolean | 是否允许修改群名片 |

## `get_group_member_list` 获取群成员列表

### 参数

| 字段名 | 数据类型 | 默认值 | 说明 |
| ----- | ------- | ----- | --- |
| `group_id` | number (int64) | - | 群号 |

### 响应数据

响应内容为 JSON 数组，每个元素的内容和上面的 `get_group_member_info` 接口相同，但对于同一个群组的同一个成员，获取列表时和获取单独的成员信息时，某些字段可能有所不同，例如 `area`、`title` 等字段在获取列表时无法获得，具体应以单独的成员信息为准。

## `get_group_honor_info` 获取群荣誉信息

### 参数

| 字段名 | 数据类型 | 默认值 | 说明 |
| ----- | ------- | ----- | --- |
| `group_id` | number (int64) | - | 群号 |
| `type` | string | - | 要获取的群荣誉类型，可传入 `talkative` `performer` `legend` `strong_newbie` `emotion` 以分别获取单个类型的群荣誉数据，或传入 `all` 获取所有数据 |

### 响应数据

| 字段名 | 数据类型 | 说明 |
| ----- | ------- | --- |
| `group_id` | number (int64) | 群号 |
| `current_talkative` | object | 当前龙王，仅 `type` 为 `talkative` 或 `all` 时有数据 |
| `talkative_list` | array | 历史龙王，仅 `type` 为 `talkative` 或 `all` 时有数据 |
| `performer_list` | array | 群聊之火，仅 `type` 为 `performer` 或 `all` 时有数据 |
| `legend_list` | array | 群聊炽焰，仅 `type` 为 `legend` 或 `all` 时有数据 |
| `strong_newbie_list` | array | 冒尖小春笋，仅 `type` 为 `strong_newbie` 或 `all` 时有数据 |
| `emotion_list` | array | 快乐之源，仅 `type` 为 `emotion` 或 `all` 时有数据 |

其中 `current_talkative` 字段的内容如下：

| 字段名 | 数据类型 | 说明 |
| ----- | ------- | --- |
| `user_id` | number (int64) | QQ 号 |
| `nickname` | string | 昵称 |
| `avatar` | string | 头像 URL |
| `day_count` | number (int32) | 持续天数 |

其它各 `*_list` 的每个元素是一个 JSON 对象，内容如下：

| 字段名 | 数据类型 | 说明 |
| ----- | ------- | --- |
| `user_id` | number (int64) | QQ 号 |
| `nickname` | string | 昵称 |
| `avatar` | string | 头像 URL |
| `description` | string | 荣誉描述 |

## `get_cookies` 获取 Cookies

### 参数

| 字段名 | 数据类型 | 默认值 | 说明 |
| ----- | ------- | ----- | --- |
| `domain` | string | 空 | 需要获取 cookies 的域名 |

### 响应数据

| 字段名 | 数据类型 | 说明 |
| ----- | ------- | --- |
| `cookies` | string | Cookies |

## `get_csrf_token` 获取 CSRF Token

### 参数

无

### 响应数据

| 字段名 | 数据类型 | 说明 |
| ----- | ------- | --- |
| `token` | number (int32) | CSRF Token |

## `get_credentials` 获取 QQ 相关接口凭证

即上面两个接口的合并。

### 参数

| 字段名 | 数据类型 | 默认值 | 说明 |
| ----- | ------- | ----- | --- |
| `domain` | string | 空 | 需要获取 cookies 的域名 |

### 响应数据

| 字段名 | 数据类型 | 说明 |
| ----- | ------- | --- |
| `cookies` | string | Cookies |
| `csrf_token` | number (int32) | CSRF Token |

## `get_record` 获取语音

> **提示**：要使用此接口，通常需要安装 ffmpeg，请参考 OneBot 实现的相关说明。

### 参数

| 字段名 | 数据类型 | 默认值 | 说明 |
| ----- | ------- | ----- | --- |
| `file` | string | - | 收到的语音文件名（消息段的 `file` 参数），如 `0B38145AA44505000B38145AA4450500.silk` |
| `out_format`  | string | - | 要转换到的格式，目前支持 `mp3`、`amr`、`wma`、`m4a`、`spx`、`ogg`、`wav`、`flac` |

### 响应数据

| 字段名 | 数据类型 | 说明 |
| ----- | ------- | --- |
| `file` | string | 转换后的语音文件路径，如 `/home/somebody/cqhttp/data/record/0B38145AA44505000B38145AA4450500.mp3` |

## `get_image` 获取图片

### 参数

| 字段名 | 数据类型 | 默认值 | 说明 |
| ----- | ------- | ----- | --- |
| `file` | string | - | 收到的图片文件名（消息段的 `file` 参数），如 `6B4DE3DFD1BD271E3297859D41C530F5.jpg` |

### 响应数据

| 字段名 | 数据类型 | 说明 |
| ----- | ------- | --- |
| `file` | string | 下载后的图片文件路径，如 `/home/somebody/cqhttp/data/image/6B4DE3DFD1BD271E3297859D41C530F5.jpg` |

## `can_send_image` 检查是否可以发送图片

### 参数

无

### 响应数据

| 字段名 | 数据类型 | 说明 |
| ----- | ------- | --- |
| `yes` | boolean | 是或否 |

## `can_send_record` 检查是否可以发送语音

### 参数

无

### 响应数据

| 字段名 | 数据类型 | 说明 |
| ----- | ------- | --- |
| `yes` | boolean | 是或否 |

## `get_status` 获取运行状态

### 参数

无

### 响应数据

| 字段名 | 数据类型 | 说明 |
| ----- | ------- | --- |
| `online` | boolean | 当前 QQ 在线，`null` 表示无法查询到在线状态 |
| `good` | boolean | 状态符合预期，意味着各模块正常运行、功能正常，且 QQ 在线 |
| …… | - | OneBot 实现自行添加的其它内容 |

通常情况下建议只使用 `online` 和 `good` 这两个字段来判断运行状态，因为根据 OneBot 实现的不同，其它字段可能完全不同。

## `get_version_info` 获取版本信息

### 参数

无

### 响应数据

| 字段名 | 数据类型 | 说明 |
| ----- | ------- | --- |
| `app_name` | string | 应用标识，如 `mirai-native` |
| `app_version` | string | 应用版本，如 `1.2.3` |
| `protocol_version` | string | OneBot 标准版本，如 `v11` |
| …… | - | OneBot 实现自行添加的其它内容 |

## `set_restart` 重启 OneBot 实现

由于重启 OneBot 实现同时需要重启 API 服务，这意味着当前的 API 请求会被中断，因此需要异步地重启，接口返回的 `status` 是 `async`。

### 参数

| 字段名 | 数据类型 | 默认值 | 说明 |
| ----- | ------- | ----- | --- |
| `delay` | number | `0` | 要延迟的毫秒数，如果默认情况下无法重启，可以尝试设置延迟为 2000 左右 |

### 响应数据

无

## `clean_cache` 清理缓存

用于清理积攒了太多的缓存文件。

### 参数

无

### 响应数据

无

# 隐藏 API

- [`.handle_quick_operation` 对事件执行快速操作](#handle_quick_operation-对事件执行快速操作)

隐藏 API 是不建议一般用户使用的，它们只应该在 OneBot 实现内部或由 SDK 和框架使用，因为不正确的使用可能造成程序运行不正常。

所有隐藏 API 都以点号（`.`）开头。

## `.handle_quick_operation` 对事件执行快速操作

关于事件的快速操作，见 [快速操作](../event/#快速操作)。

### 参数

| 字段名 | 数据类型 | 默认值 | 说明 |
| ----- | ------- | ----- | --- |
| `context` | object | - | 事件数据对象，可做精简，如去掉 `message` 等无用字段 |
| `operation` | object | - | 快速操作对象，例如 `{"ban": true, "reply": "请不要说脏话"}` |

### 响应数据

无

# 事件

- [内容字段](#内容字段)
- [快速操作](#快速操作)
- [数据类型](#数据类型)
- [相关配置](#相关配置)

事件是用户需要从 OneBot 被动接收的数据，有以下几个大类：

- 消息事件，包括私聊消息、群消息等
- 通知事件，包括群成员变动、好友变动等
- 请求事件，包括加群请求、加好友请求等
- 元事件，包括 OneBot 生命周期、心跳等

在所有能够推送事件的通信方式中（HTTP POST、正向和反向 WebSocket），事件都以 JSON 格式表示。

## 内容字段

每个事件都有 `time`、`self_id` 和 `post_type` 字段，如下：

| 字段名 | 数据类型 | 说明 |
| ----- | ------- | ---- |
| `time` | number (int64) | 事件发生的时间戳 |
| `self_id` | number (int64) | 收到事件的机器人 QQ 号 |
| `post_type` | string | 事件类型 |

其中 `post_type` 不同字段值表示的事件类型对应如下：

- `message`：消息事件
- `notice`：通知事件
- `request`：请求事件
- `meta_event`：元事件

其它字段随事件类型不同而有所不同，后面将在事件列表的「事件数据」小标题下给出。

某些字段的值是一些固定的值，在表格的「可能的值」中给出，如果「可能的值」为空则表示没有固定的可能性。

## 快速操作

如 [HTTP POST 的快速操作](../communication/http-post.md#快速操作) 中所说，HTTP POST 通信方式下，用户可在服务端返回的响应中指定快速操作，事件支持的快速操作在事件列表的「快速操作」小标题下给出。

在使用正向和反向 WebSocket 的情况下，可以通过 [`.handle_quick_operation`](../api/hidden.md#handle_quick_operation-对事件执行快速操作) API 伪造快速操作。

## 数据类型

在后面的事件列表中，「数据类型」使用 JSON 中的名字，例如 `string`、`number` 等。

特别地，数据类型 `message` 表示该字段是一个消息类型的字段。在事件数据中，`message` 的实际类型根据用户配置的消息格式的不同而不同，支持字符串和消息段数组两种格式；而在快速操作中，`message` 类型的字段允许接受字符串、消息段数组、单个消息段对象三种类型的数据。关于消息格式的更多细节请查看 [消息](../message/)。

## 相关配置

| 配置项名称 | 默认值 | 说明 |
| -------- | ------ | --- |
| `event.message_format` | `string` | 事件数据中的消息格式，`string` 为字符串格式，`array` 为消息段数组格式 |

# 消息事件

- [私聊消息](#私聊消息)
- [群消息](#群消息)

## 私聊消息

### 事件数据

| 字段名 | 数据类型 | 可能的值 | 说明 |
| ----- | ------- | ------- | ---- |
| `time` | number (int64) | - | 事件发生的时间戳 |
| `self_id` | number (int64) | - | 收到事件的机器人 QQ 号 |
| `post_type` | string | `message` | 上报类型 |
| `message_type` | string | `private` | 消息类型 |
| `sub_type` | string | `friend`、`group`、`other` | 消息子类型，如果是好友则是 `friend`，如果是群临时会话则是 `group` |
| `message_id` | number (int32) | - | 消息 ID |
| `user_id` | number (int64) | - | 发送者 QQ 号 |
| `message` | message | - | 消息内容 |
| `raw_message` | string | - | 原始消息内容 |
| `font` | number (int32) | - | 字体 |
| `sender` | object | - | 发送人信息 |

其中 `sender` 字段的内容如下：

| 字段名 | 数据类型 | 说明 |
| ----- | ------ | ---- |
| `user_id` | number (int64) | 发送者 QQ 号 |
| `nickname` | string | 昵称 |
| `sex` | string | 性别，`male` 或 `female` 或 `unknown` |
| `age` | number (int32) | 年龄 |

需要注意的是，`sender` 中的各字段是尽最大努力提供的，也就是说，不保证每个字段都一定存在，也不保证存在的字段都是完全正确的（缓存可能过期）。

### 快速操作

| 字段名 | 数据类型 | 说明 | 默认情况 |
| ----- | ------- | --- | ------- |
| `reply` | message | 要回复的内容 | 不回复 |
| `auto_escape` | boolean | 消息内容是否作为纯文本发送（即不解析 CQ 码），只在 `reply` 字段是字符串时有效 | 不转义 |

## 群消息

### 事件数据

| 字段名 | 数据类型 | 可能的值 | 说明 |
| ----- | ------- | ------- | --- |
| `time` | number (int64) | - | 事件发生的时间戳 |
| `self_id` | number (int64) | - | 收到事件的机器人 QQ 号 |
| `post_type` | string | `message` | 上报类型 |
| `message_type` | string | `group` | 消息类型 |
| `sub_type` | string | `normal`、`anonymous`、`notice` | 消息子类型，正常消息是 `normal`，匿名消息是 `anonymous`，系统提示（如「管理员已禁止群内匿名聊天」）是 `notice` |
| `message_id` | number (int32) | - | 消息 ID |
| `group_id` | number (int64) | - | 群号 |
| `user_id` | number (int64) | - | 发送者 QQ 号 |
| `anonymous` | object | - | 匿名信息，如果不是匿名消息则为 null |
| `message` | message | - | 消息内容 |
| `raw_message` | string | - | 原始消息内容 |
| `font` | number (int32) | - | 字体 |
| `sender` | object | - | 发送人信息 |

其中 `anonymous` 字段的内容如下：

| 字段名 | 数据类型 | 说明 |
| ----- | ------ | ---- |
| `id` | number (int64) | 匿名用户 ID |
| `name` | string | 匿名用户名称 |
| `flag` | string | 匿名用户 flag，在调用禁言 API 时需要传入 |

`sender` 字段的内容如下：

| 字段名 | 数据类型 | 说明 |
| ----- | ------ | ---- |
| `user_id` | number (int64) | 发送者 QQ 号 |
| `nickname` | string | 昵称 |
| `card` | string | 群名片／备注 |
| `sex` | string | 性别，`male` 或 `female` 或 `unknown` |
| `age` | number (int32) | 年龄 |
| `area` | string | 地区 |
| `level` | string | 成员等级 |
| `role` | string | 角色，`owner` 或 `admin` 或 `member` |
| `title` | string | 专属头衔 |

需要注意的是，`sender` 中的各字段是尽最大努力提供的，也就是说，不保证每个字段都一定存在，也不保证存在的字段都是完全正确的（缓存可能过期）。尤其对于匿名消息，此字段不具有参考价值。

### 快速操作

| 字段名 | 数据类型 | 说明 | 默认情况 |
| ----- | ------- | --- | ------- |
| `reply` | message | 要回复的内容 | 不回复 |
| `auto_escape` | boolean | 消息内容是否作为纯文本发送（即不解析 CQ 码），只在 `reply` 字段是字符串时有效 | 不转义 |
| `at_sender` | boolean | 是否要在回复开头 at 发送者（自动添加），发送者是匿名用户时无效 | at 发送者 |
| `delete` | boolean | 撤回该条消息 | 不撤回 |
| `kick` | boolean | 把发送者踢出群组（需要登录号权限足够），**不拒绝**此人后续加群请求，发送者是匿名用户时无效 | 不踢 |
| `ban` | boolean | 把发送者禁言 `ban_duration` 指定时长，对匿名用户也有效 | 不禁言 |
| `ban_duration` | number | 禁言时长 | 30 分钟 |

# 通知事件

- [群文件上传](#群文件上传)
- [群管理员变动](#群管理员变动)
- [群成员减少](#群成员减少)
- [群成员增加](#群成员增加)
- [群禁言](#群禁言)
- [好友添加](#好友添加)
- [群消息撤回](#群消息撤回)
- [好友消息撤回](#好友消息撤回)
- [群内戳一戳](#群内戳一戳)
- [群红包运气王](#群红包运气王)
- [群成员荣誉变更](#群成员荣誉变更)

## 群文件上传

### 事件数据

| 字段名 | 数据类型 | 可能的值 | 说明 |
| ----- | ------ | ------- | ---- |
| `time` | number (int64) | - | 事件发生的时间戳 |
| `self_id` | number (int64) | - | 收到事件的机器人 QQ 号 |
| `post_type` | string | `notice` | 上报类型 |
| `notice_type` | string | `group_upload` | 通知类型 |
| `group_id` | number (int64) | - | 群号 |
| `user_id` | number (int64) | - | 发送者 QQ 号 |
| `file` | object | - | 文件信息 |

其中 `file` 字段的内容如下：

| 字段名 | 数据类型 | 说明 |
| ----- | ------ | ---- |
| `id` | string | 文件 ID |
| `name` | string | 文件名 |
| `size` | number (int64) | 文件大小（字节数） |
| `busid` | number (int64) | busid（目前不清楚有什么作用） |

## 群管理员变动

### 事件数据

| 字段名 | 数据类型 | 可能的值 | 说明 |
| ----- | ------ | -------- | --- |
| `time` | number (int64) | - | 事件发生的时间戳 |
| `self_id` | number (int64) | - | 收到事件的机器人 QQ 号 |
| `post_type` | string | `notice` | 上报类型 |
| `notice_type` | string | `group_admin` | 通知类型 |
| `sub_type` | string | `set`、`unset` | 事件子类型，分别表示设置和取消管理员 |
| `group_id` | number (int64) | - | 群号 |
| `user_id` | number (int64) | - | 管理员 QQ 号 |

## 群成员减少

### 事件数据

| 字段名 | 数据类型 | 可能的值 | 说明 |
| ----- | ------ | -------- | --- |
| `time` | number (int64) | - | 事件发生的时间戳 |
| `self_id` | number (int64) | - | 收到事件的机器人 QQ 号 |
| `post_type` | string | `notice` | 上报类型 |
| `notice_type` | string | `group_decrease` | 通知类型 |
| `sub_type` | string | `leave`、`kick`、`kick_me` | 事件子类型，分别表示主动退群、成员被踢、登录号被踢 |
| `group_id` | number (int64) | - | 群号 |
| `operator_id` | number (int64) | - | 操作者 QQ 号（如果是主动退群，则和 `user_id` 相同） |
| `user_id` | number (int64) | - | 离开者 QQ 号 |

## 群成员增加

### 事件数据

| 字段名 | 数据类型 | 可能的值 | 说明 |
| ----- | ------ | -------- | --- |
| `time` | number (int64) | - | 事件发生的时间戳 |
| `self_id` | number (int64) | - | 收到事件的机器人 QQ 号 |
| `post_type` | string | `notice` | 上报类型 |
| `notice_type` | string | `group_increase` | 通知类型 |
| `sub_type` | string | `approve`、`invite` | 事件子类型，分别表示管理员已同意入群、管理员邀请入群 |
| `group_id` | number (int64) | - | 群号 |
| `operator_id` | number (int64) | - | 操作者 QQ 号 |
| `user_id` | number (int64) | - | 加入者 QQ 号 |

## 群禁言

### 事件数据

| 字段名 | 数据类型 | 可能的值 | 说明 |
| ----- | ------ | -------- | --- |
| `time` | number (int64) | - | 事件发生的时间戳 |
| `self_id` | number (int64) | - | 收到事件的机器人 QQ 号 |
| `post_type` | string | `notice` | 上报类型 |
| `notice_type` | string | `group_ban` | 通知类型 |
| `sub_type` | string | `ban`、`lift_ban` | 事件子类型，分别表示禁言、解除禁言 |
| `group_id` | number (int64) | - | 群号 |
| `operator_id` | number (int64) | - | 操作者 QQ 号 |
| `user_id` | number (int64) | - | 被禁言 QQ 号 |
| `duration` | number (int64) | - | 禁言时长，单位秒 |

## 好友添加

### 事件数据

| 字段名 | 数据类型 | 可能的值 | 说明 |
| ----- | ------ | -------- | --- |
| `time` | number (int64) | - | 事件发生的时间戳 |
| `self_id` | number (int64) | - | 收到事件的机器人 QQ 号 |
| `post_type` | string | `notice` | 上报类型 |
| `notice_type` | string | `friend_add` | 通知类型 |
| `user_id` | number (int64) | - | 新添加好友 QQ 号 |

## 群消息撤回

### 事件数据

| 字段名          | 数据类型   | 可能的值       | 说明           |
| ------------- | ------ | -------------- | -------------- |
| `time` | number (int64) | - | 事件发生的时间戳 |
| `self_id` | number (int64) | - | 收到事件的机器人 QQ 号 |
| `post_type`   | string | `notice`       | 上报类型       |
| `notice_type` | string | `group_recall` | 通知类型       |
| `group_id`    | number (int64)  |                | 群号           |
| `user_id`     | number (int64)  |                | 消息发送者 QQ 号   |
| `operator_id` | number (int64)  |                | 操作者 QQ 号  |
| `message_id`  | number (int64)  |                | 被撤回的消息 ID |

## 好友消息撤回

### 事件数据

| 字段名          | 数据类型   | 可能的值       | 说明           |
| ------------- | ------ | -------------- | -------------- |
| `time` | number (int64) | - | 事件发生的时间戳 |
| `self_id` | number (int64) | - | 收到事件的机器人 QQ 号 |
| `post_type`   | string | `notice`       | 上报类型       |
| `notice_type` | string | `friend_recall`| 通知类型       |
| `user_id`     | number (int64)  |                | 好友 QQ 号        |
| `message_id`  | number (int64)  |                | 被撤回的消息 ID |

## 群内戳一戳

### 上报数据

| 字段          | 类型   | 可能的值       | 说明           |
| ------------- | ------ | -------------- | -------------- |
| `time` | number (int64) | - | 事件发生的时间戳 |
| `self_id` | number (int64) | - | 收到事件的机器人 QQ 号 |
| `post_type`   | string | `notice`       | 上报类型       |
| `notice_type` | string | `notify` | 消息类型       |
| `sub_type` | string | `poke` | 提示类型 |
| `group_id` | int64 |  | 群号 |
| `user_id`     | int64  |                | 发送者 QQ 号 |
| `target_id` | int64 | | 被戳者 QQ 号 |

## 群红包运气王

### 上报数据

| 字段          | 类型   | 可能的值       | 说明           |
| ------------- | ------ | -------------- | -------------- |
| `time` | number (int64) | - | 事件发生的时间戳 |
| `self_id` | number (int64) | - | 收到事件的机器人 QQ 号 |
| `post_type`   | string | `notice`       | 上报类型       |
| `notice_type` | string | `notify` | 消息类型       |
| `sub_type` | string | `lucky_king` | 提示类型 |
| `group_id` | int64 |  | 群号 |
| `user_id`     | int64  |                | 红包发送者 QQ 号 |
| `target_id` | int64 | | 运气王 QQ 号 |

## 群成员荣誉变更

### 上报数据

| 字段          | 类型   | 可能的值       | 说明           |
| ------------- | ------ | -------------- | -------------- |
| `time` | number (int64) | - | 事件发生的时间戳 |
| `self_id` | number (int64) | - | 收到事件的机器人 QQ 号 |
| `post_type`   | string | `notice`       | 上报类型       |
| `notice_type` | string | `notify` | 消息类型       |
| `sub_type` | string | `honor` | 提示类型 |
| `group_id` | int64 |  | 群号 |
| `honor_type` | string | `talkative`、`performer`、`emotion` | 荣誉类型，分别表示龙王、群聊之火、快乐源泉 |
| `user_id`     | int64  |   | 成员 QQ 号 |

# 请求事件

- [加好友请求](#加好友请求)
- [加群请求／邀请](#加群请求邀请)

## 加好友请求

### 事件数据

| 字段名 | 数据类型 | 可能的值 | 说明 |
| ----- | ------ | -------- | --- |
| `time` | number (int64) | - | 事件发生的时间戳 |
| `self_id` | number (int64) | - | 收到事件的机器人 QQ 号 |
| `post_type` | string | `request` | 上报类型 |
| `request_type` | string | `friend` | 请求类型 |
| `user_id` | number (int64) | - | 发送请求的 QQ 号 |
| `comment` | string | - | 验证信息 |
| `flag` | string | - | 请求 flag，在调用处理请求的 API 时需要传入 |

### 快速操作

| 字段名 | 数据类型 | 说明 | 默认情况 |
| ----- | ------- | --- | ------- |
| `approve` | boolean | 是否同意请求 | 不处理 |
| `remark` | string  | 添加后的好友备注（仅在同意时有效） | 无备注 |

## 加群请求／邀请

### 事件数据

| 字段名 | 数据类型 | 可能的值 | 说明 |
| ----- | ------ | -------- | --- |
| `time` | number (int64) | - | 事件发生的时间戳 |
| `self_id` | number (int64) | - | 收到事件的机器人 QQ 号 |
| `post_type` | string | `request` | 上报类型 |
| `request_type` | string | `group` | 请求类型 |
| `sub_type` | string | `add`、`invite` | 请求子类型，分别表示加群请求、邀请登录号入群 |
| `group_id` | number (int64) | - | 群号 |
| `user_id` | number (int64) | - | 发送请求的 QQ 号 |
| `comment` | string | - | 验证信息 |
| `flag` | string | - | 请求 flag，在调用处理请求的 API 时需要传入 |

### 快速操作

| 字段名 | 数据类型 | 说明 | 默认情况 |
| ----- | ------- | --- | ------- |
| `approve` | boolean | 是否同意请求／邀请 | 不处理 |
| `reason` | string | 拒绝理由（仅在拒绝时有效） | 无理由 |

# 元事件

- [生命周期](#生命周期)
- [心跳](#心跳)
- [相关配置](#相关配置)

消息、通知、请求三大类事件是与聊天软件直接相关的、机器人真实接收到的事件，除了这些，OneBot 自己还会产生一类事件，这里称之为「元事件」，例如生命周期事件、心跳事件等，这类事件与 OneBot 本身的运行状态有关，而与聊天软件无关。元事件的上报方式和普通事件完全一样。

## 生命周期

| 字段名 | 数据类型 | 可能的值 | 说明 |
| ----- | ------ | -------- | --- |
| `time` | number (int64) | - | 事件发生的时间戳 |
| `self_id` | number (int64) | - | 收到事件的机器人 QQ 号 |
| `post_type` | string | `meta_event` | 上报类型 |
| `meta_event_type` | string | `lifecycle` | 元事件类型 |
| `sub_type` | string | `enable`、`disable`、`connect` | 事件子类型，分别表示 OneBot 启用、停用、WebSocket 连接成功 |

**注意，目前生命周期元事件中，只有 HTTP POST 的情况下可以收到 `enable` 和 `disable`，只有正向 WebSocket 和反向 WebSocket 可以收到 `connect`。**

## 心跳

| 字段名 | 数据类型 | 可能的值 | 说明 |
| ----- | ------ | -------- | --- |
| `time` | number (int64) | - | 事件发生的时间戳 |
| `self_id` | number (int64) | - | 收到事件的机器人 QQ 号 |
| `post_type` | string | `meta_event` | 上报类型 |
| `meta_event_type` | string | `heartbeat` | 元事件类型 |
| `status` | object | - | 状态信息 |
| `interval` | number (int64) | - | 到下次心跳的间隔，单位毫秒 |

其中 `status` 字段的内容和 `get_status` 接口的快速操作相同。

## 相关配置

| 配置项 | 默认值 | 说明 |
| -------- | ------ | --- |
| `heartbeat.enable` | `false` | 是否启用心跳机制 |
| `heartbeat.interval` | `15000` | 产生心跳元事件的时间间隔，单位毫秒 |