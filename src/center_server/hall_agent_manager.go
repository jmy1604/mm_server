package main

import (
	"mm_server/libs/log"
	"mm_server/libs/server_conn"
	"mm_server/libs/timer"
	"mm_server/proto/gen_go/server_message"
	"time"

	"github.com/golang/protobuf/proto"
)

type GameAgent struct {
	conn             *server_conn.ServerConn
	id               int32
	name             string
	listen_room_ip   string
	listen_client_ip string
}

func new_match_agent(conn *server_conn.ServerConn, id int32, name, listen_room_ip, listen_client_ip string) *GameAgent {
	if nil == conn || id < 0 {
		log.Error("NewGameAgent param error !", id)
		return nil
	}

	retagent := &GameAgent{}
	retagent.conn = conn
	retagent.id = id
	retagent.name = name
	retagent.listen_room_ip = listen_room_ip
	retagent.listen_client_ip = listen_client_ip
	return retagent
}

func (this *GameAgent) Send(msg_id uint16, msg proto.Message) {
	this.conn.Send(msg_id, msg, true)
}

var game_agent_mgr GameAgentManager

type GameAgentManager struct {
	start_time      time.Time
	server_node     *server_conn.Node
	id2agent        map[int32]*GameAgent
	conn2agent      map[*server_conn.ServerConn]*GameAgent
	conn2agent_lock *RWMutex
	initialized     bool
}

func (this *GameAgentManager) Init() (ok bool) {
	this.start_time = time.Now()
	this.id2agent = make(map[int32]*GameAgent)
	this.conn2agent = make(map[*server_conn.ServerConn]*GameAgent)
	this.conn2agent_lock = NewRWMutex()
	this.server_node = server_conn.NewNode(this, 0, 0, 5000, 0, 0, 0, 0, 0)
	this.server_node.SetDesc("HallAgent", "大厅服务器")

	this.RegisterMsgHandler()
	this.initialized = true
	ok = true
	return
}

func (this *GameAgentManager) Start(ip string, max_conn int32) (err error) {
	err = this.server_node.Listen(ip, max_conn)
	if err != nil {
		log.Error("启动服务(%v)失败 %v", ip, err)
		return
	}
	return
}

func (this *GameAgentManager) OnTick() {
}

func (this *GameAgentManager) OnAccept(conn *server_conn.ServerConn) {
	log.Info("新的Hall连接[%v]", conn.GetAddr())
}

func (this *GameAgentManager) OnConnect(conn *server_conn.ServerConn) {

}

func (this *GameAgentManager) OnUpdate(conn *server_conn.ServerConn, t timer.TickTime) {

}

func (this *GameAgentManager) OnDisconnect(conn *server_conn.ServerConn, reason server_conn.E_DISCONNECT_REASON) {
	log.Info("断开Hall连接[%v]", conn.GetAddr())
	this.RemoveAgent(conn)
}

func (this *GameAgentManager) CloseConnection(conn *server_conn.ServerConn, reason server_conn.E_DISCONNECT_REASON) {
	if nil == conn {
		log.Error("参数为空")
		return
	}

	conn.Close(reason)
}

func (this *GameAgentManager) SendToAllMatch(msg_id uint16, msg proto.Message) {
	this.server_node.Broadcast(msg_id, msg)
}

func (this *GameAgentManager) HasAgentByConn(conn *server_conn.ServerConn) bool {
	if nil == conn {
		return false
	}
	this.conn2agent_lock.UnSafeRLock("GameAgentManager HasAgentByConn")
	defer this.conn2agent_lock.UnSafeRUnlock()
	if nil != this.conn2agent[conn] {
		return true
	}

	return false
}

