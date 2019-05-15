package main

import (
	"errors"
	"mm_server/libs/log"
	"mm_server/libs/rpc"
	"mm_server/src/rpc_proto"
)

func get_rpc_client() *rpc.Client {
	if game_server.rpc_client == nil {
		log.Error("!!!!!!!!!! RPC Client is nil")
		return nil
	}
	return game_server.rpc_client
}

func (this *GameServer) init_rpc_client() bool {
	// 注册用户自定义RPC数据类型
	rpc_proto.RegisterRpcUserType()

	this.rpc_client = rpc.NewClient()
	var on_connect rpc.OnConnectFunc = func(args interface{}) {
		rpc_client := args.(*rpc.Client)
		proc_string := "G2R_ListenRPCProc.Do"
		var arg = rpc_proto.G2R_ListenIPNoitfy{config.ListenRpcServerIP, config.ServerId}
		var result = rpc_proto.G2R_ListenIPResult{}
		err := rpc_client.Call(proc_string, arg, &result)
		if err != nil {
			log.Error("RPC调用[%v]失败, err:%v", proc_string, err.Error())
			return
		}
		log.Info("RPC调用[%v]成功", proc_string)
	}
	this.rpc_client.SetOnConnect(on_connect)

	if !this.rpc_client.Dial(config.RpcServerIP) {
		log.Error("连接rpc服务器[%v]失败", config.RpcServerIP)
		return false
	}
	log.Info("连接rpc服务器[%v]成功!!!", config.RpcServerIP)

	this.rpc_client.Run()

	return true
}

func (this *GameServer) uninit_rpc_client() {
	if this.rpc_client != nil {
		this.rpc_client.Close()
		this.rpc_client = nil
	}
}

// 游戏服到游戏服调用
func (this *GameServer) rpc_game2game(receive_player_id int32, method string, args interface{}, reply interface{}) error {
	if this.rpc_client == nil {
		err := errors.New("!!!! rpc client is null")
		return err
	}
	transfer_args := &rpc_proto.G2R_Transfer{}
	transfer_args.Method = method
	transfer_args.Args = args
	transfer_args.ReceivePlayerId = receive_player_id
	transfer_reply := &rpc_proto.G2R_TransferResult{}
	transfer_reply.Result = reply

	log.Debug("@@@@@ #####  transfer_args[%v]  transfer_reply[%v]", transfer_args.Args, transfer_reply.Result)

	err := this.rpc_client.Call("G2G_CallProc.Do", transfer_args, transfer_reply)
	if err != nil {
		log.Error("RPC @@@ G2G_CallProc.Do(%v,%v) error(%v)", transfer_args, transfer_reply, err.Error())
	}
	return err
}

// 充值记录
func (p *Player) rpc_charge_save(channel int32, order_id, bundle_id, account string, player_id, pay_time int32, pay_time_str string) (result *rpc_proto.G2R_ChargeSaveResult) {
	rpc_client := get_rpc_client()
	if rpc_client == nil {
		return nil
	}

	var args = rpc_proto.G2R_ChargeSave{
		Channel:    channel,
		OrderId:    order_id,
		BundleId:   bundle_id,
		Account:    account,
		PlayerId:   player_id,
		PayTime:    pay_time,
		PayTimeStr: pay_time_str,
	}

	result = &rpc_proto.G2R_ChargeSaveResult{}
	err := rpc_client.Call("G2R_GlobalProc.ChargeSave", &args, result)
	if err != nil {
		log.Error("RPC ### Player[%v] charge save err[%v]", p.Id, err.Error())
	}
	return
}

// 更新玩家基本信息
func (this *Player) rpc_player_base_info_update() bool {
	rpc_client := get_rpc_client()
	if rpc_client == nil {
		return false
	}

	var args = rpc_proto.G2R_PlayerBaseInfoUpdate{
		Info: &rpc_proto.PlayerBaseInfo{
			Id:    this.Id,
			Name:  this.db.GetName(),
			Level: this.db.GetLevel(),
			Head:  this.db.Info.GetHead(),
		},
	}

	err := rpc_client.Call("G2R_PlayerProc.BaseInfoUpdate", &args, &rpc_proto.G2R_PlayerBaseInfoUpdateResult{})
	if err != nil {
		log.Error("RPC ### Player[%v] update base info err %v", this.Id, err.Error())
		return false
	}

	return true
}

