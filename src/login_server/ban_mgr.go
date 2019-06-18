package main

import (
	"sync"

	"mm_server/libs/log"
	"mm_server/src/login_server/login_db"
)

type BanMgr struct {
	ban_players map[string]*login_db.BanPlayer
	locker      sync.RWMutex
}

var ban_mgr BanMgr

func (this *BanMgr) Init() bool {
	ban_players, err := ban_player_table.SelectAllMapRecords()
	if err != nil {
		log.Error("ban mgr init err: %v", err.Error())
		return false
	}
	this.ban_players = ban_players
	return true
}

func (this *BanMgr) Has(unique_id string) bool {
	this.locker.RLock()
	defer this.locker.RUnlock()
	return this.ban_players[unique_id] != nil
}

func (this *BanMgr) Get(unique_id string) *login_db.BanPlayer {
	this.locker.RLock()
	defer this.locker.RUnlock()
	return this.ban_players[unique_id]
}

func (this *BanMgr) Insert(record *login_db.BanPlayer) bool {
	this.locker.Lock()
	defer this.locker.Unlock()

	if _, o := this.ban_players[record.Get_UniqueId()]; o {
		return false
	}
	this.ban_players[record.Get_UniqueId()] = record
	return true
}
