package main

import (
	"mm_server/libs/log"
	"mm_server/libs/server_conn"
	"mm_server/libs/timer"
	"mm_server/proto/gen_go/server_message"
	"sync/atomic"
	"time"

	"github.com/golang/protobuf/proto"
)

const (
	LOGIN_CONN_STATE_DISCONNECT  = 0
	LOGIN_CONN_STATE_CONNECTED   = 1
	LOGIN_CONN_STATE_FORCE_CLOSE = 2
)

type LoginConnection struct {
	serverid        int32
	servername      string
	listen_match_ip string
	client_node     *server_conn.Node
	state           int32

	last_conn_time int32
}

func new_login_conn(serverid int32, servername, ip string) *LoginConnection {
	if "" == ip {
		log.Error("new_login_conn param error !")
		return nil
	}

	ret_login_conn := &LoginConnection{
		serverid:        serverid,
		servername:      servername,
		listen_match_ip: ip}

	ret_login_conn.Init()
	go ret_login_conn.Start()

	return ret_login_conn
}

func (this *LoginConnection) Init() {
	this.client_node = server_conn.NewNode(this, 0, 0, 100, 0, 0, 0, 0, 0)
	this.client_node.SetDesc("登录服务器", "")
	this.state = LOGIN_CONN_STATE_DISCONNECT
	this.RegisterMsgHandler()
}

func (this *LoginConnection) Start() {
	if this.Connect(LOGIN_CONN_STATE_DISCONNECT) {
		log.Event("连接Loginerver成功", nil, log.Property{"IP", this.listen_match_ip})
	}
	for {
		state := atomic.LoadInt32(&this.state)
		if state == LOGIN_CONN_STATE_CONNECTED {
			time.Sleep(time.Second * 2)
			continue
		}

		if state == LOGIN_CONN_STATE_FORCE_CLOSE {
			this.client_node.ClientDisconnect()
			log.Event("与login的连接被强制关闭", nil)
			break
		}
		if this.Connect(state) {
			log.Event("连接loginserver成功", nil, log.Property{"IP", this.listen_match_ip})
		}
	}
}

func (this *LoginConnection) Connect(state int32) (ok bool) {
	if LOGIN_CONN_STATE_DISCONNECT == state {
		var err error
		for {
			log.Trace("连接loginServer %v", this.listen_match_ip)
			err = this.client_node.ClientConnect(this.listen_match_ip, time.Second*10)
			if nil == err {
				break
			}

			// 每隔30秒输出一次连接信息
			now := time.Now().Unix()
			if int32(now)-this.last_conn_time >= 30 {
				log.Trace("LoginServer连接中...")
				this.last_conn_time = int32(now)
			}
			time.Sleep(time.Second * 5)
		}
	}

	if atomic.CompareAndSwapInt32(&this.state, state, LOGIN_CONN_STATE_CONNECTED) {
		go this.client_node.ClientRun()
		ok = true
	}
	return
}

func (this *LoginConnection) OnAccept(c *server_conn.ServerConn) {
	log.Error("Impossible accept")
}

func (this *LoginConnection) OnConnect(c *server_conn.ServerConn) {
	log.Trace("Server[%v][%v] on LoginServer connect", config.ServerId, config.ServerName)
	c.T = this.serverid
	notify := &msg_server_message.G2LGameServerRegister{}
	notify.ServerId = config.ServerId
	notify.ServerName = config.ServerName
	notify.ListenClientIP = config.ListenClientOutIP
	c.Send(uint16(msg_server_message.MSGID_G2L_GAME_SERVER_REGISTER), notify, true)
}

func (this *LoginConnection) OnUpdate(c *server_conn.ServerConn, t timer.TickTime) {

}

func (this *LoginConnection) OnDisconnect(c *server_conn.ServerConn, reason server_conn.E_DISCONNECT_REASON) {
	/*
		if reason == server_conn.E_DISCONNECT_REASON_FORCE_CLOSED {
			this.state = LOGIN_CONN_STATE_FORCE_CLOSE
		} else {
			this.state = LOGIN_CONN_STATE_DISCONNECT
		}
	*/
	this.state = LOGIN_CONN_STATE_FORCE_CLOSE
	log.Event("与LoginServer连接断开", nil)
	if c.T > 0 {
		login_conn_mgr.RemoveLogin(c.T)
	}
}

