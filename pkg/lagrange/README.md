# pkg/lagrange
用于 [Jel1ySpot/GoroBot](https://github.com/Jel1ySpot/GoroBot) 的基于 [LagrangeDev/LagrangeGo](https://github.com/LagrangeDev/LagrangeGo) 的 QQNT Protocol 适配器

## 快速开始
1. 载入包
    - `import "github.com/Jel1ySpot/GoroBot/pkg/lagrange"`
2. 创建实例并使用
    - `lgr := lagrange.Create(); grb.Use(lgr)`
3. [填写配置文件](#填写配置文件)
4. [登录](#登录)

### 填写配置文件
首次运行后，会在 `conf/lagrange` 目录下创建空配置文件。 配置示例：
```json
{
  "account": {
    "password": "暂不支持密码登录，可以留空",
    "sig_path": "./sig.bin",
    "uin": 1919810
  },
  "app_info": "linux 3.2.15-30366",
  "command_prefix": "/",
  "ignore_self": true,
  "music_sign_server_url": "",
  "sign_server_url": "sign_url需要与上面app_info的版本号对应"
}
```

### 登录
默认使用二维码登录。启动程序后如果配置文件无误，会在控制台输出一个链接。打开链接使用手机QQ扫描二维码即可登录。