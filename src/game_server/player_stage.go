package main

import (
	"math/rand"
	"mm_server/libs/log"
	"mm_server/libs/timer"
	"mm_server/proto/gen_go/client_message"
	"mm_server/src/common"
	"mm_server/src/tables"
	"sync"
	"time"

	"github.com/golang/protobuf/proto"
)

const (
	FINISHE_ALL_STAR = 3
)

type StagePassSession struct {
	SessionId    int32 // 会话Id
	ret          *msg_client_message.S2CStagePass
	b_center_ret bool
}

type StagePassMgr struct {
	sessions_lock *sync.RWMutex
	id2session    map[int32]*StagePassSession

	session_id_lock *sync.Mutex
	nex_session_id  int32
}

var stage_pass_mgr StagePassMgr

func (this *StagePassMgr) Init() bool {
	this.sessions_lock = &sync.RWMutex{}
	this.id2session = make(map[int32]*StagePassSession)

	return true
}

func (this *StagePassMgr) GetNextSessionId() int32 {
	this.sessions_lock.Lock()
	defer this.sessions_lock.Unlock()

	this.nex_session_id++
	return this.nex_session_id
}

func (this *StagePassMgr) AddSession(session *StagePassSession) {
	if nil == session {
		log.Error("StagePassMgr AddSession session nil !")
		return
	}

	this.sessions_lock.Lock()
	defer this.sessions_lock.Unlock()

	this.id2session[session.SessionId] = session

	return
}

func (this *StagePassMgr) PopSessionById(sid int32) *StagePassSession {
	this.sessions_lock.Lock()
	defer this.sessions_lock.Unlock()

	cur_ssession := this.id2session[sid]
	if nil == cur_ssession {
		return nil
	}

	delete(this.id2session, sid)

	return cur_ssession
}

// ============================================================================

func (this *Player) GetDayBuyTiLiCount() int32 {
	cur_unix_day := timer.GetDayFrom1970WithCfg(0)
	if cur_unix_day != this.db.Info.GetDayBuyTiLiUpDay() {
		this.db.Info.SetDayBuyTiLiCount(0)
		this.db.Info.SetDayBuyTiLiUpDay(cur_unix_day)
		return 0
	}

	return this.db.Info.GetDayBuyTiLiCount()
}

// ============================================================================

func (this *Player) CheckBeginStage(data *StageBeginData) bool {
	level := level_table_mgr.GetLevel(data.stage_id)
	if level == nil {
		return false
	}

	if level.NeedPower > this.CalcSpirit() {
		log.Error("Player[%v] not enough stamina to begin stage[%v]", this.Id, data.stage_id)
		return false
	}

	if this.stage_state == 1 {
		log.Warn("Player[%v] already begin stage[%v]", this.Id, data.stage_id)
	}

	this.SubSpirit(level.NeedPower, "pass_stage", "stage")

	if data.cat_id > 0 {
		if !this.db.Cats.HasIndex(data.cat_id) {
			return false
		}
	}

	if data.item_ids != nil {
		items := make(map[int32]int32)
		for i := 0; i < len(data.item_ids); i++ {
			if items[data.item_ids[i]] == 0 {
				items[data.item_ids[i]] = 1
			} else {
				items[data.item_ids[i]] += 1
			}
		}
		for k, v := range items {
			num := this.GetItemResourceValue(k)
			if num < v {
				log.Error("Player[%v] begin stage[%v] with item[%v,%v] not enough", this.Id, data.stage_id, k, num)
				return false
			}
		}
		for i := 0; i < len(data.item_ids); i++ {
			this.RemoveItemResource(data.item_ids[i], 1, "begin_stage", "stage")
		}
		this.SendItemsUpdate()
	}

	this.stage_id = data.stage_id
	this.stage_cat_id = data.cat_id
	this.stage_state = 1

	return true
}

