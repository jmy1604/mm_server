package main

import (
	"sync"
)

// CatInfoPool ...
type CatInfoPool struct {
	pool *sync.Pool
}

// Init ...
func (pool *CatInfoPool) Init() {
	this.pool = &sync.Pool{
		New: func() interface{} {
			return &msg_client_message.CatInfo{}
		},
	}
}

// Get ...
func (pool *CatInfoPool) Get() *msg_client_message.CatInfo {
	return pool.pool.Get().(*msg_client_message.CatInfo)
}

// Put ...
func (pool *CatInfoPool) Put(ds *msg_client_message.CatInfo) {
	pool.pool.Put(ds)
}
