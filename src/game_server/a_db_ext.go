package main

import (
	"mm_server/libs/log"
	"mm_server/proto/gen_go/client_message"
	"mm_server/src/share_data"
	"mm_server/src/tables"
	"sync/atomic"
	"time"
)

func (this *DBC) on_preload() (err error) {
	var p *Player
	for _, db := range this.Players.m_rows {
		if nil == db {
			log.Error("DBC on_preload Players have nil db !")
			continue
		}

		p = new_player_with_db(db.m_PlayerId, db)
		if nil == p {
			continue
		}

		player_mgr.Add2IdMap(p)
		player_mgr.Add2UidMap(p.UniqueId, p)

		//friend_recommend_mgr.CheckAndAddPlayer(p.Id)

		if p.db.GetLevel() == 0 {
			p.db.SetLevel(p.db.Info.GetLvl())
		}
	}

	return
}

func (this *dbGlobalRow) GetNextPlayerId() int32 {
	curr_id := atomic.AddInt32(&this.m_CurrentPlayerId, 1)
	new_id := share_data.GeneratePlayerId(config.ServerId, curr_id) //((config.ServerId << 20) & 0x7ff00000) | curr_id
	this.m_lock.UnSafeLock("dbGlobalRow.GetNextPlayerId")
	this.m_CurrentPlayerId_changed = true
	this.m_lock.UnSafeUnlock()
	return new_id
}

func (this *dbGlobalRow) GetNextGuildId() int32 {
	curr_id := atomic.AddInt32(&this.m_CurrentGuildId, 1)
	new_id := share_data.GenerateGuildId(config.ServerId, curr_id) //((config.ServerId << 20) & 0x7ff00000) | curr_id
	this.m_lock.UnSafeLock("dbGlobalRow.GetNextGuildId")
	this.m_CurrentGuildId_changed = true
	this.m_lock.UnSafeUnlock()
	return new_id
}

func (this *dbPlayerInfoColumn) FillBaseInfo(bi *msg_client_message.S2CRetBaseInfo) {
	this.m_row.m_lock.UnSafeRLock("dbPlayerInfoColumn.FillBaseInfo")
	defer this.m_row.m_lock.UnSafeRUnlock()
	tmp_data := this.m_data
	bi.Coins = tmp_data.Gold
	bi.Diamonds = tmp_data.Diamond
	return
}

func (this *dbPlayerInfoColumn) SubCoin(v int32) int32 {
	this.m_row.m_lock.UnSafeLock("dbPlayerInfoColumn.SubCoin")
	defer this.m_row.m_lock.UnSafeUnlock()
	this.m_data.Gold = this.m_data.Gold - v
	if this.m_data.Gold < 0 {
		this.m_data.Gold = 0
	}

	this.m_changed = true
	return this.m_data.Gold
}

func (this *dbPlayerInfoColumn) SubDiamond(v int32) int32 {
	this.m_row.m_lock.UnSafeLock("dbPlayerInfoColumn.SubDiamond")
	defer this.m_row.m_lock.UnSafeUnlock()
	this.m_data.Diamond = this.m_data.Diamond - v
	if this.m_data.Diamond < 0 {
		this.m_data.Diamond = 0
	}
	this.m_changed = true
	return this.m_data.Diamond
}

func (this *dbPlayerSignInfoColumn) FillSyncMsg(msg *msg_client_message.S2CSyncSignInfo) {
	this.m_row.m_lock.UnSafeRLock("dbPlayerSignInfoColumn.FillSyncMsg")
	defer this.m_row.m_lock.UnSafeRUnlock()

	msg.CurSignSum = this.m_data.CurSignSum
	msg.CurSignDays = this.m_data.CurSignDays
	msg.CurGetSignSumRewards = this.m_data.RewardSignSum

	return
}

func (this *dbPlayerGuidesColumn) ForceAdd(guide_id int32) {
	this.m_row.m_lock.UnSafeLock("dbPlayerGuidesColumn.ForceAdd")
	defer this.m_row.m_lock.UnSafeUnlock()
	_, has := this.m_data[guide_id]
	if has {
		return
	}
	d := &dbPlayerGuidesData{}
	d.GuideId = guide_id
	d.SetUnix = int32(time.Now().Unix())
	this.m_data[guide_id] = d
	this.m_changed = true
	return
}

