package main

import (
	"errors"
	"fmt"
	"mm_server/libs/log"
	"mm_server/libs/rpc"
	"mm_server/src/common"
	"mm_server/src/rpc_proto"
	"strconv"
	"time"
)

// 游戏服到游戏服的调用
type G2G_CallProc struct {
}

func (this *G2G_CallProc) Do(args *rpc_proto.G2R_Transfer, reply *rpc_proto.G2R_TransferResult) error {
	defer func() {
		if err := recover(); err != nil {
			log.Stack(err)
		}
	}()
	rpc_client := GetRpcClientByPlayerId(args.ReceivePlayerId)
	if rpc_client == nil {
		err_str := fmt.Sprintf("!!!!!! Not found rpc client by player id %v", args.ReceivePlayerId)
		return errors.New(err_str)
	}

	log.Debug("G2G_CallProc @@@@@@@ call method[%v] args[%v] reply[%v]", args.Method, args.Args, reply.Result)

	var result interface{}
	err := rpc_client.Call(args.Method, args.Args, result)
	if err != nil {
		return err
	}
	log.Debug("G2G_CallProc @@@@@@@ call method[%v] result[%v]", args.Method, result)
	reply.Result = result
	return nil
}

// ping 大厅
type G2R_PingProc struct {
}

func (this *G2R_PingProc) Do(args *rpc_proto.G2R_Ping, result *rpc_proto.G2R_Pong) error {
	// 不做任何处理
	return nil
}

/* 监听RPC调用 */
type G2R_ListenRPCProc struct {
}

func (this *G2R_ListenRPCProc) Do(args *rpc_proto.G2R_ListenIPNoitfy, result *rpc_proto.G2R_ListenIPResult) error {
	defer func() {
		if err := recover(); err != nil {
			log.Stack(err)
		}
	}()

	log.Info("get notify listen rpc ip: %v", args.ListenIP)
	// 再连接到GameServer

	if !server.connect_game(args.ListenIP, args.ServerId) {
		err_str := fmt.Sprintf("不能连接到大厅[IP:%v, Id:%v]", args.ListenIP, args.ServerId)
		return errors.New(err_str)
	}

	time.Sleep(time.Second * 1)
	return nil
}

// 全局调用
type G2R_GlobalProc struct {
}

func (this *G2R_GlobalProc) ChargeSave(args *rpc_proto.G2R_ChargeSave, result *rpc_proto.G2R_ChargeSaveResult) error {
	defer func() {
		if err := recover(); err != nil {
			log.Error(err)
		}
	}()

	if args.Channel == 1 {
		row := dbc.GooglePays.GetRow(args.OrderId)
		if row == nil {
			row = dbc.GooglePays.AddRow(args.OrderId)
			row.SetBundleId(args.BundleId)
			row.SetAccount(args.Account)
			row.SetPlayerId(args.PlayerId)
			row.SetPayTime(args.PayTime)
			row.SetPayTimeStr(args.PayTimeStr)
		}
	} else if args.Channel == 2 {
		row := dbc.ApplePays.GetRow(args.OrderId)
		if row == nil {
			row = dbc.ApplePays.AddRow(args.OrderId)
			row.SetBundleId(args.BundleId)
			row.SetAccount(args.Account)
			row.SetPlayerId(args.PlayerId)
			row.SetPayTime(args.PayTime)
			row.SetPayTimeStr(args.PayTimeStr)
		}
	} else {
		err_str := fmt.Sprintf("@@@ G2R_GlobalProc::ChargeSave Player[%v,%v], Unknown Channel %v", args.Account, args.PlayerId, args.Channel)
		return errors.New(err_str)
	}

	log.Trace("@@@ Charge Save %v", args)

	return nil
}

// 玩家调用
type G2R_PlayerProc struct {
}

// 基本信息更新
func (this *G2R_PlayerProc) BaseInfoUpdate(args *rpc_proto.G2R_PlayerBaseInfoUpdate, result *rpc_proto.G2R_PlayerBaseInfoUpdateResult) error {
	defer func() {
		if err := recover(); err != nil {
			log.Error(err)
		}
	}()

	row := dbc.PlayerBaseInfos.GetRow(args.Info.Id)
	if row == nil {
		row = dbc.PlayerBaseInfos.AddRow(args.Info.Id)
	}

	if args.Info.Name != "" {
		row.SetName(args.Info.Name)
	}

	if args.Info.Level > 0 {
		row.SetLevel(args.Info.Level)
	}

	if args.Info.Head > 0 {
		row.SetHead(args.Info.Head)
	}

	return nil
}

