# wechat-go

wechat-go是一个内部模块化使用的微信sdk。
会提供多种语言客户端，如php, go。对于go语言的项目也支持直接依赖到项目当中而不抽出来作为微服务模块。

## 特点

- 业务隔离，与你的业务代码无关
- 多公众号(小程序)管理
- 微服务模块化：如果多个项目，可以共用
- 即开即用，使用go编写产出二进制文件直接运行即可
- 搭建国内首个微信本地沙箱测试环境
- 高效的函数式编程

## 客户端

- [go](https://github.com/chenhg5/go-wechat-client)

## 运行

现在仍未完成开发，故而不提供二进制文件。如果想要体验，请检出本项目到 GOPATH 路径下。然后执行：

```
make deps
make
```

新建config.go 例子：

```
package main

var EnvConfig = map[string]interface{}{
	"SERVER_PORT":           "4000",
	"DATABASE_IP":           "127.0.0.1",
	"DATABASE_PORT":         "3306",
	"DATABASE_USER":         "root",
	"DATABASE_PWD":          "root",
	"DATABASE_NAME":         "wechat",
	"DATABASE_MAX_IDLE_CON": 50,  // 连接池连接数
	"DATABASE_MAX_OPEN_CON": 150, // 最大连接数

	"REDIS_IP":       "127.0.0.1",
	"REDIS_PORT":     "6379",
	"REDIS_PASSWORD": "",
	"REDIS_DB":       1,
}
```

## 接口

- 全局
    - [x] 获取access_token
- 网页授权
    - [x] 获取特殊的网页授权access_token
    - [x] 刷新token
    - [x] 拉取用户信息(需scope为 snsapi_userinfo)
    - [x] 检验token有效性
- 模板消息    
    - [x] 发送模板消息
- 小程序    
    - [x] 小程序获取sessionkey
    - [x] 获取小程序码
    - [x] 获取小程序码无限制
    - [x] 获取小程序码二维码
    - [x] 发送小程序服务通知
- 微信支付
    - [x] 下订单
- 公众号管理

## TODO

- [ ] rfc协议支持