func (this *dbPlayerGuidesColumn) FillSyncMsg(msg *msg_client_message.S2CSyncGuideData) {
	if nil == msg {
		log.Error("dbPlayerGuidesColumn FillSyncMsg msg nil !")
		return
	}

	this.m_row.m_lock.UnSafeRLock("dbPlayerGuidesColumn.FillSyncMsg")
	defer this.m_row.m_lock.UnSafeRUnlock()

	msg.GuideIds = make([]int32, 0, len(this.m_data))
	for _, val := range this.m_data {
		msg.GuideIds = append(msg.GuideIds, val.GuideId)
	}

	return
}

func (this *dbPlayerFriendColumn) FillAllListMsg(msg *msg_client_message.S2CRetFriendListResult) {
	var tmp_info *msg_client_message.FriendInfo
	this.m_row.m_lock.UnSafeRLock("dbPlayerFriendColumn.FillAllListMsg")
	defer this.m_row.m_lock.UnSafeRUnlock()
	msg.FriendList = make([]*msg_client_message.FriendInfo, 0, len(this.m_data))
	for _, val := range this.m_data {
		if nil == val {
			continue
		}

		tmp_info = &msg_client_message.FriendInfo{}
		tmp_info.PlayerId = val.FriendId
		tmp_info.Name = val.FriendName
		tmp_info.Level = val.Level
		tmp_info.VipLevel = val.VipLevel
		tmp_info.LastLogin = val.LastLogin
		tmp_info.Head = val.Head
		tmp_info.IsOnline = true
		log.Info("附加值到好友列表 %v", tmp_info)
		msg.FriendList = append(msg.FriendList, tmp_info)
	}

	return
}

func (this *dbPlayerFriendColumn) GetAviFriendId() int32 {
	this.m_row.m_lock.UnSafeRLock("dbPlayerFriendColumn.GetAviFriendId")
	defer this.m_row.m_lock.UnSafeRUnlock()
	for i := int32(1); i <= global_config.MaxFriendNum; i++ {
		if nil == this.m_data[i] {
			return i
		}
	}
	return 0
}

func (this dbPlayerFriendColumn) TryAddFriend(new_friend *dbPlayerFriendData) {
	if nil == new_friend {
		log.Error("dbPlayerFriendColumn TryAddFriend ")
		return
	}

	this.m_row.m_lock.UnSafeLock("dbPlayerFriendColumn.TryAddFriend")
	defer this.m_row.m_lock.UnSafeUnlock()

	if nil == this.m_data[new_friend.FriendId] {
		this.m_data[new_friend.FriendId] = new_friend
		this.m_changed = true
	}

	return
}

func (this *dbPlayerFriendReqColumn) FillAllListMsg(msg *msg_client_message.S2CRetFriendListResult) {

	var tmp_info *msg_client_message.FriendReq
	this.m_row.m_lock.UnSafeRLock("dbPlayerFriendReqColumn.FillAllListMsg")
	defer this.m_row.m_lock.UnSafeRUnlock()

	msg.Reqs = make([]*msg_client_message.FriendReq, 0, len(this.m_data))
	for _, val := range this.m_data {
		if nil == val {
			continue
		}

		tmp_info = &msg_client_message.FriendReq{}
		tmp_info.PlayerId = val.PlayerId
		tmp_info.Name = val.PlayerName
		msg.Reqs = append(msg.Reqs, tmp_info)
	}

	return
}

func (this *dbPlayerFriendReqColumn) CheckAndAdd(player_id int32, player_name string) int32 {
	this.m_row.m_lock.UnSafeLock("dbPlayerFriendReqColumn.CheckAndAdd")
	defer this.m_row.m_lock.UnSafeUnlock()

	d := this.m_data[player_id]
	if d != nil {
		log.Warn("!!! Player[%v,%v] already in request list of player[%v]", player_id, player_name, this.m_row.GetPlayerId())
		return int32(msg_client_message.E_ERR_FRIEND_THE_PLAYER_REQUESTED)
	}

	d = &dbPlayerFriendReqData{}
	d.PlayerId = player_id
	d.PlayerName = player_name
	this.m_data[player_id] = d
	this.m_changed = true
	return 1
}

func (this *dbPlayerFriendReqColumn) AgreeFriend(friend_id int32) bool {
	this.m_row.m_lock.UnSafeLock("dbPlayerFriendReqColumn.AgreeFriend")
	defer this.m_row.m_lock.UnSafeUnlock()

	d := this.m_data[friend_id]
	if d != nil {

	}
	return true
}