// 获取好友关卡积分
func (this *G2R_PlayerProc) GetFriendStageScore(args *rpc_proto.G2R_GetFriendStageScore, result *rpc_proto.G2R_GetFriendStageScoreResult) error {
	defer func() {
		if err := recover(); err != nil {
			log.Error(err)
		}
	}()

	for _, id := range args.FriendIds {
		row := dbc.PlayerStageTotalScores.GetRow(id)
		if row == nil {
			continue
		}
		if !row.Stages.HasIndex(args.StageId) {
			continue
		}
		o, name, level, head := get_player_base_info(id)
		if !o {
			continue
		}
		stage_score, _ := row.Stages.GetTopScore(args.StageId)
		result.FriendsScoreData = append(result.FriendsScoreData, &rpc_proto.FriendStageScoreData{
			Id:         args.PlayerId,
			StageScore: stage_score,
			Name:       name,
			Level:      level,
			Head:       head,
		})
	}

	return nil
}

// 排行榜调用
type G2R_RankListProc struct {
}

func _player_stage_total_score_ranklist_update(args *rpc_proto.G2R_RankListDataUpdate, now_time int32) {
	rank_list_mgr.UpdateItem(common.RANK_LIST_TYPE_STAGE_TOTAL_SCORE, &common.PlayerInt32RankItem{
		Value:      args.RankParam[0],
		UpdateTime: now_time,
		PlayerId:   args.PlayerId,
	})
	row := dbc.PlayerStageTotalScores.GetRow(args.PlayerId)
	if row == nil {
		row = dbc.PlayerStageTotalScores.AddRow(args.PlayerId)
	}
	row.SetScore(args.RankParam[0])
	row.SetUpdateTime(now_time)
	curr_rank := rank_list_mgr.GetRankByKey(common.RANK_LIST_TYPE_STAGE_TOTAL_SCORE, args.PlayerId)
	if curr_rank < row.HistoryTopData.GetRank() {
		row.HistoryTopData.SetRank(curr_rank)
		row.HistoryTopData.SetScore(args.RankParam[0])
	}
	if len(args.RankParam) >= 3 {
		stage_id := args.RankParam[1]
		stage_score := args.RankParam[2]
		if !row.Stages.HasIndex(stage_id) {
			row.Stages.Add(&dbPlayerStageTotalScoreStageData{
				Id:       stage_id,
				TopScore: stage_score,
			})
		} else {
			row.Stages.SetTopScore(stage_id, stage_score)
		}
	}
}

func _player_charm_ranklist_update(args *rpc_proto.G2R_RankListDataUpdate, now_time int32) {
	rank_list_mgr.UpdateItem(common.RANK_LIST_TYPE_CHARM, &common.PlayerInt32RankItem{
		Value:      args.RankParam[0],
		UpdateTime: now_time,
		PlayerId:   args.PlayerId,
	})
	row := dbc.PlayerCharms.GetRow(args.PlayerId)
	if row == nil {
		row = dbc.PlayerCharms.AddRow(args.PlayerId)
	}
	row.SetCharmValue(args.RankParam[0])
	row.SetUpdateTime(now_time)
	curr_rank := rank_list_mgr.GetRankByKey(common.RANK_LIST_TYPE_CHARM, args.PlayerId)
	if curr_rank < row.HistoryTopData.GetRank() {
		row.HistoryTopData.SetRank(curr_rank)
		row.HistoryTopData.SetCharm(args.RankParam[0])
	}
}

