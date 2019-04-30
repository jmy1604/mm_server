package main

import (
	"mm_server/libs/log"
	"mm_server/libs/rpc"
	"mm_server/src/rpc_proto"
)

// ping RPC服务
type R2G_PingProc struct{}

func (this *R2G_PingProc) Do(args *rpc_proto.R2G_Ping, reply *rpc_proto.R2G_Pong) error {
	// 不做任何处理
	log.Info("收到rpc服务的ping请求")
	return nil
}

// 全局调用
type G2G_GlobalProc struct {
}

func (this *G2G_GlobalProc) WorldChat(args *rpc_proto.G2G_WorldChat, result *rpc_proto.G2G_WorldChatResult) error {
	defer func() {
		if err := recover(); err != nil {
			log.Stack(err)
		}
	}()

	log.Debug("@@@ G2G_GlobalProc::WorldChat Player[%v] world chat content[%v]", args.FromPlayerId, args.ChatContent)
	return nil
}

// 玩家调用
type R2G_PlayerProc struct {
}

func (this *R2G_PlayerProc) GetPlayerBaseInfo(args *rpc_proto.R2G_GetPlayerBaseInfo, result *rpc_proto.R2G_GetPlayerBaseInfoResult) error {
	defer func() {
		if err := recover(); err != nil {
			log.Stack(err)
		}
	}()

	p := player_mgr.GetPlayerById(args.PlayerId)

	if result.BaseInfo == nil {
		result.BaseInfo = &rpc_proto.PlayerBaseInfo{}
	}
	result.BaseInfo.Id = args.PlayerId
	result.BaseInfo.Name = p.db.GetName()
	result.BaseInfo.Level = p.db.GetLevel()
	result.BaseInfo.Head = p.db.Info.GetHead()

	return nil
}

// 初始化rpc服务
func (this *GameServer) init_rpc_service() bool {
	if this.rpc_service != nil {
		return true
	}
	this.rpc_service = &rpc.Service{}

	// 注册RPC服务
	if !this.rpc_service.Register(&R2G_PingProc{}) {
		return false
	}
	if !this.rpc_service.Register(&G2G_GlobalProc{}) {
		return false
	}

	if !this.rpc_service.Register(&G2G_CommonProc{}) {
		return false
	}

	if !this.rpc_service.Register(&R2G_PlayerProc{}) {
		return false
	}

	if this.rpc_service.Listen(config.ListenRpcServerIP) != nil {
		log.Error("监听rpc服务端口[%v]失败", config.ListenRpcServerIP)
		return false
	}
	log.Info("监听rpc服务端口[%v]成功", config.ListenRpcServerIP)
	go this.rpc_service.Serve()
	return true
}

// 反初始化rpc服务
func (this *GameServer) uninit_rpc_service() {
	if this.rpc_service != nil {
		this.rpc_service.Close()
		this.rpc_service = nil
	}
}