func (this *dbPlayerFriendColumn) GetAllIds() (ret_ids []int32) {
	this.m_row.m_lock.UnSafeRLock("dbPlayerFriendColumn.GetAllIds")
	defer this.m_row.m_lock.UnSafeRUnlock()
	tmp_len := len(this.m_data)
	if tmp_len <= 0 {
		return nil
	}

	ret_ids = make([]int32, 0, len(this.m_data))
	for _, v := range this.m_data {
		ret_ids = append(ret_ids, v.FriendId)
	}
	return
}

func (this *dbPlayerFocusPlayerColumn) GetAllIds() (ret_ids []int32) {
	this.m_row.m_lock.UnSafeRLock("dbPlayerFocusPlayerColumn.GetAllIds")
	defer this.m_row.m_lock.UnSafeRUnlock()
	tmp_len := len(this.m_data)
	if tmp_len <= 0 {
		return nil
	}

	ret_ids = make([]int32, 0, len(this.m_data))
	for _, v := range this.m_data {
		ret_ids = append(ret_ids, v.FriendId)
	}

	return
}

func (this *dbPlayerBeFocusPlayerColumn) GetNum() int32 {
	this.m_row.m_lock.UnSafeRLock("dbPlayerBeFocusPlayerColumn.GetNum")
	defer this.m_row.m_lock.UnSafeRUnlock()
	return int32(len(this.m_data))
}

func (this *dbPlayerItemColumn) ChkAddItemByNum(cfgid, num int32) int32 {
	this.m_row.m_lock.UnSafeLock("dbPlayerItemColumn.Add")
	defer this.m_row.m_lock.UnSafeUnlock()

	item := item_table_mgr.Map[cfgid]
	if item == nil {
		log.Error("添加物品时找不到物品配置ID[%v]", cfgid)
		return 0
	}
	d, has := this.m_data[cfgid]
	if has {
		if item.ValidTime == 0 {
			d.ItemNum += num
			if d.ItemNum > item.MaxNumber {
				d.ItemNum = item.MaxNumber
			}
		} else {
			d.ItemNum = 1
			d.StartTimeUnix = int32(time.Now().Unix())
			d.RemainSeconds = item.ValidTime * 3600
		}
	} else {
		d = &dbPlayerItemData{}
		d.ItemCfgId = cfgid
		if item.ValidTime == 0 {
			if num > item.MaxNumber {
				num = item.MaxNumber
			}
			d.ItemNum = num
		} else {
			d.ItemNum = 1
			d.StartTimeUnix = int32(time.Now().Unix())
			d.RemainSeconds = item.ValidTime * 3600
		}
		this.m_data[cfgid] = d
	}
	this.m_changed = true

	return d.ItemNum
}

func (this *dbPlayerItemColumn) ChkRemoveItem(item_id, num int32) (bool, int32) {
	this.m_row.m_lock.UnSafeLock("dbPlayerItemColumn.Remove")
	defer this.m_row.m_lock.UnSafeUnlock()
	item := item_table_mgr.Map[item_id]
	if item == nil {
		log.Error("删除物品[%v]时找不到ID", item_id)
		return false, 0
	}
	d, has := this.m_data[item_id]
	if !has {
		return false, 0
	}
	var left int32
	if d.ItemNum > num {
		d.ItemNum -= num
		left = d.ItemNum
	} else {
		delete(this.m_data, item_id)
		left = 0
	}
	this.m_changed = true
	return true, left
}

func (this *dbPlayerStageColumn) ChkSetTopScore(id int32, v int32) int32 {
	this.m_row.m_lock.UnSafeLock("dbPlayerStageColumn.ChkSetTopScore")
	defer this.m_row.m_lock.UnSafeUnlock()
	d := this.m_data[id]
	if d == nil {
		log.Error("not exist %v %v", this.m_row.GetPlayerId(), id)
		return d.TopScore
	}
	if d.TopScore < v {
		d.TopScore = v
		this.m_changed = true
	}

	return d.TopScore
}

func (this *dbPlayerStageColumn) GetTotalTopStar() int32 {
	this.m_row.m_lock.UnSafeRLock("dbPlayerStageColumn.GetTotalTopStar")
	defer this.m_row.m_lock.UnSafeRUnlock()

	total_top := int32(0)
	for _, d := range this.m_data {
		if nil == d {
			continue
		}

		total_top += d.Stars
	}

	return total_top
}

func (this *dbPlayerInfoColumn) ChkSetCurMaxStage(v int32) bool {
	this.m_row.m_lock.UnSafeLock("dbPlayerInfoColumn.ChkSetCurMaxStage")
	defer this.m_row.m_lock.UnSafeUnlock()
	if this.m_data.CurMaxStage < v {
		this.m_data.CurMaxStage = v
		this.m_changed = true
		return true
	}
	return false
}

