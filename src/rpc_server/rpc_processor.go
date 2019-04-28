package main

import (
	"errors"
	"fmt"
	"mm_server/libs/log"
	"mm_server/libs/rpc"
	"mm_server/src/common"
	"mm_server/src/rpc_proto"
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

// 排行榜调用
type G2R_RankListProc struct {
}

func (this *G2R_RankListProc) UpdateData(args *rpc_proto.G2R_RankListDataUpdate, result *rpc_proto.G2R_RankListDataUpdateResult) error {
	defer func() {
		if err := recover(); err != nil {
			log.Error(err)
		}
	}()

	now_time := int32(time.Now().Unix())
	if args.RankType == common.RANK_LIST_TYPE_STAGE_TOTAL_SCORE {
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
	} else if args.RankType == common.RANK_LIST_TYPE_CHARM {
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
	} else if args.RankType == common.RANK_LIST_TYPE_CAT_OUQI {
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
	} else if args.RankType == common.RANK_LIST_TYPE_BE_ZANED {
		rank_list_mgr.UpdateItem(common.RANK_LIST_TYPE_BE_ZANED, &common.PlayerInt32RankItem{
			Value:      args.RankParam[0],
			UpdateTime: now_time,
			PlayerId:   args.PlayerId,
		})
		row := dbc.PlayerBeZaneds.GetRow(args.PlayerId)
		if row == nil {
			row = dbc.PlayerBeZaneds.AddRow(args.PlayerId)
		}
		row.SetZaned(args.RankParam[0])
		row.SetUpdateTime(now_time)
		curr_rank := rank_list_mgr.GetRankByKey(common.RANK_LIST_TYPE_BE_ZANED, args.PlayerId)
		if curr_rank < row.HistoryTopData.GetRank() {
			row.HistoryTopData.SetRank(curr_rank)
			row.HistoryTopData.SetZaned(args.RankParam[0])
		}
	} else {
		log.Warn("Unknown rank type %v from player %v", args.RankType, args.PlayerId)
	}

	return nil
}

func (this *G2R_RankListProc) GetRankItems(args *rpc_proto.G2R_RankListGetData, result *rpc_proto.G2R_RankListGetDataResult) error {
	defer func() {
		if err := recover(); err != nil {
			log.Error(err)
		}
	}()

	rank_items, self_rank, self_value := rank_list_mgr.GetItemsByRange(args.RankType, args.PlayerId, args.StartRank, args.RankNum)
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

	if !this.rpc_service.Register(&G2R_RankListProc{}) {
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
