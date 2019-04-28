package main

import (
	"errors"
	"mm_server/libs/log"
	"mm_server/libs/rpc"
	"mm_server/libs/timer"
	"mm_server/src/rpc_proto"
	"mm_server/src/tables"
	"sync"
	"time"
)

type RpcServer struct {
	quit                  bool
	shutdown_lock         *sync.Mutex
	shutdown_completed    bool
	ticker                *timer.TickTimer
	initialized           bool
	rpc_service           *rpc.Service             // rpc服务
	game_rpc_clients      map[int32]*GameRpcClient // 连接GameServer的rpc客户端(key: GameServerId, value: *rpc.Client)
	game_rpc_clients_lock *sync.Mutex
}

var server RpcServer
var shop_mgr tables.ShopTableManager
var global_config tables.GlobalConfig

func (this *RpcServer) Init() (err error) {
	if this.initialized {
		return
	}

	this.shutdown_lock = &sync.Mutex{}

	if !this.OnInit() {
		return errors.New("RpcServer OnInit Failed !")
	}
	this.initialized = true

	return
}

func (this *RpcServer) load_config() bool {
	/*if !shop_mgr.Init() {
		return false
	}*/
	if !global_config.Init("global.json") {
		return false
	}
	return true
}

func (this *RpcServer) OnInit() bool {
	if !this.load_config() {
		log.Error("load config failed")
		return false
	}

	if !this.init_proc_service() {
		log.Error("init rpc service failed")
		return false
	}
	if !this.init_game_clients() {
		log.Error("init rpc clients failed")
		return false
	}
	if !global_data.Init() {
		log.Error("init global data failed")
		return false
	}

	return true
}

func (this *RpcServer) Start() {
	if !this.initialized {
		return
	}

	this.Run()
}

func (this *RpcServer) Run() {
	defer func() {
		if err := recover(); err != nil {
			log.Stack(err)
		}

		this.shutdown_completed = true
	}()

	this.ticker = timer.NewTickTimer(1000)
	this.ticker.Start()
	defer this.ticker.Stop()

	// rpc服务
	go this.rpc_service.Serve()
	// redis
	go global_data.RunRedis()

	for {
		select {
		case _, ok := <-this.ticker.Chan:
			{
				if !ok {
					return
				}

				begin := time.Now()
				time_cost := time.Now().Sub(begin).Seconds()
				if time_cost > 1 {
					log.Trace("耗时 %v", time_cost)
					if time_cost > 30 {
						log.Error("耗时 %v", time_cost)
					}
				}
			}
		}
	}
}

func (this *RpcServer) Shutdown() {
	if !this.initialized {
		return
	}

	this.shutdown_lock.Lock()
	defer this.shutdown_lock.Unlock()

	this.uninit_proc_service()
	this.uninit_game_clients()
	global_data.Close()

	if this.quit {
		return
	}
	this.quit = true

	log.Trace("关闭游戏主循环")

	begin := time.Now()

	if this.ticker != nil {
		this.ticker.Stop()
	}

	for {
		if this.shutdown_completed {
			break
		}

		time.Sleep(time.Millisecond * 100)
	}

	log.Trace("关闭游戏主循环耗时 %v 秒", time.Now().Sub(begin).Seconds())
}

func (this *RpcServer) init_game_clients() bool {
	if this.game_rpc_clients == nil {
		this.game_rpc_clients = make(map[int32]*GameRpcClient)
	}
	if this.game_rpc_clients_lock == nil {
		this.game_rpc_clients_lock = &sync.Mutex{}
	}
	return true
}

func (this *RpcServer) uninit_game_clients() {
	if this.game_rpc_clients != nil {
		for _, c := range this.game_rpc_clients {
			if c.rpc_client != nil {
				c.rpc_client.Close()
				c.rpc_client = nil
			}
		}
	}
	if this.game_rpc_clients_lock != nil {
		this.game_rpc_clients_lock = nil
	}
}

func (this *RpcServer) connect_game(addr string, server_id int32) bool {
	this.game_rpc_clients_lock.Lock()
	defer this.game_rpc_clients_lock.Unlock()

	for _, c := range this.game_rpc_clients {
		if c.server_ip == addr {
			c.rpc_client.Close()
			log.Info("断掉旧的GameServer[%v]的连接", c.server_ip)
			break
		}
	}

	rc := &rpc.Client{}
	if !rc.Dial(addr) {
		log.Error("到GameServer[%v]的rpc连接失败", addr)
		return false
	}

	gr := &GameRpcClient{}
	gr.server_ip = addr
	gr.rpc_client = rc
	gr.server_id = server_id

	this.game_rpc_clients[server_id] = gr

	log.Info("到GameServer[%v]的rpc连接成功", addr)
	return true
}

func (this *RpcServer) check_connect() {
	var args = rpc_proto.G2R_Ping{}
	var result = rpc_proto.G2R_Pong{}

	to_del_ids := make(map[int32]int32)
	for id, c := range this.game_rpc_clients {
		if c == nil {
			to_del_ids[id] = id
		} else if c.rpc_client == nil {
			to_del_ids[id] = id
		} else {
			if c.rpc_client.Call("G2R_PingProc.Do", args, &result) != nil {
				to_del_ids[id] = id
			}
		}
	}

	for id, _ := range to_del_ids {
		delete(this.game_rpc_clients, id)
	}
}