func (this *dbPlayerStageColumn) ChkGetTopScore(id int32) int32 {
	this.m_row.m_lock.UnSafeRLock("dbPlayerStageColumn.ChkGetTopScore")
	defer this.m_row.m_lock.UnSafeRUnlock()
	d := this.m_data[id]
	if d == nil {
		return 0
	}

	return d.TopScore
}

func (this *dbPlayerItemColumn) FillAllMsg(msg *msg_client_message.S2CGetItemInfos) {
	this.m_row.m_lock.UnSafeRLock("dbPlayerItemColumn.FillAllMsg")
	defer this.m_row.m_lock.UnSafeRUnlock()

	var tmp_item *msg_client_message.ItemInfo
	msg.Items = make([]*msg_client_message.ItemInfo, 0, len(this.m_data))
	for _, v := range this.m_data {
		if nil == v {
			continue
		}

		tmp_item = &msg_client_message.ItemInfo{}
		tmp_item.ItemCfgId = v.ItemCfgId
		tmp_item.ItemNum = v.ItemNum
		tmp_item.RemainSeconds = get_time_item_remain_seconds(v)
		msg.Items = append(msg.Items, tmp_item)
	}

	return
}

func (this *dbPlayerBuildingColumn) FillAllMsg(msg *msg_client_message.S2CGetBuildingInfos) {
	this.m_row.m_lock.UnSafeRLock("dbPlayerBuildingColumn.FillAllMsg")
	defer this.m_row.m_lock.UnSafeRUnlock()

	var tmp_build *msg_client_message.BuildingInfo
	msg.Builds = make([]*msg_client_message.BuildingInfo, 0, len(this.m_data))
	for _, v := range this.m_data {
		if nil == v {
			continue
		}

		tmp_build = &msg_client_message.BuildingInfo{}
		tmp_build.Id = v.Id
		tmp_build.CfgId = v.CfgId
		tmp_build.X = v.X
		tmp_build.Y = v.Y
		tmp_build.Dir = v.Dir
		msg.Builds = append(msg.Builds, tmp_build)
		if nil != map_chest_mgr.Map[v.CfgId] {
			tmp_time := time.Unix(int64(v.CreateUnix), 0)
			log.Info("宝箱[%d:%d]的开始时间 %s", v.Id, v.CfgId, tmp_time.Format("2006-01-02 15:04:05.999999999 -0700 MST"))
			tmp_time = time.Unix(int64(v.OverUnix), 0)
			log.Info("宝箱[%d:%d]的结束时间 %s", v.Id, v.CfgId, tmp_time.Format("2006-01-02 15:04:05.999999999 -0700 MST"))
		}
	}

	return
}

func (this *dbPlayerBuildingDepotColumn) FillAllMsg(msg *msg_client_message.S2CGetDepotBuildingInfos) {
	this.m_row.m_lock.UnSafeRLock("dbPlayerBuildingDepotColumn.FillAllMsg")
	defer this.m_row.m_lock.UnSafeRUnlock()

	var tmp_build *msg_client_message.DepotBuildingInfo
	msg.DepotBuilds = make([]*msg_client_message.DepotBuildingInfo, 0, len(this.m_data))
	for _, v := range this.m_data {
		if nil == v {
			continue
		}
		tmp_build = &msg_client_message.DepotBuildingInfo{}
		tmp_build.CfgId = v.CfgId
		tmp_build.Num = v.Num
		msg.DepotBuilds = append(msg.DepotBuilds, tmp_build)
	}
	return
}

func (this *dbPlayerCatColumn) FillAllMsg(msg *msg_client_message.S2CGetCatInfos) {
	this.m_row.m_lock.UnSafeRLock("dbPlayerCatColumn.FillAllMsg")
	defer this.m_row.m_lock.UnSafeRUnlock()

	var tmp_cat *msg_client_message.CatInfo
	msg.Cats = make([]*msg_client_message.CatInfo, 0, len(this.m_data))
	for _, v := range this.m_data {
		if nil == v {
			continue
		}

		tmp_cat = &msg_client_message.CatInfo{}
		tmp_cat.Id = v.Id
		tmp_cat.CatCfgId = v.CfgId
		tmp_cat.Level = v.Level
		tmp_cat.Star = v.Star
		tmp_cat.SkillLevel = v.SkillLevel
		lock := false
		if v.Locked > 0 {
			lock = true
		}
		tmp_cat.Locked = lock
		tmp_cat.Exp = v.Exp
		tmp_cat.CoinAbility = v.CoinAbility
		tmp_cat.ExploreAbility = v.ExploreAbility
		tmp_cat.MatchAbility = v.MatchAbility
		tmp_cat.Nick = v.Nick
		if v.CathouseId > 0 {
			tmp_cat.State = CAT_STATE_IN_CATHOUSE
		}
		msg.Cats = append(msg.Cats, tmp_cat)
	}

	return
}

