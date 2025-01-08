# 快速入门

## 安装框架
> 本框架仍然处于开发状态，代码迭代速度快，建议使用第二种安装方法。

### 从代码包导入
- `go get github.com/Jel1ySpot/GoroBot`
- 在 `main.go` 中导入包

### 从仓库导入
Fork 并克隆仓库，在 `main.go` 文件与 `plugins/` 文件夹中编写业务逻辑代码（已添加至 `.gitignore`）。
这样既能够适应框架的快速迭代，还可以更方便向本仓库提交 PR 
（对，我 ~~们~~ 鼓励任何人提交修复或功能代码，也欢迎在 github issue 中提出在使用过程中的任何问题与建议）

示范：
```shell
#/bin/bash
git clone https://github.com/PathtoYou/rRepo

cd rRpot
mkdir plugins
touch main.go
```

## 最小示例
这是一个最小示例：
```go
package main

import (
	GoroBot "github.com/Jel1ySpot/GoroBot/pkg/core"
)

func main() {
	grb := GoroBot.Create()

	if err := grb.Run(); err != nil {
		panic(err)
	}
}
```
第一次运行上面的代码会报错，提示我们要填写配置文件（conf/config.json）。我们可以按照以下格式填写：
```json5
{
  "log_level": 1, // 日志等级。 0:Debug, 1:Info, 2:Warning, 3:Error
  "owner": { // 机器人所有者
    "qq": "你的QQ号" // 格式："平台": "ID"
  }
}
```
填写完成后再次启动，可以发现报错消失了，但控制台没有输出。这是因为这段代码没有载入任何逻辑代码。在这段代码中，我们创建了一个 GoroBot 实例（为了方便，在文档中会用 grb 代表 GoroBot 实例），并通过 `grb.Run()` 运行了它。

## 引入适配器
GoroBot 框架使用适配器支持各种 IM 平台。具体支持列表见 [**支持平台**](README.md#支持平台)。适配器也是一种插件。让我们尝试引入一个适配器：
```go
import LgrClient "github.com/Jel1ySpot/GoroBot/pkg/lagrange" // 引入模块

// 在 main 函数中：
grb.Use(LgrClient.Create())
```
插件使用 `Create()` 创建一个服务，我们可以使用 `grb.Use(*Service)` 来使用一个服务。运行这段代码，如无意外还需要填写配置文件，详细请见 [pkg/lagrange](https://github.com/Jel1ySpot/GoroBot/tree/master/pkg/lagrange)。
配置好服务后再次运行，没有意外的话机器人服务就启动成功了。

# 使用插件
同样是一个例子：
```go
import "github.com/Jel1ySpot/GoroBot/example_plugin/message_logger" // 引入模块

// 在 main 函数中
grb.Use(message_logger.Create())
```
再次运行代码，尝试向机器人账号发送消息，如果控制台中出现了刚才发送的消息，代表框架已经搭建成功了！开始编写属于你自己的 IM Chatbot 吧！

## 接下来应该做什么
- [插件列表](plugins_list.md)
- [api 文档](api/README.md)