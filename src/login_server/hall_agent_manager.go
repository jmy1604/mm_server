package main

import (
	"mm_server/libs/log"
	"mm_server/libs/server_conn"
	"mm_server/libs/timer"
	"mm_server/proto/gen_go/server_message"
	"sync"
	"time"

	"github.com/golang/protobuf/proto"
)

const (
	AGENT_ACCOUNT_STATE_DISCONNECTED = iota
	AGENT_ACCOUNT_STATE_CONNECTED    = 1
	AGENT_ACCOUNT_STATE_IN_LOGIN     = 2
	AGENT_ACCOUNT_STATE_IN_GAME      = 3
)

const (
	GAME_AGENT_DISCONNECT = iota
	GAME_AGENT_CONNECTED  = 1
	GAME_AGENT_CREATED    = 2
)

type GameAgent struct {
	conn             *server_conn.ServerConn // 连接
	state            int32                   // agent状态
	name             string                  // game_server name
	id               int32                   // game_server ID
	max_player_num   int32                   // 最大在线人数
	curr_player_num  int32                   // 当前在线人数
	aids             map[string]int32        // 已有的账号
	aids_lock        *sync.RWMutex
	listen_client_ip string // 监听客户端IP
}

func new_agent(c *server_conn.ServerConn, state int32) (agent *GameAgent) {
	agent = &GameAgent{}
	agent.conn = c
	agent.state = state
	agent.aids = make(map[string]int32)
	agent.aids_lock = &sync.RWMutex{}
	return
}

func (this *GameAgent) HasAid(aid string) (ok bool) {
	this.aids_lock.RLock()
	defer this.aids_lock.RUnlock()

	state, o := this.aids[aid]
	if !o {
		return
	}
	if state <= 0 {
		return
	}
	ok = true
	return
}

func (this *GameAgent) AddAid(aid string) (ok bool) {
	this.aids_lock.Lock()
	defer this.aids_lock.Unlock()

	_, o := this.aids[aid]
	if o {
		return
	}
	this.aids[aid] = 1
	ok = true
	return
}

func (this *GameAgent) RemoveAid(aid string) (ok bool) {
	this.aids_lock.Lock()
	defer this.aids_lock.Unlock()

	_, o := this.aids[aid]
	if !o {
		return
	}

	delete(this.aids, aid)
	ok = true
	return
}

func (this *GameAgent) UpdatePlayersNum(max_num, curr_num int32) {
	this.aids_lock.Lock()
	defer this.aids_lock.Unlock()

	this.max_player_num = max_num
	this.curr_player_num = curr_num
	return
}

func (this *GameAgent) GetPlayersNum() (max_num, curr_num int32) {
	this.aids_lock.RLock()
	defer this.aids_lock.RUnlock()

	max_num = this.max_player_num
	curr_num = this.curr_player_num
	return
}

func (this *GameAgent) Send(msg_id uint16, msg proto.Message) {
	this.conn.Send(msg_id, msg, true)
}

func (this *GameAgent) Close(force bool) {
	this.aids_lock.Lock()
	defer this.aids_lock.Unlock()
	if force {
		this.conn.Close(server_conn.E_DISCONNECT_REASON_FORCE_CLOSED)
	} else {
		this.conn.Close(server_conn.E_DISCONNECT_REASON_LOGGIN_FAILED)
	}
}

//========================================================================

type GameAgentManager struct {
	net                *server_conn.Node
	id2agents          map[int32]*GameAgent
	conn2agents        map[*server_conn.ServerConn]*GameAgent
	agents_lock        *sync.RWMutex
	inited             bool
	quit               bool
	shutdown_lock      *sync.Mutex
	shutdown_completed bool
	ticker             *timer.TickTimer
	listen_err_chan    chan error
}

var game_agent_manager GameAgentManager

func (this *GameAgentManager) Init() (ok bool) {
	this.id2agents = make(map[int32]*GameAgent)
	this.conn2agents = make(map[*server_conn.ServerConn]*GameAgent)
	this.agents_lock = &sync.RWMutex{}
	this.net = server_conn.NewNode(this, 0, 0, 5000,
		0,
		0,
		2048,
		2048,
		2048)
	this.net.SetDesc("HallAgent", "大厅服务器")

	this.shutdown_lock = &sync.Mutex{}
	this.listen_err_chan = make(chan error)
	this.init_message_handle()
	this.inited = true
	ok = true
	return
}