func (this *dbPlayerAreaColumn) FillAllMsg(msg *msg_client_message.S2CGetAreasInfos) {
	this.m_row.m_lock.UnSafeRLock("dbPlayerAreaColumn.GetAll")
	defer this.m_row.m_lock.UnSafeRUnlock()

	var tmp_area *msg_client_message.AreaInfo
	msg.Areas = make([]*msg_client_message.AreaInfo, 0, len(this.m_data))
	for _, v := range this.m_data {
		if nil == v {
			continue
		}

		tmp_area = &msg_client_message.AreaInfo{}
		tmp_area.CfgId = v.CfgId
		msg.Areas = append(msg.Areas, tmp_area)
	}
	return
}

func (this *dbPlayerAreaColumn) GetAllAreaInfo() (all_area []*msg_client_message.AreaInfo) {
	this.m_row.m_lock.UnSafeRLock("dbPlayerAreaColumn.GetAllAreaInfo")
	defer this.m_row.m_lock.UnSafeRUnlock()

	var tmp_area *msg_client_message.AreaInfo
	all_area = make([]*msg_client_message.AreaInfo, 0, len(this.m_data))
	for _, v := range this.m_data {
		if nil == v {
			continue
		}

		tmp_area = &msg_client_message.AreaInfo{}
		tmp_area.CfgId = v.CfgId
		all_area = append(all_area, tmp_area)
	}
	return
}

func (this *dbPlayerStageColumn) FillAllMsg(msg *msg_client_message.S2CGetStageInfos) {
	this.m_row.m_lock.UnSafeRLock("dbPlayerStageColumn.GetAll")
	defer this.m_row.m_lock.UnSafeRUnlock()

	var tmp_stage *msg_client_message.StageInfo
	msg.Stages = make([]*msg_client_message.StageInfo, 0, len(this.m_data))
	for stageid, v := range this.m_data {
		if nil == v {
			continue
		}
		tmp_stage = &msg_client_message.StageInfo{}
		tmp_stage.StageId = stageid
		tmp_stage.TopScore = v.TopScore
		tmp_stage.Star = v.Stars
		msg.Stages = append(msg.Stages, tmp_stage)
	}
}

func (this *dbPlayerAreaColumn) GetAllIdxs() (list []int32) {
	this.m_row.m_lock.UnSafeRLock("dbPlayerAreaColumn.GetAll")
	defer this.m_row.m_lock.UnSafeRUnlock()
	list = make([]int32, 0, len(this.m_data))

	for _, v := range this.m_data {
		list = append(list, v.CfgId)
	}
	return
}

func (this *dbPlayerBuildingColumn) GetAllBuildingPos() (pos_map map[int32]int32, cur_area_block_count map[int32]int32) { // , cur_area_use_count map[int32]int32
	this.m_row.m_lock.UnSafeRLock("dbPlayerBuildingColumn.GetAllBuildingPos")
	defer this.m_row.m_lock.UnSafeRUnlock()

	pos_map = make(map[int32]int32, len(this.m_data))
	//cur_area_use_count = make(map[int32]int32)
	cur_area_block_count = make(map[int32]int32)
	var arena_xy, arena_id int32
	var building_cfg *tables.XmlBuildingItem
	for _, d := range this.m_data {
		if nil == d {
			continue
		}

		building_cfg = building_table_mgr.Map[d.CfgId]
		if nil == building_cfg {
			continue
		}

		var width, height int32
		if tables.BUILDING_DIR_BIG_X_DIR == d.Dir {
			width, height = building_cfg.MapSizes[0], building_cfg.MapSizes[1]
		} else {
			width, height = building_cfg.MapSizes[1], building_cfg.MapSizes[0]
		}

		if nil != block_table_mgr.Map[d.CfgId] {
			arena_xy = (d.X)<<16 | (d.Y)&0x0000FFFF
			arena_id = build_area_mgr.AreaXY2AreaId[arena_xy]
			if arena_id > 0 {
				cur_area_block_count[arena_id] = cur_area_block_count[arena_id] + 1
			}
		}

		for tmp_x := int32(0); tmp_x < width; tmp_x++ {
			for tmp_y := int32(0); tmp_y < height; tmp_y++ {
				arena_xy = (d.X+tmp_x)<<16 | (d.Y+tmp_y)&0x0000FFFF
				pos_map[arena_xy] = d.Id
				//arena_id = cfg_build_area_mgr.AreaXY2AreaId[arena_xy]
				//if arena_id > 0 {
				//cur_area_use_count[arena_id] = cur_area_use_count[arena_id] + 1
				//}
			}
		}
	}

	return
}