func (this *Player) ChkFinishStage(stageid, star, score int32, ret_msg *msg_client_message.S2CStagePass, bforce bool) (top_score int32) {
	if this.stage_id != stageid {
		return int32(msg_client_message.E_ERR_STAGE_NO_MATCH_WITH_END)
	}
	if this.stage_state == 0 {
		return int32(msg_client_message.E_ERR_STAGE_ALREADY_FINISHED)
	}

	if !bforce {
		if stageid > this.db.Info.GetMaxUnlockStage() {
			return int32(msg_client_message.E_ERR_STAGE_PASS_NOT_UNLOCK)
		}

		cur_max_stage_id := this.db.Info.GetCurMaxStage()
		if cur_max_stage_id > 0 && stageid > cur_max_stage_id {
			cur_max_stage_cfg := level_table_mgr.Map[cur_max_stage_id]
			if nil == cur_max_stage_cfg {
				log.Error("Player ChkFinishStage faild to find cur_max_stage_cfg[%d] !", cur_max_stage_id)
				return int32(msg_client_message.E_ERR_STAGE_PASS_NOT_UNLOCK)
			}

			if cur_max_stage_cfg.NextLevel != stageid {
				return int32(msg_client_message.E_ERR_STAGE_PASS_OVER_NEXT_STATE)
			}
		}
	}

	stagecfg := level_table_mgr.Map[stageid]
	if nil == stagecfg {
		log.Error("Player ChkFinishStage failed to find stage[%d]", stageid)
		return int32(msg_client_message.E_ERR_STAGE_TABLE_DATA_NOT_FOUND)
	}

	bfirst := false
	bfirst_3star := false
	cur_stage_db := this.db.Stages.Get(stageid)
	old_top_score := int32(0)
	if nil == cur_stage_db {
		new_db := &dbPlayerStageData{}
		new_db.LastFinishedUnix = int32(time.Now().Unix())
		new_db.StageId = stageid
		new_db.Stars = star
		new_db.TopScore = score
		this.db.Stages.Add(new_db)
		bfirst = true
		top_score = score
		this.AddStar(star, "pass_stage", "stage")
		if star >= FINISHE_ALL_STAR {
			bfirst_3star = true
		}
	} else {
		old_top_score, _ = this.db.Stages.GetTopScore(stageid)
		top_score = this.db.Stages.ChkSetTopScore(stageid, score)
		if star > cur_stage_db.Stars {
			this.db.Stages.SetStars(stageid, star)
			this.AddStar(star-cur_stage_db.Stars, "pass_stage", "stage")
			if cur_stage_db.Stars < FINISHE_ALL_STAR && star >= FINISHE_ALL_STAR {
				bfirst_3star = true
			}
		}
	}

	// update ranking list, the score is top score
	if score > old_top_score {
		/*if this.rpc_call_rank_update_stage_total_score(this.db.Stages.GetTotalScore()) == nil {
			log.Warn("Player[%v] update stages total score failed", this.Id)
		}
		if this.rpc_call_rank_update_stage_score(stageid, top_score) == nil {
			log.Warn("Player[%v] update stage[%v] top score[%v] failed", this.Id, stageid, score)
		}*/
	}

	if this.db.Info.ChkSetCurMaxStage(stageid) {
		this.b_base_prop_chg = true
	}

	// 给予首次通关奖励
	var tmp_item *msg_client_message.ItemInfo
	var tmp_cat *msg_client_message.CatInfo
	var tmp_building *msg_client_message.DepotBuildingInfo

	ret_msg.GetItems = make([]*msg_client_message.ItemInfo, 0)
	ret_msg.GetCats = make([]*msg_client_message.CatInfo, 0)
	ret_msg.GetBuildings = make([]*msg_client_message.DepotBuildingInfo, 0)
	if bfirst {
		ret_msg.GetItemsFirst = make([]*msg_client_message.ItemInfo, 0, len(stagecfg.FirstClearReward)/2)
		log.Info("首次通关[%d]给予奖励 %v", stageid, stagecfg.FirstClearReward)
		for i := 0; i < len(stagecfg.FirstClearReward)/2; i++ {
			tmp_item = &msg_client_message.ItemInfo{}
			tmp_item.ItemCfgId = stagecfg.FirstClearReward[2*i]
			tmp_item.ItemNum = stagecfg.FirstClearReward[2*i+1]
			ret_msg.GetItemsFirst = append(ret_msg.GetItemsFirst, tmp_item)

			this.AddItemResource(stagecfg.FirstClearReward[2*i], stagecfg.FirstClearReward[2*i+1], "FirstClearReward", "Stage")
		}
	}
	if /*FINISHE_ALL_STAR == star && (bfirst || cur_stage_db.Stars < FINISHE_ALL_STAR)*/ bfirst_3star {
		ret_msg.GetItems3Star = make([]*msg_client_message.ItemInfo, 0, len(stagecfg.FirstAllStarReward)/2)
		for i := 0; i < len(stagecfg.FirstAllStarReward)/2; i++ {
			tmp_item = &msg_client_message.ItemInfo{}
			tmp_item.ItemCfgId = stagecfg.FirstAllStarReward[2*i]
			tmp_item.ItemNum = stagecfg.FirstAllStarReward[2*i+1]
			ret_msg.GetItems3Star = append(ret_msg.GetItems3Star, tmp_item)

			this.AddItemResource(stagecfg.FirstAllStarReward[2*i], stagecfg.FirstAllStarReward[2*i+1], "StageFirstAllStar", "Stage")
		}
		log.Debug("Player[%v] First Finish Stage[%v], Stamina[%v]", this.Id, stageid, this.db.Info.GetSpirit())
	}

	// 额外增加的金币
	extra_coin := this.get_cat_match_ability(this.stage_cat_id) * stagecfg.CoinReward / 100
	ret_msg.CatExtraAddCoin = extra_coin
	log.Debug("@@@@ stage_cat_id[%v] extra_coin[%v]", this.stage_cat_id, extra_coin)

	// 给予普通奖励
	this.AddGold(stagecfg.CoinReward+extra_coin, "StagePass", "Stage")
	ret_msg.GetCoin = stagecfg.CoinReward + extra_coin
	coin_item := &msg_client_message.ItemInfo{
		ItemCfgId: ITEM_RESOURCE_ID_GOLD,
		ItemNum:   stagecfg.CoinReward + extra_coin,
	}
	ret_msg.GetItems = append(ret_msg.GetItems, coin_item)

	var b bool
	if len(stagecfg.ExtraReward1) == 2 && rand.Int31n(100) < stagecfg.ExtraReward1[1] {
		//this.AddItem(stagecfg.ExtraReward1, 1, "StagePass", "Stage", true)
		b, tmp_item, tmp_cat, tmp_building = this.drop_item_by_id(stagecfg.ExtraReward1[0], false)
		if b {
			if tmp_item != nil {
				ret_msg.GetItems = append(ret_msg.GetItems, tmp_item)
			}
			if tmp_cat != nil {
				ret_msg.GetCats = append(ret_msg.GetCats, tmp_cat)
			}
			if tmp_building != nil {
				ret_msg.GetBuildings = append(ret_msg.GetBuildings, tmp_building)
			}
			//this.AddItem(stagecfg.ExtraReward1, 1, "StageFirstAllStar", "Stage", true)
		}
	}

	if len(stagecfg.ExtraReward2) == 2 && rand.Int31n(100) < stagecfg.ExtraReward2[1] {
		//this.AddItem(stagecfg.ExtraReward2, 1, "StagePass", "Stage", true)
		b, tmp_item, tmp_cat, tmp_building = this.drop_item_by_id(stagecfg.ExtraReward2[0], false)
		if b {
			if tmp_item != nil {
				ret_msg.GetItems = append(ret_msg.GetItems, tmp_item)
			}
			if tmp_cat != nil {
				ret_msg.GetCats = append(ret_msg.GetCats, tmp_cat)
			}
			if tmp_building != nil {
				ret_msg.GetBuildings = append(ret_msg.GetBuildings, tmp_building)
			}
			//this.AddItem(stagecfg.ExtraReward2, 1, "StageFirstAllStar", "Stage", true)
		}
	}

	this.stage_state = 0

	return
}

