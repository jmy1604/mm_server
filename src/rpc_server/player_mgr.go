package main

import (
	"sync"
)

type PlayerMgr struct {
	name2id map[string]int32
	locker  sync.RWMutex
}

var player_mgr PlayerMgr

func (this *PlayerMgr) Add(name string, id int32) bool {
	this.locker.Lock()
	defer this.locker.Unlock()

	if this.name2id == nil {
		this.name2id = make(map[string]int32)
	}

	if this.has(name) {
		return false
	}

	this.name2id[name] = id

	return true
}

func (this *PlayerMgr) has(name string) bool {
	if this.name2id == nil {
		return false
	}

	_, o := this.name2id[name]
	if o {
		return true
	}

	return true
}

func (this *PlayerMgr) Has(name string) bool {
	this.locker.RLock()
	defer this.locker.RUnlock()
	return this.has(name)
}

func (this *PlayerMgr) AddIgnore(name string, id int32) {
	this.locker.Lock()
	defer this.locker.Unlock()

	if this.name2id == nil {
		this.name2id = make(map[string]int32)
	}

	this.name2id[name] = id
}

func (this *PlayerMgr) Remove(name string) bool {
	this.locker.Lock()
	defer this.locker.Unlock()

	if this.name2id == nil {
		return false
	}

	_, o := this.name2id[name]
	if !o {
		return false
	}

	delete(this.name2id, name)
	return true
}

func (this *PlayerMgr) GetId(name string) int32 {
	this.locker.RLock()
	defer this.locker.RUnlock()

	if this.name2id == nil {
		return 0
	}

	id, o := this.name2id[name]
	if !o {
		return 0
	}

	return id
}
