package main

import (
	"mm_server/proto/gen_go/client_message"
	"sync"
)

// DelaySkillPool
type CatInfoPool struct {
	pool *sync.Pool
}

func (this *CatInfoPool) Init() {
	this.pool = &sync.Pool{
		New: func() interface{} {
			return &msg_client_message.CatInfo{}
		},
	}
}

func (this *CatInfoPool) Get() *msg_client_message.CatInfo {
	return this.pool.Get().(*msg_client_message.CatInfo)
}

func (this *CatInfoPool) Put(ds *msg_client_message.CatInfo) {
	this.pool.Put(ds)
}
