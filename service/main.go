package main

import (
	"runtime"
)

func main() {

	runtime.GOMAXPROCS(runtime.NumCPU())

	// 初始化数据库
	InitDB(EnvConfig["DATABASE_USER"].(string),
		EnvConfig["DATABASE_PWD"].(string),
		EnvConfig["DATABASE_PORT"].(string),
		EnvConfig["DATABASE_IP"].(string),
		EnvConfig["DATABASE_NAME"].(string))

	// 初始化redis
	InitRedis()

	// 初始化账号集
	InitAccount()

	// 初始化服务器
	InitServer(EnvConfig["SERVER_PORT"].(string))

	// 初始化Logger
	if EnvConfig["LOG_IN_FILE"].(bool) {
		InitLogger()
	}
}