func (this *dbPlayerExpeditionColumn) CheckUpdateExpedition(p_lvl int32) (cur_ids map[int32]bool, cur_count int32) {
	var task_cfg *tables.XmlExpeditionItem
	del_map := make(map[int32]bool)
	this.m_row.m_lock.UnSafeLock("dbPlayerExpeditionsColumn.CheckUpdateExpedition")
	defer this.m_row.m_lock.UnSafeUnlock()

	//log.Info("dbPlayerExpeditionColumn CheckUpdateExpedition, m_data[%v]", this.m_data)

	cur_unix := int32(time.Now().Unix())
	//cur_count := int32(0)
	cur_ids = make(map[int32]bool)
	var pass_sec int32
	for _, val := range this.m_data {
		if nil == val {
			continue
		}

		task_cfg = expedition_table_mgr.Map[val.TaskId]
		if nil == task_cfg {
			continue
		}
		if PLAYER_EXPEDITION_STATE_DOING == val.CurState && cur_unix > val.EndUnix {
			if val.Result > 0 {
				val.CurState = PLAYER_EXPEDITION_STATE_SUCCEED
			} else {
				val.CurState = PLAYER_EXPEDITION_STATE_FAILED
			}
		}

		if PLAYER_EXPEDITION_TYPE_TIMELIMIT == task_cfg.TaskType && PLAYER_EXPEDITION_STATE_INIT == val.CurState {
			pass_sec = cur_unix - val.TaskLeftSecLastUpUnix
			if pass_sec >= val.TaskLeftSec {
				log.Info("限时任务超过期限%d pass_sec[%d-%d=%d]", val.TaskLeftSec, cur_unix, val.TaskLeftSecLastUpUnix, pass_sec)
				del_map[val.Id] = true
				continue
			} else {
				val.TaskLeftSec -= pass_sec
				val.TaskLeftSecLastUpUnix = cur_unix
			}

		}

		cur_count++
		cur_ids[val.TaskId] = true
		log.Info("CheckUpdateExpedition2 val [%v]", *val)
	}

	for id, _ := range del_map {
		delete(this.m_data, id)
		this.m_changed = true
	}

	log.Info("需要随机%d-%d个任务 删除了%d个任务", global_config.ExpeditionTaskCount, cur_count, len(del_map))

	//need_count = global_config_mgr.GetGlobalConfig().ExpeditionTaskCount - cur_count

	/*
		if cur_count < global_config_mgr.GetGlobalConfig().ExpeditionTaskCount {
			new_tasks := cfg_expedition_mgr.RandNWithExistIds(cur_ids, p_lvl, global_config_mgr.GetGlobalConfig().ExpeditionTaskCount-cur_count)
			var tmp_task *dbPlayerExpeditionData

			var rand_val, total_weight int32
			for _, task := range new_tasks {
				if nil == task {
					continue
				}

				tmp_task = &dbPlayerExpeditionData{TaskId: task.Id, StartUnix: int32(time.Now().Unix())}
				if PLAYER_EXPEDITION_TYPE_TIMELIMIT == task.TaskType {
					tmp_task.TaskLeftSecLastUpUnix = cur_unix
					tmp_task.TaskLeftSec = task.LimitTimeSec
					log.Info("设置限时任务的刷新时间 %d", tmp_task.TaskLeftSec)
				}

				// 随机任务条件
				log.Info("随机任务[%d]的条件", task.Id)
				tmp_task.Conditions = make([]dbExpeditionConData, task.NeedConditionNum)
				total_weight = task.TotalConWeight
				cur_map := make(map[int]bool)
				for cur_num := int32(0); cur_num < task.NeedConditionNum; cur_num++ {
					if total_weight <= 0 {
						log.Info("第%d次随机任务条件totalweight[%d]<0退出", cur_num+1, total_weight)
						break
					}

					rand_val = rand.Int31n(total_weight)
					log.Info("第%d次随机任务条件，totol_weight[%d] rand_val[%d] 当前随机好的对象%v", cur_num+1, total_weight, rand_val, cur_map)
					for idx, tmp_con := range task.Conditions {
						if cur_map[idx] {
							continue
						}

						log.Info("	===随机任务条件对比weight[%d] rand_val[%d]", tmp_con.Con_Weight, rand_val)
						if rand_val < tmp_con.Con_Weight {
							total_weight -= tmp_con.Con_Weight
							tmp_task.Conditions[cur_num].ConType = tmp_con.Con_Type
							if PLAYER_EXPEDITION_CON_CAT_COLOR == tmp_con.Con_Type {
								tmp_task.Conditions[cur_num].ConVals = make([]int32, 0, tmp_con.Con_Val)
								sub_total_weight := tmp_con.Ext_val
								sub_cur_map := make(map[int]bool)
								for sub_cur_num := int32(0); sub_cur_num < tmp_con.Con_Val; sub_cur_num++ {
									if sub_total_weight <= 0 {
										break
									}

									sub_rand_val := rand.Int31n(sub_total_weight)
									for sub_idx, color_weight := range tmp_con.Ext_vals {
										if sub_cur_map[sub_idx] {
											continue
										}

										if sub_rand_val < color_weight {
											tmp_task.Conditions[cur_num].ConVals = append(tmp_task.Conditions[cur_num].ConVals, int32(1<<(uint32(sub_idx))))
											sub_cur_map[sub_idx] = true
											sub_total_weight -= color_weight
											break
										} else {
											sub_rand_val -= color_weight
										}
									}
								}
							} else {
								tmp_task.Conditions[cur_num].ConVals = make([]int32, 1)
								tmp_task.Conditions[cur_num].ConVals[0] = tmp_con.Con_Val
							}
							cur_map[idx] = true
							break
						} else {
							rand_val -= tmp_con.Con_Weight
						}
					}

				}

				this.m_data[task.Id] = tmp_task

				log.Info("赋值任务[task.id]给m_data", tmp_task)
			}
		}

	*/

	return
}

