# waifu!D
waifu!d 是一个 rss 附件下载器, 同时也可以自己写插件支持更多的功能

目前支持在 telegram 机器人上订阅与取消订阅

---
## 使用方法

### 命令行
```bash
waifud [-c, --config=CONFIG_PATH]
CONFIG_PATH :
	config file (default "config.toml")
```

### 配置
```toml
[service]
    [service.database]
    min-ttl = 600					# rss 最小 ttl
    saved-path = "waifud.gob"       # database 存储路径

    [service.telebot]
    token = ""						# telebot token

    [service.aria2c]
    url = "http://127.0.0.1:6800/jsonrpc"		# aria2 rpc 地址
    secret = ""						# aria2 rpc 密码(?)
```

### telebot
```
/sub url
/ubsub url
```
目前没有返回 (

效果如下

![]( https://i.imgur.com/rm4ovay.png)
