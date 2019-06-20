package main

import (
	"mm_server/libs/log"
	"mm_server/src/login_server/login_db"

	"github.com/huoshan017/mysql-go/manager"
)

var db_new mysql_manager.DB
var account_table *login_db.AccountsTable
var ban_player_table *login_db.BanPlayerTable

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

	db_tables := login_db.NewTablesManager(&db_new)
	account_table = db_tables.GetAccountsTable()
	ban_player_table = db_tables.GetBanPlayerTable()

	db_new.GoRun()

	return true
}
