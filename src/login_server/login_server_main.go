package main

import (
	"fmt"
	"mm_server/libs/log"
	"mm_server/src/server_config"
	"mm_server/src/share_data"
)

var (
	db_use_new = false
)

var config server_config.LoginServerConfig
var shutingdown bool
var dbc DBC
var server_list share_data.ServerList

func main() {
	defer func() {
		log.Event("关闭服务器", nil)
		if err := recover(); err != nil {
			log.Stack(err)
		}
		if server != nil {
			server.Shutdown()
		}
		log.Close()
	}()

	if !server_config.ServerConfigLoad("login_server.json", &config) {
		fmt.Printf("载入LoginServer配置失败\n")
		return
	}

	fmt.Printf("配置:服务器ID\n", config.ServerId)
	fmt.Printf("配置:服务器名称\n", config.ServerName)
	fmt.Printf("配置:服务器地址(对Client)\n", config.ListenClientIP)
	fmt.Printf("配置:服务器地址(对Game)\n", config.ListenGameIP)

	server_list.ReadConfig(server_config.GetConfPathFile("server_list.json"))

	if !global_config_load() {
		fmt.Printf("global_config_load failed !\n")
		return
	}

	fmt.Printf("连接数据库\n", config.MYSQL_NAME, log.Property{"地址", config.MYSQL_IP})

	var err error
	if !db_use_new {
		err = dbc.Conn(config.MYSQL_NAME, config.MYSQL_IP, config.MYSQL_ACCOUNT, config.MYSQL_PWD, func() string {
			if config.MYSQL_COPY_PATH == "" {
				return config.GetDBBackupPath()
			} else {
				return config.MYSQL_COPY_PATH
			}
		}())
		if err != nil {
			fmt.Printf("连接数据库失败 %v\n", err)
			return
		} else {
			fmt.Printf("连接数据库成功\n", nil)
			go dbc.Loop()
		}
	} else {
		/*if !db_new_init(config.DB_DEFINE) {
			return
		}*/
		fmt.Printf("db new init success\n")
	}

	if !signal_mgr.Init() {
		fmt.Printf("signal_mgr init failed\n")
		return
	}

	if !db_use_new {
		if nil != dbc.Preload() {
			fmt.Printf("dbc Preload Failed !!\n")
			return
		} else {
			fmt.Printf("dbc Preload succeed !!\n")
		}
	}

	server = new(LoginServer)
	if !server.Init() {
		return
	}

	if signal_mgr.IfClosing() {
		return
	}

	if !game_agent_manager.Init() {
		return
	}

	center_conn.Init()
	go center_conn.Start()

	err = game_agent_manager.Start()
	if err != nil {
		return
	}

	server.Start(config.UseHttps)
}
