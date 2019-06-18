package main

import (
	"sync"

	"mm_server/libs/log"
	"mm_server/src/login_server/login_db"
)

type AccountRecordMgr struct {
	accounts map[string]*login_db.Account
	locker   sync.RWMutex
}

var account_record_mgr AccountRecordMgr

func (this *AccountRecordMgr) Init() bool {
	accounts, err := account_table.SelectAllMapRecords()
	if err != nil {
		log.Error("account new manager init err: %v", err.Error())
		return false
	}

	this.accounts = accounts
	return true
}

func (this *AccountRecordMgr) Has(account string) bool {
	this.locker.RLock()
	defer this.locker.RUnlock()
	return this.accounts[account] != nil
}

func (this *AccountRecordMgr) Get(account string) *login_db.Account {
	this.locker.RLock()
	defer this.locker.RUnlock()
	return this.accounts[account]
}

func (this *AccountRecordMgr) Insert(record *login_db.Account) bool {
	this.locker.Lock()
	defer this.locker.Unlock()
	if _, o := this.accounts[record.Get_AccountId()]; o {
		return false
	}
	this.accounts[record.Get_AccountId()] = record
	return true
}
