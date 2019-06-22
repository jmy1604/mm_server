package main

import (
	"sync"

	"mm_server/libs/log"
	"mm_server/src/login_server/login_db"
)

type AccountRecordMgr struct {
	accounts map[string]*login_db.Accounts
	locker   sync.RWMutex
}

var account_record_mgr *login_db.AccountsRecordMgr = login_db.NewAccountsRecordMgr(1000)

func select_account_record_func(key string) (*login_db.Accounts, error) {
	return account_table.SelectByPrimaryField(key)
}

func select_account_records_func() (map[string]*login_db.Accounts, error) {
	return account_table.SelectAllMapRecords()
}

func (this *AccountRecordMgr) Init() bool {
	accounts, err := account_table.SelectAllMapRecords()
	if err != nil {
		log.Error("account new manager init err: %v", err.Error())
		return false
	}
	this.accounts = accounts
	log.Trace("load all account records: %v", accounts)
	return true
}

func (this *AccountRecordMgr) Has(account string) bool {
	this.locker.RLock()
	defer this.locker.RUnlock()
	return this.accounts[account] != nil
}

func (this *AccountRecordMgr) Get(account string) *login_db.Accounts {
	this.locker.RLock()
	defer this.locker.RUnlock()
	return this.accounts[account]
}

func (this *AccountRecordMgr) Insert(record *login_db.Accounts) bool {
	this.locker.Lock()
	defer this.locker.Unlock()
	if _, o := this.accounts[record.Get_AccountId()]; o {
		return false
	}
	this.accounts[record.Get_AccountId()] = record
	return true
}

func (this *AccountRecordMgr) New(account string) *login_db.Accounts {
	a := &login_db.Accounts{
		AccountId: account,
	}
	if !this.Insert(a) {
		return nil
	}
	return a
}

func (this *AccountRecordMgr) Remove(account string) bool {
	this.locker.Lock()
	defer this.locker.Unlock()
	if this.accounts[account] == nil {
		return false
	}
	delete(this.accounts, account)
	return true
}