func (this *GameAgentManager) AddAgent(conn *server_conn.ServerConn, id int32, name, listen_room_ip, listen_client_ip string) *GameAgent {
	new_agent := new_match_agent(conn, id, name, listen_room_ip, listen_client_ip)
	if nil == new_agent {
		log.Error("GameAgentManager AddAgent new_agent nil ", conn, id, name, listen_room_ip, listen_client_ip)
		return nil
	}

	this.conn2agent_lock.UnSafeLock("GameAgentManager AddAgent")
	defer this.conn2agent_lock.UnSafeUnlock()
	this.conn2agent[conn] = new_agent
	conn.T = id
	this.id2agent[id] = new_agent
	return new_agent
}

func (this *GameAgentManager) GetAgentById(id int32) *GameAgent {
	this.conn2agent_lock.UnSafeRLock("GameAgentManager GetAgentById")
	defer this.conn2agent_lock.UnSafeRUnlock()

	return this.id2agent[id]
}

func (this *GameAgentManager) RemoveAgent(conn *server_conn.ServerConn) {
	this.conn2agent_lock.UnSafeLock("GameAgent RemoveAgent")
	defer this.conn2agent_lock.UnSafeUnlock()
	cur_agent := this.conn2agent[conn]
	if nil != cur_agent {
		if nil != this.id2agent[cur_agent.id] {
			delete(this.id2agent, cur_agent.id)
		}
		delete(this.conn2agent, conn)
	}
	return
}

func (this *GameAgentManager) Broadcast(msg_id uint16, msg proto.Message) {
	this.server_node.Broadcast(msg_id, msg)
}

func (this *GameAgentManager) RandOneAgent() *GameAgent {
	this.conn2agent_lock.UnSafeLock("GameAgent RemoveAgent")
	defer this.conn2agent_lock.UnSafeUnlock()
	for _, game := range this.id2agent {
		return game
	}

	return nil
}

//==========================================================================================================

func (this *GameAgentManager) RegisterMsgHandler() {
	this.server_node.SetPid2P(game_agent_msgid2msg)
	this.SetMessageHandler(uint16(msg_server_message.MSGID_G2C_GAME_SERVER_REGISTER), G2CGameServerRegisterHandler)
}

func (this *GameAgentManager) SetMessageHandler(type_id uint16, h server_conn.Handler) {
	this.server_node.SetHandler(type_id, h)
}

func game_agent_msgid2msg(msg_id uint16) proto.Message {
	if msg_id == uint16(msg_server_message.MSGID_G2C_GAME_SERVER_REGISTER) {
		return &msg_server_message.G2CGameServerRegister{}
	} else {
		log.Error("Cant found proto message by msg_id[%v]", msg_id)
	}
	return nil
}

func G2CGameServerRegisterHandler(conn *server_conn.ServerConn, m proto.Message) {
	req := m.(*msg_server_message.G2CGameServerRegister)
	if nil == conn || nil == req {
		log.Error("G2CGameServerRegisterHandler param error !")
		return
	}

	cur_agent := game_agent_mgr.GetAgentById(req.GetServerId())
	if nil != cur_agent {
		conn.Close(server_conn.E_DISCONNECT_REASON_FORCE_CLOSED)
		log.Error("G2MGameServerRegisterHandler Server Id [%v] Already Registered, Check server config file !!!!!!!!!!!!!!! ", req.GetServerId())
		return
	}

	new_agent := game_agent_mgr.AddAgent(conn, req.GetServerId(), req.GetServerName(), req.GetListenRoomIP(), req.GetListenClientIP())
	log.Info("M2C New GameServer(Id:%d Name:%s) Register", req.GetServerId(), req.GetServerName())

	if nil == new_agent {
		log.Error("G2CGameServerRegisterHandler agent nil ")
		return
	}

	res := &msg_server_message.C2GLoginServerList{}
	res.ServerList = login_info_mgr.GetInfoList()
	if len(res.ServerList) > 0 {
		conn.Send(uint16(msg_server_message.MSGID_C2G_LOGIN_SERVER_LIST), res, true)
	}

	return
}
