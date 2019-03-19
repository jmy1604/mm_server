package main

import (
	"fmt"
	"mm_server/libs/log"
	"mm_server/src/server_config"
)

var config server_config.TestClientConfig
var shutingdown bool

func main() {
	defer func() {
		log.Event("关闭测试客户端", nil)
		if err := recover(); err != nil {
			log.Stack(err)
		}
		test_client.Shutdown()
		log.Close()
	}()

	if !server_config.ServerConfigLoad("test_client.json", &config) {
		fmt.Printf("载入TestClient配置失败")
		return
	}

	msg_handler_mgr.Init()

	game_conn_mgr.Init()

	if !test_client.Init() {
		return
	}

	test_client.Start()
}
