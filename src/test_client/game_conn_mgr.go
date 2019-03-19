package main

import (
	"mm_server/libs/log"
	"sync"
)

type GameConnMgr struct {
	acc2gconn      map[string]*GameConnection
	acc_arr        []*GameConnection
	acc2gconn_lock *sync.RWMutex
}

var game_conn_mgr GameConnMgr

func (this *GameConnMgr) Init() bool {
	this.acc2gconn = make(map[string]*GameConnection)
	this.acc2gconn_lock = &sync.RWMutex{}
	return true
}

func (this *GameConnMgr) AddGameConn(conn *GameConnection) {
	if nil == conn {
		log.Error("GameConnMgr AddGameConn param error !")
		return
	}

	this.acc2gconn_lock.Lock()
	defer this.acc2gconn_lock.Unlock()

	this.acc2gconn[conn.acc] = conn
	this.acc_arr = append(this.acc_arr, conn)
	log.Debug("add new game connection %v", conn.acc)
}

func (this *GameConnMgr) RemoveGameConnByAcc(acc string) {
	this.acc2gconn_lock.Lock()
	defer this.acc2gconn_lock.Unlock()

	conn := this.acc2gconn[acc]
	if conn == nil {
		return
	}
	delete(this.acc2gconn, acc)
	if this.acc_arr != nil {
		for i := 0; i < len(this.acc_arr); i++ {
			if this.acc_arr[i] == conn {
				this.acc_arr[i] = nil
				break
			}
		}
	}
}

func (this *GameConnMgr) GetGameConnByAcc(acc string) *GameConnection {
	this.acc2gconn_lock.RLock()
	defer this.acc2gconn_lock.RUnlock()

	return this.acc2gconn[acc]
}