// ============================================================================

type StageBeginData struct {
	stage_id int32
	cat_id   int32
	item_ids []int32
}

func C2SStagePassBeginHandler(p *Player, msg_data []byte) int32 {
	var req msg_client_message.C2SStageBegin
	err := proto.Unmarshal(msg_data, &req)
	if err != nil {
		log.Error("unmarshal msg failed %v", err.Error())
		return -1
	}

	var data = StageBeginData{
		stage_id: req.GetStageId(),
		cat_id:   req.GetCatId(),
		item_ids: req.GetItemIds(),
	}
	if !p.CheckBeginStage(&data) {
		return -1
	}

	response := &msg_client_message.S2CStageBeginResult{}
	response.StageId = req.GetStageId()
	p.Send(uint16(msg_client_message.S2CStageBeginResult_ProtoID), response)

	log.Trace("Player %v begin stage %v", p.Id, req.GetStageId())

	return 1
}

func (this *dbPlayerStageColumn) GetTotalScore() int32 {
	this.m_row.m_lock.UnSafeRLock("dbPlayerStageColumn.GetTotalScore")
	defer this.m_row.m_lock.UnSafeRUnlock()

	total_score := int32(0)
	for _, v := range this.m_data {
		total_score += v.TopScore
	}
	return total_score
}

func (this *Player) _get_friend_stage_score(stage_id int32) []*msg_client_message.PlayerStageInfo {
	result := this.rpc_get_friends_stage_score(stage_id)
	if result == nil {
		return nil
	}
	var score_list []*msg_client_message.PlayerStageInfo
	for _, r := range result.FriendsScoreData {
		score_list = append(score_list, &msg_client_message.PlayerStageInfo{
			PlayerId: r.Id,
			Score:    r.StageScore,
			Name:     r.Name,
			Lvl:      r.Level,
			Head:     r.Head,
		})
	}
	return score_list
}

