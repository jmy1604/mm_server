package main

import (
	"mm_server/libs/log"
	"mm_server/libs/rpc"
	"mm_server/libs/socket"
	"mm_server/libs/timer"
	"mm_server/libs/utils"
	"mm_server/src/server_config"
	"sync"
	"time"
)

type GameServer struct {
	start_time         time.Time
	net                *socket.Node
	quit               bool
	shutdown_lock      *sync.Mutex
	shutdown_completed bool
	ticker             *timer.TickTimer
	initialized        bool
	last_gc_time       int32
	rpc_client         *rpc.Client  // 连接到rpc服务
	rpc_service        *rpc.Service // 接受rpc连接
	redis_conn         *utils.RedisConn
}

var game_server GameServer

func (this *GameServer) Init() (ok bool) {
	this.start_time = time.Now()
	this.shutdown_lock = &sync.Mutex{}
	this.net = socket.NewNode(&game_server, time.Duration(config.RecvMaxMSec), time.Duration(config.SendMaxMSec), 5000, nil) //(this, 0, 0, 5000, 0, 0, 0, 0, 0)

	this.redis_conn = &utils.RedisConn{}
	if !this.redis_conn.Connect(config.RedisServerIP) {
		return
	}

	login_token_mgr.LoadRedisData()

	// rpc初始化
	if !this.init_rpc_service() {
		return
	}
	if !this.init_rpc_client() {
		return
	}

	err := this.OnInit()
	if err != nil {
		log.Error("服务器初始化失败[%s]", err.Error())
		return
	}

	// 世界频道
	world_chat_mgr.Init(CHAT_CHANNEL_WORLD)
	// 招募频道
	//recruit_chat_mgr.Init(CHAT_CHANNEL_RECRUIT)
	// 系统频道
	system_chat_mgr.Init(CHAT_CHANNEL_SYSTEM)
	// 公告跑马灯
	anouncement_mgr.Init()

	this.initialized = true

	ok = true
	return
}

func (this *GameServer) OnInit() (err error) {
	player_mgr.RegMsgHandler()

	if USE_CONN_TIMER_WHEEL == 0 {
		conn_timer_mgr.Init()
	} else {
		conn_timer_wheel.Init()
	}

	return
}

func (this *GameServer) Start(use_https bool) (err error) {
	log.Event("服务器已启动", nil, log.Property{"IP", config.ListenClientInIP})
	log.Trace("**************************************************")

	go this.Run()

	if use_https {
		crt_path := server_config.GetConfPathFile("server.crt")
		key_path := server_config.GetConfPathFile("server.key")
		msg_handler_mgr.StartHttps(crt_path, key_path)
	} else {
		msg_handler_mgr.StartHttp()
	}

	return
}

func (this *GameServer) Run() {
	defer func() {
		if err := recover(); err != nil {
			log.Stack(err)
		}

		this.shutdown_completed = true
	}()

	this.ticker = timer.NewTickTimer(1000)
	this.ticker.Start()
	defer this.ticker.Stop()

	go this.redis_conn.Run(1000)
	if USE_CONN_TIMER_WHEEL == 0 {
		go conn_timer_mgr.Run()
	} else {
		go conn_timer_wheel.Run()
	}

	//go friend_recommend_mgr.Run()

	//go charge_month_card_manager.Run()

	//go activity_mgr.Run()

	for {
		select {
		case d, ok := <-this.ticker.Chan:
			{
				if !ok {
					return
				}

				begin := time.Now()
				this.OnTick(d)
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

func (this *GameServer) Shutdown() {
	if !this.initialized {
		return
	}

	this.shutdown_lock.Lock()
	defer this.shutdown_lock.Unlock()

	if this.quit {
		return
	}
	this.quit = true

	this.redis_conn.Close()
	//arena_season_mgr.ToEnd()

	log.Trace("关闭游戏主循环")

	begin := time.Now()

	if this.ticker != nil {
		this.ticker.Stop()
		for {
			if this.shutdown_completed {
				break
			}

			time.Sleep(time.Millisecond * 100)
		}
	}

	log.Trace("等待 shutdown_completed 完毕")
	center_conn.client_node.Shutdown()
	this.net.Shutdown()
	if nil != msg_handler_mgr.msg_http_listener {
		msg_handler_mgr.msg_http_listener.Close()
	}

	this.uninit_rpc_service()
	this.uninit_rpc_client()

	log.Trace("关闭游戏主循环耗时 %v 秒", time.Now().Sub(begin).Seconds())

	dbc.Save(false)
	dbc.Shutdown()
}

func (this *GameServer) OnTick(t timer.TickTime) {
	player_mgr.OnTick()
}

func (this *GameServer) OnAccept(c *socket.TcpConn) {
	log.Info("HallServer OnAccept [%s]", c.GetAddr())
}

func (this *GameServer) OnConnect(c *socket.TcpConn) {

}

func (this *GameServer) OnDisconnect(c *socket.TcpConn, reason socket.E_DISCONNECT_REASON) {
	if c.T > 0 {
		cur_p := player_mgr.GetPlayerById(int32(c.T))
		if nil != cur_p {
			player_mgr.PlayerLogout(cur_p)
		}
	}
	log.Trace("玩家[%d] 断开连接[%v]", c.T, c.GetAddr())
}

func (this *GameServer) CloseConnection(c *socket.TcpConn, reason socket.E_DISCONNECT_REASON) {
	if c == nil {
		log.Error("参数为空")
		return
	}

	c.Close(reason)
}

func (this *GameServer) OnUpdate(c *socket.TcpConn, t timer.TickTime) {

}