// 排行榜数据更新
func (this *Player) rpc_rank_list_update_data(rank_type int32, rank_params []int32) (result *rpc_proto.G2R_RankListDataUpdateResult) {
	rpc_client := get_rpc_client()
	if rpc_client == nil {
		return nil
	}

	var args = rpc_proto.G2R_RankListDataUpdate{
		RankType:  rank_type,
		PlayerId:  this.Id,
		RankParam: rank_params,
	}

	result = &rpc_proto.G2R_RankListDataUpdateResult{}
	err := rpc_client.Call("G2R_RankListProc.UpdateData", &args, result)
	if err != nil {
		log.Error("RPC ### Player[%v] update rank type %v data by params %v, err %v", this.Id, args.RankType, args.RankParam, err.Error())
	}

	return
}

// 排行榜获取数据
func (this *Player) rpc_rank_list_get_data(rank_type, start_rank, rank_num int32, rank_param int32) (result *rpc_proto.G2R_RankListGetDataResult) {
	rpc_client := get_rpc_client()
	if rpc_client == nil {
		return nil
	}

	var args = rpc_proto.G2R_RankListGetData{
		RankType:  rank_type,
		PlayerId:  this.Id,
		StartRank: start_rank,
		RankNum:   rank_num,
		RankParam: rank_param,
	}

	result = &rpc_proto.G2R_RankListGetDataResult{}
	err := rpc_client.Call("G2R_RankListProc.GetRankItems", &args, result)
	if err != nil {
		log.Error("RPC ### Player[%v] get rank type %v items by start_rank(%v) rank_num(%v), err %v", this.Id, rank_type, start_rank, rank_num)
	}

	return
}

// 获取好友关卡积分
func (this *Player) rpc_get_friends_stage_score(stage_id int32) (result *rpc_proto.G2R_GetFriendStageScoreResult) {
	rpc_client := get_rpc_client()
	if rpc_client == nil {
		return nil
	}

	friend_ids := this.db.Friends.GetAllIndex()
	var args = rpc_proto.G2R_GetFriendStageScore{
		StageId:   stage_id,
		FriendIds: friend_ids,
	}

	result = &rpc_proto.G2R_GetFriendStageScoreResult{}
	err := rpc_client.Call("G2R_PlayerProc.GetFriendStageScore", &args, result)
	if err != nil {
		log.Error("RPC ### Player[%v] get friends %v stage %v score err %v", this.Id, friend_ids, stage_id, err.Error())
	}

	return
}

// 查找好友
func (this *Player) rpc_search_friends(key string) (result *rpc_proto.G2R_SearchFriendResult) {
	rpc_client := get_rpc_client()
	if rpc_client == nil {
		return nil
	}

	var args = rpc_proto.G2R_SearchFriend{
		Key: key,
	}

	result = &rpc_proto.G2R_SearchFriendResult{}
	err := rpc_client.Call("G2R_FriendProc.SearchFriends", &args, result)
	if err != nil {
		log.Error("RPC ### Player[%v] search friends err %v", this.Id, err.Error())
	}

	return
}

// 获取多个玩家基础信息
func (this *Player) rpc_get_players_base_info(player_ids []int32) (result *rpc_proto.G2R_GetPlayersBaseInfoResult) {
	rpc_client := get_rpc_client()
	if rpc_client == nil {
		return nil
	}

	var args = rpc_proto.G2R_GetPlayersBaseInfo{
		PlayerIds: player_ids,
	}

	result = &rpc_proto.G2R_GetPlayersBaseInfoResult{}
	err := rpc_client.Call("G2R_PlayerProc.GetPlayersBaseInfo", &args, result)
	if err != nil {
		log.Error("RPC ### Player %v get players %v base info err %v", this.Id, player_ids, err.Error())
	}

	return
}