func (p *Player) stage_pass(result int32, stageid int32, score int32, stars int32, items []*msg_client_message.ItemInfo, bforce bool) int32 {
	new_session := &StagePassSession{}
	new_session.SessionId = stage_pass_mgr.GetNextSessionId()
	tmp_ret := &msg_client_message.S2CStagePass{}
	tmp_ret.StageId = stageid
	tmp_ret.Score = score
	tmp_ret.Stars = stars
	tmp_ret.UseItems = items
	tmp_ret.Result = result

	// 未过关
	if result == 0 {
		tmp_ret.FriendItems = make([]*msg_client_message.PlayerStageInfo, 0)
		tmp_ret.GetBuildings = make([]*msg_client_message.DepotBuildingInfo, 0)
		tmp_ret.GetCats = make([]*msg_client_message.CatInfo, 0)
		tmp_ret.GetCoin = 0
		tmp_ret.GetItems = make([]*msg_client_message.ItemInfo, 0)
		p.Send(uint16(msg_client_message.S2CStagePass_ProtoID), tmp_ret)
		return 1
	}

	top_score := p.ChkFinishStage(stageid, stars, score, tmp_ret, bforce)
	if top_score < 0 {
		return top_score
	}

	if result > 0 {
		p.db.Stages.IncbyPassCount(stageid, 1)
	}
	p.db.Stages.IncbyPlayedCount(stageid, 1)

	new_session.ret = tmp_ret
	new_session.ret.TopScore = top_score

	// 物品消耗
	if items != nil {
		for i := 0; i < len(items); i++ {
			p.RemoveItem(items[i].GetItemCfgId(), items[i].GetItemNum(), true)
		}
	}

	// 更新任务
	p.TaskUpdate(tables.TASK_COMPLETE_TYPE_PASS_NUM, false, 0, 1)
	level := level_table_mgr.GetLevel(stageid)
	if level != nil {
		chapter_levels := level_table_mgr.GetChapter(level.Chapter)
		if chapter_levels != nil {
			c := true
			for _, v := range chapter_levels.Levels {
				if !p.db.Stages.HasIndex(v.Id) {
					c = false
					break
				}
			}
			if c {
				p.TaskUpdate(tables.TASK_COMPLETE_TYPE_PASS_CHAPTER, false, stageid, 1)
			}
		}
	}

	log.Trace("Stage Pass res %v", new_session.ret)

	p.SendItemsUpdate()
	p.SendCatsUpdate()
	p.SendDepotBuildingUpdate()

	new_session.ret.FriendItems = p._get_friend_stage_score(stageid)
	p.Send(uint16(msg_client_message.S2CStagePass_ProtoID), new_session.ret)

	p.send_stage_info()

	p.rpc_rank_list_update_data(common.RANK_LIST_TYPE_STAGE_TOTAL_SCORE, []int32{p.db.Stages.GetTotalScore(), stageid, score})

	return 1
}

func C2SStagePassHandler(p *Player, msg_data []byte) int32 {
	var req msg_client_message.C2SStagePass
	err := proto.Unmarshal(msg_data, &req)
	if err != nil {
		log.Error("unmarshal msg failed %v", err.Error())
		return -1
	}

	stageid := req.GetStageId()
	score := req.GetScore()
	stars := req.GetStars()

	return p.stage_pass(req.GetResult(), stageid, score, stars, req.GetItems(), false)
}

/*func C2SDayBuyTiLiHandler(p *Player, msg_data []byte) int32 {
	global_cfg := global_config_mgr.GetGlobalConfig()
	cur_count := p.GetDayBuyTiLiCount()
	if cur_count >= global_cfg.MaxDayBuyTiLiCount {
		return int32(msg_client_message.E_ERR_DAYBUY_TILI_MAX_COUNT)
	}

	if p.GetDiamond() < global_cfg.DayBuyTiLiCost {
		return int32(msg_client_message.E_ERR_DAYBUY_TILI_LESS_DIAMOND)
	}

	p.db.Info.SetDayBuyTiLiCount(cur_count + 1)
	p.SubDiamond(global_cfg.DayBuyTiLiCost, "DayBuyTiLi", "Stage")
	p.AddSpirit(global_cfg.DayBuyTiliAdd, "DayBuyTiLi", "Stage")

	return 1
}*/

// -------------------------
