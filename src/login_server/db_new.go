package main

import (
	"mm_server/libs/log"
	"mm_server/src/login_server/login_db"

	"github.com/huoshan017/mysql-go/manager"
)

var db_new mysql_manager.DB
var db_new_tables login_db.TablesManager
var account_table *login_db.Account_Table
var ban_player_table *login_db.BanPlayer_Table

func db_new_init(db_config string) bool {
	if !db_new.LoadConfig(db_config) {
		log.Error("db new load define failed")
		return false
	}
	err := db_new.Connect(config.MYSQL_IP, config.MYSQL_ACCOUNT, config.MYSQL_PWD, config.MYSQL_NAME)
	if err != nil {
		log.Error("db_new connect err: %v", err.Error())
		return false
	}

	db_new_tables.Init(&db_new)
	account_table = db_new_tables.Get_Account_Table()
	ban_player_table = db_new_tables.Get_BanPlayer_Table()

	db_new.GoRun()

	return true
}