func (this *LoginConnection) ForceClose(bimmidate bool) {
	this.state = LOGIN_CONN_STATE_FORCE_CLOSE
	if bimmidate {
		this.client_node.ClientDisconnect()
	}
}

func (this *LoginConnection) Send(msg_id uint16, msg proto.Message) {
	if LOGIN_CONN_STATE_CONNECTED != this.state {
		log.Info("与登录服务器未连接，不能发送消息!!!")
		return
	}
	if nil == this.client_node {
		return
	}
	this.client_node.GetClient().Send(msg_id, msg, false)
}

//=============================================================================

func (this *LoginConnection) RegisterMsgHandler() {
	this.client_node.SetPid2P(login_conn_msgid2msg)
	this.SetMessageHandler(uint16(msg_server_message.MSGID_L2G_SYNC_ACCOUNT_TOKEN), L2GSyncAccountTokenHandler)
	this.SetMessageHandler(uint16(msg_server_message.MSGID_L2G_DISCONNECT_NOTIFY), L2GDissconnectNotifyHandler)
	this.SetMessageHandler(uint16(msg_server_message.MSGID_L2G_BIND_NEW_ACCOUNT_REQUEST), L2GBindNewAccountHandler)
}

func (this *LoginConnection) SetMessageHandler(type_id uint16, h server_conn.Handler) {
	this.client_node.SetHandler(type_id, h)
}

func login_conn_msgid2msg(msg_id uint16) proto.Message {
	if msg_id == uint16(msg_server_message.MSGID_L2G_SYNC_ACCOUNT_TOKEN) {
		return &msg_server_message.L2GSyncAccountToken{}
	} else if msg_id == uint16(msg_server_message.MSGID_L2G_DISCONNECT_NOTIFY) {
		return &msg_server_message.L2GDissconnectNotify{}
	} else if msg_id == uint16(msg_server_message.MSGID_L2G_BIND_NEW_ACCOUNT_REQUEST) {
		return &msg_server_message.L2GBindNewAccountRequest{}
	} else {
		log.Error("Cant found proto message by msg_id[%v]", msg_id)
	}
	return nil
}

func L2GSyncAccountTokenHandler(conn *server_conn.ServerConn, msg proto.Message) {
	req := msg.(*msg_server_message.L2GSyncAccountToken)
	if nil == req {
		log.Error("ID_L2GSyncAccountTokenHandler param error !")
		return
	}

	login_token_mgr.AddToUid2Token(req.GetUniqueId(), req.GetAccount(), req.GetToken(), int32(req.GetPlayerId()), conn)
	log.Info("ID_L2GSyncAccountTokenHandler UniqueId[%v] Account[%v] Token[%v] PlayerId[%v]", req.GetUniqueId(), req.GetAccount(), req.GetToken(), req.GetPlayerId())
}

func L2GDissconnectNotifyHandler(conn *server_conn.ServerConn, msg proto.Message) {

	log.Info("L2GDissconnectNotifyHandler param error !")

	return
}

func L2GBindNewAccountHandler(conn *server_conn.ServerConn, msg proto.Message) {
	req := msg.(*msg_server_message.L2GBindNewAccountRequest)
	if req == nil {
		log.Error("L2GBindNewAccountHandler msg param invalid")
		return
	}

	p := player_mgr.GetPlayerByUid(req.GetUniqueId())
	if p == nil {
		log.Error("Cant found account %v to bind new account %v", req.GetAccount(), req.GetNewAccount())
		return
	}

	row := dbc.Players.GetRow(p.Id)
	if row == nil {
		log.Error("Cant found db row with player account[%v] and id[%v]", req.GetAccount(), p.Id)
		return
	}

	p.Account = req.GetNewAccount() // 新账号
	row.SetAccount(req.GetNewAccount())

	login_token_mgr.BindNewAccount(req.GetUniqueId(), req.GetAccount(), req.GetNewAccount())

	log.Debug("Account %v bind new account %v", req.GetAccount(), req.GetNewAccount())
}