func (this *dbPlayerExpeditionColumn) FillAllClientMsg(msg *msg_client_message.S2CRetAllExpedition) {
	if nil == msg {
		log.Error("dbPlayerExpeditionColumn FillAllClientMsg msg nil !")
		return
	}

	this.m_row.m_lock.UnSafeRLock("dbPlayerExpeditionsColumn.FillAllClientMsg")
	defer this.m_row.m_lock.UnSafeRUnlock()

	tmp_len := int32(len(this.m_data))
	if tmp_len <= 0 {
		return
	}

	cur_unix := int32(time.Now().Unix())

	msg.Tasks = make([]*msg_client_message.ExpeditionItem, 0, tmp_len)
	var tmp_item *msg_client_message.ExpeditionItem
	var task_cfg *tables.XmlExpeditionItem
	for _, val := range this.m_data {
		if nil == val {
			continue
		}

		task_cfg = expedition_table_mgr.Map[val.TaskId]
		if nil == task_cfg {
			log.Error("dbPlayerExpeditionColumn FillAllClientMsg no task cfg[%d]", val.TaskId)
			continue
		}

		tmp_item = &msg_client_message.ExpeditionItem{}
		if PLAYER_EXPEDITION_TYPE_TIMELIMIT == task_cfg.TaskType {
			tmp_item.TaskLeftSec = val.TaskLeftSec
		}

		tmp_item.Id = val.Id
		tmp_item.TaskId = val.TaskId
		tmp_item.InCatIds = val.InCatIds
		if val.EndUnix > cur_unix {
			tmp_item.ExpeditionLeftSec = val.EndUnix - cur_unix
		}

		tmp_item.ExpeditionPassSec = cur_unix - val.StartUnix

		tmp_item.Result = val.Result
		tmp_item.CurState = val.CurState

		// 条件
		tmp_item.Conditions = make([]*msg_client_message.ExpeditonCondition, 0, len(val.Conditions))
		for _, tmp_con := range val.Conditions {
			msg_con := &msg_client_message.ExpeditonCondition{}
			msg_con.ConditionType = tmp_con.ConType
			msg_con.ConVals = tmp_con.ConVals
			tmp_item.Conditions = append(tmp_item.Conditions, msg_con)
		}

		// 事件
		tmp_item.Events = make([]*msg_client_message.ExpeditonEvent, 0, len(val.EventIds))
		for _, tmp_event := range val.EventIds {
			msg_event := &msg_client_message.ExpeditonEvent{}
			msg_event.EventId = tmp_event.ClientId
			msg_event.Sec = tmp_event.Sec
			msg_event.DropIdNums = tmp_event.DropIdNums
			tmp_item.Events = append(tmp_item.Events, msg_event)
		}

		msg.Tasks = append(msg.Tasks, tmp_item)
	}

	return
}