func _player_cat_ouqi_ranklist_update(args *rpc_proto.G2R_RankListDataUpdate, now_time int32) {
	cat_id := args.RankParam[0]
	ouqi := args.RankParam[1]
	row := dbc.PlayerCatOuqis.GetRow(args.PlayerId)
	var item = common.PlayerCatOuqiRankItem{
		PlayerId: args.PlayerId,
		CatId:    cat_id,
	}
	if ouqi > 0 {
		rank_list_mgr.UpdateItem(common.RANK_LIST_TYPE_CAT_OUQI, &common.PlayerCatOuqiRankItem{
			PlayerId:   args.PlayerId,
			CatId:      cat_id,
			Ouqi:       ouqi,
			UpdateTime: now_time,
		})
		if row == nil {
			row = dbc.PlayerCatOuqis.AddRow(args.PlayerId)
		}
		if !row.Cats.HasIndex(cat_id) {
			row.Cats.Add(&dbPlayerCatOuqiCatData{
				CatId:      cat_id,
				Ouqi:       ouqi,
				UpdateTime: now_time,
			})
		} else {
			row.Cats.SetOuqi(cat_id, ouqi)
			row.Cats.SetUpdateTime(cat_id, now_time)
		}
		curr_rank := rank_list_mgr.GetRankByKey(common.RANK_LIST_TYPE_CAT_OUQI, item.GetKey())
		top_rank, _ := row.Cats.GetHistoryTopRank(cat_id)
		if curr_rank < top_rank {
			row.Cats.SetHistoryTopRank(cat_id, curr_rank)
		}
	} else {
		rank_list_mgr.DeleteItem(common.RANK_LIST_TYPE_CAT_OUQI, item.GetKey())
		if row != nil {
			row.Cats.Remove(cat_id)
		}
	}
}

func _player_be_zaned_ranklist_update(args *rpc_proto.G2R_RankListDataUpdate, now_time int32) {
	to_player_id := args.RankParam[0]
	row := dbc.PlayerBeZaneds.GetRow(to_player_id)
	if row == nil {
		row = dbc.PlayerBeZaneds.AddRow(to_player_id)
	}
	zaned := row.Zan()
	rank_list_mgr.UpdateItem(common.RANK_LIST_TYPE_BE_ZANED, &common.PlayerInt32RankItem{
		Value:      zaned,
		UpdateTime: now_time,
		PlayerId:   to_player_id,
	})
	curr_rank := rank_list_mgr.GetRankByKey(common.RANK_LIST_TYPE_BE_ZANED, to_player_id)
	if curr_rank < row.HistoryTopData.GetRank() {
		row.HistoryTopData.SetRank(curr_rank)
		row.HistoryTopData.SetZaned(zaned)
	}
}

// 更新排行榜
func (this *G2R_RankListProc) UpdateData(args *rpc_proto.G2R_RankListDataUpdate, result *rpc_proto.G2R_RankListDataUpdateResult) error {
	defer func() {
		if err := recover(); err != nil {
			log.Error(err)
		}
	}()

	now_time := int32(time.Now().Unix())
	if args.RankType == common.RANK_LIST_TYPE_STAGE_TOTAL_SCORE {
		_player_stage_total_score_ranklist_update(args, now_time)
	} else if args.RankType == common.RANK_LIST_TYPE_CHARM {
		_player_charm_ranklist_update(args, now_time)
	} else if args.RankType == common.RANK_LIST_TYPE_CAT_OUQI {
		_player_cat_ouqi_ranklist_update(args, now_time)
	} else if args.RankType == common.RANK_LIST_TYPE_BE_ZANED {
		_player_be_zaned_ranklist_update(args, now_time)
	} else {
		log.Warn("Unknown rank type %v from player %v", args.RankType, args.PlayerId)
	}

	return nil
}