func (this *GameAgentManager) wait_listen_res() (err error) {
	timeout := make(chan bool, 1)
	go func() {
		time.Sleep(5 * time.Second)
		timeout <- true
	}()

	var o bool
	select {
	case err, o = <-this.listen_err_chan:
		{
			if !o {
				log.Trace("wait listen_err_chan failed")
				return
			}
		}
	case <-timeout:
		{
		}
	}

	return
}

func (this *GameAgentManager) Start() (err error) {
	log.Event("GameAgentManager已启动", nil, log.Property{"IP", config.ListenGameIP})
	log.Trace("**************************************************")

	go this.Run()

	go this.Listen()

	err = this.wait_listen_res()

	return
}

func (this *GameAgentManager) Listen() {
	err := this.net.Listen(config.ListenGameIP, config.MaxGameConnections)
	if err != nil {
		this.listen_err_chan <- err
		log.Error("启动GameAgentManager失败 %v", err)
	} else {
		close(this.listen_err_chan)
	}
	return
}

func (this *GameAgentManager) Run() {
	defer func() {
		if err := recover(); err != nil {
			log.Stack(err)
		}
		this.shutdown_completed = true
	}()

	this.ticker = timer.NewTickTimer(1000)
	this.ticker.Start()
	defer this.ticker.Stop()

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

func (this *GameAgentManager) OnAccept(c *server_conn.ServerConn) {
	this.AddAgent(c, GAME_AGENT_CONNECTED)
	log.Trace("新的HallAgent连接")
}

func (this *GameAgentManager) OnConnect(c *server_conn.ServerConn) {
}

func (this *GameAgentManager) OnUpdate(c *server_conn.ServerConn, t timer.TickTime) {
}

func (this *GameAgentManager) OnDisconnect(c *server_conn.ServerConn, reason server_conn.E_DISCONNECT_REASON) {
	this.DisconnectAgent(c, reason)
	log.Trace("断开GameAgent连接")
}

func (this *GameAgentManager) OnTick(t timer.TickTime) {
}

func (this *GameAgentManager) set_ih(type_id uint16, h server_conn.Handler) {
	this.net.SetHandler(type_id, h)
}

func (this *GameAgentManager) HasAgent(server_id int32) (ok bool) {
	this.agents_lock.RLock()
	defer this.agents_lock.RUnlock()
	_, o := this.id2agents[server_id]
	if !o {
		return
	}
	ok = true
	return
}

func (this *GameAgentManager) GetAgent(c *server_conn.ServerConn) (agent *GameAgent) {
	this.agents_lock.RLock()
	defer this.agents_lock.RUnlock()
	a, o := this.conn2agents[c]
	if !o {
		return
	}
	agent = a
	return
}

func (this *GameAgentManager) GetAgentByID(hall_id int32) (agent *GameAgent) {
	this.agents_lock.RLock()
	defer this.agents_lock.RUnlock()
	a, o := this.id2agents[hall_id]
	if !o {
		return
	}
	agent = a
	return
}

func (this *GameAgentManager) AddAgent(c *server_conn.ServerConn, state int32) (agent *GameAgent) {
	this.agents_lock.Lock()
	defer this.agents_lock.Unlock()

	_, o := this.conn2agents[c]
	if o {
		return
	}

	agent = new_agent(c, state)
	this.conn2agents[c] = agent
	return
}

func (this *GameAgentManager) SetAgentByID(id int32, agent *GameAgent) (ok bool) {
	this.agents_lock.Lock()
	defer this.agents_lock.Unlock()

	agent.id = id

	this.id2agents[id] = agent
	ok = true
	return
}

func (this *GameAgentManager) RemoveAgent(c *server_conn.ServerConn, lock bool) (ok bool) {
	if lock {
		this.agents_lock.Lock()
		defer this.agents_lock.Unlock()
	}

	agent, o := this.conn2agents[c]
	if !o {
		return
	}

	delete(this.conn2agents, c)
	delete(this.id2agents, agent.id)

	agent.aids = nil

	ok = true
	return
}

func (this *GameAgentManager) DisconnectAgent(c *server_conn.ServerConn, reason server_conn.E_DISCONNECT_REASON) (ok bool) {
	if c == nil {
		return
	}

	ok = this.RemoveAgent(c, true)

	res := &msg_server_message.L2GDissconnectNotify{}
	res.Reason = int32(reason)
	c.Send(uint16(msg_server_message.MSGID_L2G_DISCONNECT_NOTIFY), res, true)
	return
}

func (this *GameAgentManager) SetMessageHandler(type_id uint16, h server_conn.Handler) {
	this.set_ih(type_id, h)
}

func (this *GameAgentManager) UpdatePlayersNum(server_id int32, max_num, curr_num int32) {
	this.agents_lock.RLock()
	defer this.agents_lock.RUnlock()

	agent := this.id2agents[server_id]
	if agent == nil {
		return
	}

	agent.UpdatePlayersNum(max_num, curr_num)
}

func (this *GameAgentManager) GetPlayersNum(server_id int32) (agent *GameAgent, max_num, curr_num int32) {
	this.agents_lock.RLock()
	defer this.agents_lock.RUnlock()

	agent = this.id2agents[server_id]
	if agent == nil {
		return
	}

	max_num, curr_num = agent.GetPlayersNum()
	return
}

func (this *GameAgentManager) GetSuitableHallAgent() *GameAgent {
	this.agents_lock.RLock()
	defer this.agents_lock.RUnlock()

	for _, agent := range this.id2agents {
		if nil != agent {
			return agent
		}
	}

	return nil
}

//====================================================================================================

func (this *GameAgentManager) init_message_handle() {
	this.net.SetPid2P(game_agent_msgid2msg)
	this.SetMessageHandler(uint16(msg_server_message.MSGID_G2L_GAME_SERVER_REGISTER), G2LGameServerRegisterHandler)
	this.SetMessageHandler(uint16(msg_server_message.MSGID_G2L_ACCOUNT_LOGOUT_NOTIFY), G2LAccountLogoutNotifyHandler)
	this.SetMessageHandler(uint16(msg_server_message.MSGID_G2L_ACCOUNT_BAN), G2LAccountBanHandler)
}

func game_agent_msgid2msg(msg_id uint16) proto.Message {
	if msg_id == uint16(msg_server_message.MSGID_G2L_GAME_SERVER_REGISTER) {
		return &msg_server_message.G2LGameServerRegister{}
	} else if msg_id == uint16(msg_server_message.MSGID_G2L_ACCOUNT_LOGOUT_NOTIFY) {
		return &msg_server_message.G2LAccountLogoutNotify{}
	} else if msg_id == uint16(msg_server_message.MSGID_G2L_ACCOUNT_BAN) {
		return &msg_server_message.G2LAccountBan{}
	} else {
		log.Error("Cant found proto message by msg_id[%v]", msg_id)
	}
	return nil
}

func G2LGameServerRegisterHandler(conn *server_conn.ServerConn, m proto.Message) {
	req := m.(*msg_server_message.G2LGameServerRegister)
	if nil == req {
		log.Error("G2LGameServerRegisterHandler param error !")
		return
	}

	server_id := req.GetServerId()
	server_name := req.GetServerName()

	a := game_agent_manager.GetAgent(conn)
	if a == nil {
		log.Error("Agent[%v] not found", conn)
		return
	}

	if a.id == server_id /*game_agent_manager.HasAgent(server_id)*/ {
		game_agent_manager.DisconnectAgent(a.conn, server_conn.E_DISCONNECT_REASON_FORCE_CLOSED)
		log.Error("大厅服务器[%v]已有，不能有重复的ID", server_id)
		return
	}

	a.id = server_id
	a.name = server_name
	a.state = GAME_AGENT_CONNECTED
	a.listen_client_ip = req.GetListenClientIP()

	game_agent_manager.SetAgentByID(server_id, a)

	log.Trace("大厅服务器[%d %s %s]已连接", server_id, server_name, a.listen_client_ip)
}

func G2LAccountLogoutNotifyHandler(conn *server_conn.ServerConn, m proto.Message) {
	req := m.(*msg_server_message.G2LAccountLogoutNotify)
	if req == nil {
		log.Error("G2LAccountLogoutNotifyHandler param invalid")
		return
	}

	account_logout(req.GetAccount())

	log.Trace("Account %v log out notify", req.GetAccount())
}

func G2LAccountBanHandler(conn *server_conn.ServerConn, m proto.Message) {
	req := m.(*msg_server_message.G2LAccountBan)
	if req == nil {
		log.Error("G2LAccountBanHandler msg invalid")
		return
	}

	uid := req.GetUniqueId()
	ban := req.GetBanOrFree()
	row := dbc.BanPlayers.GetRow(uid)
	if ban > 0 {
		if row == nil {
			row = dbc.BanPlayers.AddRow(uid)
		}
		row.SetAccount(req.GetAccount())
		row.SetPlayerId(req.GetPlayerId())
		now_time := time.Now()
		row.SetStartTime(int32(now_time.Unix()))
		row.SetStartTimeStr(now_time.Format("2006-01-02 15:04:05"))
	} else {
		if row != nil {
			row.SetStartTime(0)
			row.SetStartTimeStr("")
		}
	}

	log.Trace("Unique id %v ban %v", uid, ban)
}