func (this *dbPlayerExpeditionColumn) IfCatInExpedition(in_catid int32) bool {
	if in_catid <= 0 {
		log.Error("dbPlayerExpeditionColumn IfCatInExpedition")
		return true
	}

	this.m_row.m_lock.UnSafeRLock("dbPlayerExpeditionsColumn.FillAllClientMsg")
	defer this.m_row.m_lock.UnSafeRUnlock()

	for _, val := range this.m_data {
		if nil == val {
			continue
		}

		for _, catid := range val.InCatIds {
			if catid == in_catid {
				return true
			}
		}
	}

	return false
}

func (this *dbPlayerExpeditionColumn) Stop(taskid int32) {
	this.m_row.m_lock.UnSafeLock("dbPlayerExpeditionColumn.Stop")
	defer this.m_row.m_lock.UnSafeUnlock()
	d := this.m_data[taskid]
	if d == nil {
		log.Error("dbPlayerExpeditionColumn.Stop not exist %v %v", this.m_row.GetPlayerId(), taskid)
		return
	}

	d.CurState = PLAYER_EXPEDITION_STATE_INIT
	d.TaskLeftSecLastUpUnix = int32(time.Now().Unix())
	d.InCatIds = nil

	this.m_changed = true
	return
}

func (this *dbPlayerBuildingColumn) GetCountByType(b_type int32) (count int32) {
	this.m_row.m_lock.UnSafeRLock("dbPlayerBuildingColumn.GetCountByType")
	defer this.m_row.m_lock.UnSafeRUnlock()

	var building_cfg *tables.XmlBuildingItem
	for _, val := range this.m_data {
		building_cfg = building_table_mgr.Map[val.CfgId]
		if nil == building_cfg {
			continue
		}

		if b_type == building_cfg.Type {
			count++
		}
	}

	return
}

func (this *dbPlayerChapterUnLockColumn) SetNewUnlockChapter(chapter_id int32) {
	this.m_row.m_lock.UnSafeLock("dbPlayerChapterUnLockColumn.SetNewUnlockChapter")
	defer this.m_row.m_lock.UnSafeUnlock()
	this.m_data.ChapterId = chapter_id
	this.m_data.PlayerIds = nil
	this.m_data.CurHelpIds = nil
	this.m_data.StartUnix = int32(time.Now().Unix())
	this.m_changed = true
	return
}

func (this *dbPlayerChapterUnLockColumn) Reset() {
	this.m_row.m_lock.UnSafeLock("dbPlayerChapterUnLockColumn.SetNewUnlockChapter")
	defer this.m_row.m_lock.UnSafeUnlock()
	this.m_data.ChapterId = 0
	this.m_data.PlayerIds = nil
	this.m_data.CurHelpIds = nil
	this.m_data.StartUnix = 0
	this.m_changed = true
	return
}

func (this *dbPlayerBuildingColumn) ChkBuildingOver() (over_ids map[int32]bool) {
	cur_unix := int32(time.Now().Unix())
	over_ids = make(map[int32]bool)

	//log.Info("dbPlayerBuildingColumn ChkBuildingOver ")

	this.m_row.m_lock.UnSafeLock("dbPlayerBuildingColumn.ChkBuildingOver")
	defer this.m_row.m_lock.UnSafeUnlock()
	for _, d := range this.m_data {
		if nil == d {
			continue
		}

		if d.OverUnix > 0 && d.OverUnix < cur_unix {
			over_ids[d.Id] = true
		}
	}

	if len(over_ids) > 0 {
		for bid, _ := range over_ids {
			delete(this.m_data, bid)
		}
		this.m_changed = true
	}

	return
}

func (this *dbPlayerInfoColumn) ChkGetNextExpeditionId() (r int32) {
	this.m_row.m_lock.UnSafeLock("dbPlayerInfoColumn.ChkGetNextExpeditionId")
	defer this.m_row.m_lock.UnSafeUnlock()
	this.m_data.NextExpeditionId += 1
	if this.m_data.NextExpeditionId <= 0 {
		this.m_data.NextExpeditionId = 1
	}
	this.m_changed = true
	return this.m_data.NextExpeditionId
}