// 获得数据
func (this *G2R_RankListProc) GetRankItems(args *rpc_proto.G2R_RankListGetData, result *rpc_proto.G2R_RankListGetDataResult) error {
	defer func() {
		if err := recover(); err != nil {
			log.Error(err)
		}
	}()

	var key interface{}
	if args.RankType == common.RANK_LIST_TYPE_CAT_OUQI {
		var item = common.PlayerCatOuqiRankItem{
			PlayerId: args.PlayerId,
			CatId:    args.RankParam,
		}
		key = item.GetKey()
	} else {
		key = args.PlayerId
	}
	rank_items, self_rank, self_value := rank_list_mgr.GetItemsByRange(args.RankType, key, args.StartRank, args.RankNum)
	result.RankType = args.RankType
	result.PlayerId = args.PlayerId
	result.StartRank = args.StartRank
	result.RankNum = args.RankNum
	result.RankItems = rank_items
	result.SelfRank = self_rank
	result.SelfValue = self_value

	if args.RankType == common.RANK_LIST_TYPE_STAGE_TOTAL_SCORE {
		row := dbc.PlayerStageTotalScores.GetRow(args.PlayerId)
		if row != nil {
			result.SelfHistoryTopRank = row.HistoryTopData.GetRank()
		}
	} else if args.RankType == common.RANK_LIST_TYPE_CHARM {
		row := dbc.PlayerCharms.GetRow(args.PlayerId)
		if row != nil {
			result.SelfHistoryTopRank = row.HistoryTopData.GetRank()
		}
	} else if args.RankType == common.RANK_LIST_TYPE_CAT_OUQI {
		row := dbc.PlayerCatOuqis.GetRow(args.PlayerId)
		if row != nil {
			result.SelfHistoryTopRank, _ = row.Cats.GetHistoryTopRank(args.RankParam)
		}
	} else if args.RankType == common.RANK_LIST_TYPE_BE_ZANED {
		row := dbc.PlayerBeZaneds.GetRow(args.PlayerId)
		if row != nil {
			result.SelfHistoryTopRank = row.HistoryTopData.GetRank()
		}
	}

	if rank_items != nil {
		for _, r := range rank_items {
			var player_id int32
			if args.RankType != common.RANK_LIST_TYPE_CAT_OUQI {
				rr := r.(*common.PlayerInt32RankItem)
				if rr == nil {
					continue
				}
				player_id = rr.PlayerId
			} else {
				rr := r.(*common.PlayerCatOuqiRankItem)
				if rr == nil {
					continue
				}
				player_id = rr.PlayerId
			}

			if result.PlayerBaseInfos == nil {
				result.PlayerBaseInfos = make(map[int32]*rpc_proto.PlayerBaseInfo)
			}

			_, name, level, head := get_player_base_info(player_id)
			result.PlayerBaseInfos[player_id] = &rpc_proto.PlayerBaseInfo{
				Id:    player_id,
				Name:  name,
				Level: level,
				Head:  head,
			}

		}
	}
	log.Trace("@@@ Rank List Get Items %v", result)

	return nil
}

type G2R_FriendProc struct {
}

func (this *G2R_FriendProc) SearchFriends(args *rpc_proto.G2R_SearchFriend, result *rpc_proto.G2R_SearchFriendResult) error {
	id, err := strconv.Atoi(args.Key)
	if err == nil {
		row := dbc.PlayerBaseInfos.GetRow(int32(id))
		result.Players = append(result.Players, &rpc_proto.SearchPlayerInfo{
			Id:    int32(id),
			Nick:  row.GetName(),
			Level: row.GetLevel(),
			Head:  row.GetHead(),
		})
	}

	id = int(player_mgr.GetId(args.Key))
	if id > 0 {
		row := dbc.PlayerBaseInfos.GetRow(int32(id))
		result.Players = append(result.Players, &rpc_proto.SearchPlayerInfo{
			Id:    int32(id),
			Nick:  row.GetName(),
			Level: row.GetLevel(),
			Head:  row.GetHead(),
		})
	}

	log.Trace("@@@ Searched Friends %v by key %v", result.Players, args.Key)
	return nil
}

// 初始化
func (this *RpcServer) init_proc_service() bool {
	this.rpc_service = &rpc.Service{}

	if !this.rpc_service.Register(&G2G_CallProc{}) {
		return false
	}

	if !this.rpc_service.Register(&G2R_ListenRPCProc{}) {
		return false
	}

	if !this.rpc_service.Register(&G2R_GlobalProc{}) {
		return false
	}

	if !this.rpc_service.Register(&G2G_CommonProc{}) {
		return false
	}

	if !this.rpc_service.Register(&G2R_PlayerProc{}) {
		return false
	}

	if !this.rpc_service.Register(&G2R_RankListProc{}) {
		return false
	}

	if !this.rpc_service.Register(&G2R_FriendProc{}) {
		return false
	}

	// 注册用户自定义RPC数据类型
	rpc_proto.RegisterRpcUserType()

	if this.rpc_service.Listen(config.ListenIP) != nil {
		return false
	}
	return true
}

// 反初始化
func (this *RpcServer) uninit_proc_service() {
	if this.rpc_service != nil {
		this.rpc_service.Close()
		this.rpc_service = nil
	}
}
