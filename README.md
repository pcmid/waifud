# waifu!D
waifu!d 是一个 rss 附件下载器, 同时也可以自己写插件支持更多的功能

目前支持在 telegram 机器人上订阅与取消订阅

---
## 使用方法

### 命令行
```bash
waifud [-c, --config=CONFIG_PATH]
CONFIG_PATH :
	config file (default "config.yaml")
```

### 配置
```yaml
service:
  puller:
    enable: true                # 是否启用; 默认: true
    saved-path: "waifud.gob"    # 订阅存储路径
    min-ttl: 600                # rss 最小 ttl

  telebot:
    enable: true                # 是否启用; 默认: true
    token: ""                   # telebot token

  aria2c:
    enable: true                # 是否启用; 默认: true
    url: ""                     # aria2 rpc 地址
    secret: ""                  # aria2 rpc 密码
    session: ""                 # session 保存路径; 空为不保存
```

### telebot
```
/ping  
/sub url [dir]	    # 添加订阅 aria2全局下的相对目录
/ubsub url	        # 取消订阅
/getsub		        # 查看全部订阅
/link url [dir]	    # 直接下载链接
/status             # 查看下载项与状态
```

效果如下

![Imgur](https://imgur.com/51a2jN9.png)
