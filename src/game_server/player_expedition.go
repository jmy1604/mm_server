package main

import (
	"math/rand"
	"mm_server/libs/log"
	"mm_server/libs/timer"
	"mm_server/proto/gen_go/client_message"

	"mm_server/src/tables"
	"time"

	"github.com/golang/protobuf/proto"
)

const (
	PLAYER_EXPEDITION_CON_CAT_LVL   = 1 // 猫咪等级条件
	PLAYER_EXPEDITION_CON_CAT_QUA   = 2 // 猫咪品阶条件
	PLAYER_EXPEDITION_CON_CAT_STAR  = 3 // 猫咪星阶条件
	PLAYER_EXPEDITION_CON_CAT_COLOR = 4 // 猫咪颜色条件
	PLAYER_EXPEDITION_CON_CAT_NUM   = 5 // 猫咪数量条件
)

const (
	PLAYER_EXPEDITION_TYPE_NORMAL    = 1
	PLAYER_EXPEDITION_TYPE_TIMELIMIT = 2

	PLAYER_EXPEDITION_STATE_INIT    = 0 // 探险任务初始状态
	PLAYER_EXPEDITION_STATE_DOING   = 1 // 探险任务探险中状态
	PLAYER_EXPEDITION_STATE_SUCCEED = 2 // 探险任务成功状态
	PLAYER_EXPEDITION_STATE_FAILED  = 3 // 探险任务失败状态

	PLAYER_EXPEDITION_RESULT_FAILED  = 0 // 探险任务结果失败
	PLAYER_EXPEDITION_RESULT_SUCCEED = 1 // 探险任务结果成功
)

func (this *Player) GetCatExpeditionVal(catid int32) int32 {
	cur_cat := this.db.Cats.Get(catid)
	if nil == cur_cat {
		return 0
	}

	cat_cfg := cat_table_mgr.Map[cur_cat.CfgId]
	if nil == cat_cfg {
		log.Error("player GetCatExpeditionVal cat_cfg[%d] nil", cur_cat.CfgId)
		return 0
	}

	return cat_cfg.GrowthRate*cur_cat.Level/100 + cur_cat.ExploreAbility + cur_cat.ExploreAbility*cat_cfg.InitialRate*(cur_cat.Level-1)/100 + cat_cfg.AddExplores[cur_cat.Star-1]
}

func (this *Player) GetAviTotalCatExpditionVal(catids []int32) (int32, int32) {
	if len(catids) < 1 {
		log.Error("Player GetAviTotalCatExpditionVal catids len < 1")
		return 0, 0
	}
	total_val := int32(0)
	total_count := int32(0)
	for _, catid := range catids {
		total_val += this.GetCatExpeditionVal(catid)
		total_count++
	}

	return total_val / total_count, total_val
}

func (this *Player) GetTodayExpeditionCount() int32 {
	cur_unix_day := timer.GetDayFrom1970WithCfg(0)
	if this.db.Info.GetDayExpeditionUpDay() != cur_unix_day {
		log.Info("需要重置当前探险任务次数 %d %d", this.db.Info.GetDayExpeditionUpDay(), cur_unix_day)
		this.db.Info.SetDayExpeditionCount(0)
		this.db.Info.SetDayExpeditionUpDay(cur_unix_day)
		return 0
	}

	return this.db.Info.GetDayExpeditionCount()
}

func (this *Player) ChkDayExpeditionChgCount() (cur_count int32, cur_cost int32) {
	cur_unix_day := timer.GetDayFrom1970WithCfg(0)
	if this.db.Info.GetDayChgExpeditionUpDay() != cur_unix_day {
		log.Info("需要重置当前弹性任务刷新次数 %d %d", this.db.Info.GetDayChgExpeditionUpDay(), cur_unix_day)
		this.db.Info.SetDayChgExpeditionCount(0)
		this.db.Info.SetDayChgExpeditionUpDay(cur_unix_day)
		return 0, 0
	}

	cur_count = this.db.Info.GetDayChgExpeditionCount()
	if cur_count >= global_config.ExpeditionDayFreeChgCount {
		cur_cost = (cur_count + 1 - global_config.ExpeditionDayFreeChgCount) * global_config.ExpeditionDayChgAddCost
		if cur_cost > global_config.ExpeditionDayMaxChgCost {
			cur_cost = global_config.ExpeditionDayMaxChgCost
		}
	}

	return
}

func (this *Player) OnLoginExpeditionChk() {
	this.CheckUpdateExpedition()
}

func (p *Player) CheckUpdateExpedition() {
	cur_unix := int32(time.Now().Unix())
	cur_ids, cur_count := p.db.Expeditions.CheckUpdateExpedition(p.db.Info.GetLvl())
	log.Info("C2SGetAllExpeditionHandler %v %v", cur_ids, cur_count)
	if cur_count < global_config.ExpeditionTaskCount {
		new_tasks := expedition_table_mgr.RandNWithExistIds(cur_ids, p.db.Info.GetLvl(), global_config.ExpeditionTaskCount-cur_count)
		var tmp_task *dbPlayerExpeditionData

		var rand_val, total_weight int32
		for _, task := range new_tasks {
			if nil == task {
				continue
			}

			tmp_task = &dbPlayerExpeditionData{TaskId: task.Id, StartUnix: cur_unix}
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

			tmp_task.Id = p.db.Info.ChkGetNextExpeditionId()
			p.db.Expeditions.Add(tmp_task)
			log.Info("赋值任务[task.id]给m_data", tmp_task)
		}
	}
}

// ----------------------------------------------------------------------------

func C2SGetAllExpeditionHandler(p *Player, msg_data []byte) int32 {
	var req msg_client_message.C2SGetAllExpedition
	err := proto.Unmarshal(msg_data, &req)
	if err != nil {
		log.Error("unmarshal msg failed %v", err.Error())
		return -1
	}

	p.CheckUpdateExpedition()

	res_2cli := &msg_client_message.S2CRetAllExpedition{}
	p.db.Expeditions.FillAllClientMsg(res_2cli)
	if len(res_2cli.Tasks) < 1 {
		return 0
	}

	cur_count, cur_cost := p.ChkDayExpeditionChgCount()
	res_2cli.CurChgCount = global_config.ExpeditionDayFreeChgCount - cur_count
	if res_2cli.GetCurChgCount() < 0 {
		res_2cli.CurChgCount = 0
	}
	res_2cli.CurChgCost = cur_cost
	res_2cli.CurChgCost = cur_cost

	p.Send(uint16(msg_client_message.S2CRetAllExpedition_ProtoID), res_2cli)

	return 1
}

func C2SStartExpeditionHandler(p *Player, msg_data []byte) int32 {
	var req msg_client_message.C2SStartExpedition
	err := proto.Unmarshal(msg_data, &req)
	if err != nil {
		log.Error("unmarshal msg failed %v", err.Error())
		return -1
	}

	task_id := req.GetId()
	task_db := p.db.Expeditions.Get(task_id)
	if nil == task_db {
		return int32(msg_client_message.E_ERR_EXPEDITION_NO_TASK)
	}

	task_cfg := expedition_table_mgr.Map[task_db.TaskId]
	if nil == task_cfg {
		return int32(msg_client_message.E_ERR_EXPEDITION_NO_TASK)
	}

	if PLAYER_EXPEDITION_STATE_SUCCEED == task_db.CurState || PLAYER_EXPEDITION_STATE_DOING == task_db.CurState {
		return int32(msg_client_message.E_ERR_EXPEDITION_TASK_DOING)
	}

	if p.GetTodayExpeditionCount() >= global_config.ExpeditionDayStartCount {
		return int32(msg_client_message.E_ERR_EXPEDITION_START_LESS_COUNT)
	}

	action_item_num := p.GetItemResourceValue(ITEM_RESOURCE_ID_ACTION)
	if action_item_num < 1 {
		return int32(msg_client_message.E_ERR_ITEM_ACTION_NOT_ENOUGH)
	}

	catids := req.GetCatIds()
	tmp_len := int32(len(catids))
	if tmp_len < 1 {
		return int32(msg_client_message.E_ERR_EXPEDITION_LESS_CAT)
	}
	var cat_id, idx int32
	var cat_db *dbPlayerCatData
	var cat_cfg *tables.XmlCharacterItem
	var c_db_len, c_cfg_len int32
	all_catdbs := make([]*dbPlayerCatData, 0, tmp_len)
	all_catcfgs := make([]*tables.XmlCharacterItem, 0, tmp_len)
	for idx := int32(0); idx < tmp_len; idx++ {
		cat_id = catids[idx]

		cat_db = p.db.Cats.Get(cat_id)
		if nil == cat_db {
			return int32(msg_client_message.E_ERR_EXPEDITION_LESS_CAT)
		}

		cat_cfg = cat_table_mgr.Map[cat_db.CfgId]
		if nil == cat_cfg {
			return int32(msg_client_message.E_ERR_EXPEDITION_NOT_CAT_CFG)
		}

		if p.IfCatBusy(cat_id) {
			return int32(msg_client_message.E_ERR_EXPEDITION_CAT_WORKING)
		}

		all_catdbs = append(all_catdbs, cat_db)
		all_catcfgs = append(all_catcfgs, cat_cfg)
	}

	c_db_len = int32(len(all_catdbs))
	c_cfg_len = int32(len(all_catcfgs))

	for _, tmp_con := range task_db.Conditions {
		switch tmp_con.ConType {
		case PLAYER_EXPEDITION_CON_CAT_LVL:
			{
				for idx = int32(0); idx < c_db_len; idx++ {
					cat_db = all_catdbs[idx]
					if len(tmp_con.ConVals) < 1 || cat_db.Level < tmp_con.ConVals[0] {
						log.Info("==con== cat[%d] lvl %d %v", cat_db.Id, cat_db.Level, tmp_con.ConVals)
						return int32(msg_client_message.E_ERR_EXPEDITION_LESS_LEVEL)
					}
				}
			}
		case PLAYER_EXPEDITION_CON_CAT_QUA:
			{
				for idx = int32(0); idx < c_cfg_len; idx++ {
					cat_cfg = all_catcfgs[idx]
					if len(tmp_con.ConVals) < 1 || cat_cfg.Rarity < tmp_con.ConVals[0] {
						log.Info("==con== cat[%d] qua %d %v", cat_db.Id, cat_cfg.Rarity, tmp_con.ConVals)
						return int32(msg_client_message.E_ERR_EXPEDITION_LESS_QUA)
					}
				}
			}
		case PLAYER_EXPEDITION_CON_CAT_STAR:
			{
				for idx = int32(0); idx < c_db_len; idx++ {
					cat_db = all_catdbs[idx]
					if len(tmp_con.ConVals) < 1 || cat_db.Star < tmp_con.ConVals[0] {
						log.Info("==con== cat[%d] qua %d %v", cat_db.Id, cat_cfg.Rarity, tmp_con.ConVals)
						return int32(msg_client_message.E_ERR_EXPEDITION_LESS_STAR)
					}
				}
			}
		case PLAYER_EXPEDITION_CON_CAT_COLOR:
			{
				var total_count int32
				for idx = int32(0); idx < c_cfg_len; idx++ {
					cat_cfg = all_catcfgs[idx]

					total_count = 0
					for _, val := range tmp_con.ConVals {
						total_count += cat_cfg.Color & val
					}

					if total_count < 1 {
						log.Info("==con==cat[%d] color[%d] not in convals[%v]", cat_cfg.Id, cat_cfg.Color, tmp_con.ConVals)
						return int32(msg_client_message.E_ERR_EXPEDITION_WRONG_COLOR)
					}
				}

			}
		case PLAYER_EXPEDITION_CON_CAT_NUM:
			{
				if len(tmp_con.ConVals) < 1 || c_db_len < tmp_con.ConVals[0] {
					return int32(msg_client_message.E_ERR_EXPEDITION_LESS_CAT)
				}
			}
		}
	}

	res_2cli := &msg_client_message.S2CStartExpedition{}
	p.db.Expeditions.SetStartUnix(task_id, int32(time.Now().Unix()))
	p.db.Expeditions.SetEndUnix(task_id, int32(time.Now().Unix())+task_cfg.CostTime)
	p.db.Expeditions.SetInCatIds(task_id, catids)
	p.db.Expeditions.SetCurState(task_id, PLAYER_EXPEDITION_STATE_DOING)
	expe_avi, expe_total := p.GetAviTotalCatExpditionVal(catids)
	log.Info("返回的猫的平均值%d总值%d", expe_avi, expe_total)
	if rand.Int31n(10000) < expe_avi*task_cfg.SucceedBaseRate { // /10000
		p.db.Expeditions.SetResult(task_id, 1)
		res_2cli.Result = 1
	} else {
		res_2cli.Result = 0
	}

	// 随机任务特殊事件
	var tmp_event *dbExpeditionEventData
	var tmp_e_cfg *tables.XmlExpeditionEventItem
	tmp_sec := global_config.ExpeditionSPEventSec
	event_map := make(map[int32]*dbExpeditionEventData)
	event_rate := expe_total * task_cfg.EventBaseRate // / 10000
	event_ratebless := int32(0)                       //成功率祝福值
	event_count := int32(0)
	var totol_special_num, rand_val, dif_val int32
	var msg_event *msg_client_message.ExpeditonEvent

	for cur_sec := tmp_sec; cur_sec < task_cfg.CostTime; cur_sec += tmp_sec {

		rand_val = rand.Int31n(10000)
		if event_ratebless >= 10000 || rand_val <= event_rate {
			event_ratebless = 0 //清空祝福
		} else {
			log.Info("随机%d秒的特殊事件失败 rate %d %d", cur_sec, rand_val, event_rate)
			event_ratebless = event_ratebless + event_rate //失败补偿祝福
			continue
		}

		tmp_e_cfg = expedition_table_mgr.RandEvent(task_cfg.SearchEventId)
		if nil == tmp_e_cfg {
			log.Info("随机%d秒的特殊事件失败 缺少配置%d", cur_sec, task_cfg.SearchEventId)
			continue
		}

		tmp_event = &dbExpeditionEventData{}
		tmp_event.EventId = tmp_e_cfg.SearchEventID
		tmp_event.Sec = cur_sec
		tmp_event.ClientId = tmp_e_cfg.ClientId

		//bret, items, cats, buildings = p.DropItems2(tmp_e_cfg.DropIds, false)
		//if !bret {
		//log.Info("随机%d秒的特殊事件的奖励失败 %v", cur_sec, tmp_e_cfg.DropIds)
		//continue
		//}

		//items_len = int32(len(items))
		//cats_len = int32(len(cats))
		//buildings_len = int32(len(buildings))
		//tmp_num = (items_len + cats_len + buildings_len) * 2
		//totol_special_num += tmp_num
		tmp_len = int32(len(tmp_e_cfg.DropIds))
		tmp_event.DropIdNums = make([]int32, 0, tmp_len/3)
		for jdx := int32(0); jdx+2 < tmp_len; jdx += 3 {
			dif_val = tmp_e_cfg.DropIds[2] - tmp_e_cfg.DropIds[1]
			if dif_val > 0 {
				tmp_event.DropIdNums = append(tmp_event.DropIdNums, tmp_e_cfg.DropIds[0])
				tmp_event.DropIdNums = append(tmp_event.DropIdNums, tmp_e_cfg.DropIds[1]+rand.Int31n(dif_val)+1)
			} else {
				tmp_event.DropIdNums = append(tmp_event.DropIdNums, tmp_e_cfg.DropIds[0])
				tmp_event.DropIdNums = append(tmp_event.DropIdNums, tmp_e_cfg.DropIds[1])
			}
		}

		/*
			if items_len > 0 {

				for idx = 0; idx < items_len; idx++ {
					tmp_event.DropIdNums = append(tmp_event.DropIdNums, items[idx].GetItemCfgId())
					tmp_event.DropIdNums = append(tmp_event.DropIdNums, items[idx].GetItemNum())
				}
			}

			if cats_len > 0 {

				for idx = 0; idx < cats_len; idx++ {
					tmp_event.DropIdNums = append(tmp_event.DropIdNums, cats[idx].GetCatCfgId())
					tmp_event.DropIdNums = append(tmp_event.DropIdNums, 1)
				}
			}

			if buildings_len > 0 {

				for idx = 0; idx < buildings_len; idx++ {
					tmp_event.DropIdNums = append(tmp_event.DropIdNums, buildings[idx].GetCfgId())
					tmp_event.DropIdNums = append(tmp_event.DropIdNums, buildings[idx].GetNum())
				}
			}
		*/

		event_map[cur_sec] = tmp_event
		event_count++
	}

	dbEvents := make([]dbExpeditionEventData, 0, event_count)
	res_2cli.Events = make([]*msg_client_message.ExpeditonEvent, 0, event_count)
	event_count = 0
	for _, tmp_e := range event_map {
		dbEvents = append(dbEvents, *tmp_e)
		msg_event = &msg_client_message.ExpeditonEvent{}
		msg_event.Sec = tmp_e.Sec
		msg_event.EventId = tmp_e.ClientId
		msg_event.DropIdNums = tmp_e.DropIdNums
		res_2cli.Events = append(res_2cli.Events, msg_event)
	}
	p.db.Expeditions.SetEventIds(task_id, dbEvents)
	p.db.Expeditions.SetTotalSpecials(task_id, totol_special_num)
	p.RemoveItemResource(ITEM_RESOURCE_ID_ACTION, 1, "start expedition", "expedition")

	res_2cli.Id = task_id
	res_2cli.CatIds = catids
	res_2cli.ExpeditionLeftSec = task_cfg.CostTime
	res_2cli.CurState = PLAYER_EXPEDITION_STATE_DOING

	p.Send(uint16(msg_client_message.S2CStartExpedition_ProtoID), res_2cli)

	p.send_cats_update(catids)
	p.SendItemsUpdate()

	return 1
}

func C2SChgExpeditionHandler(p *Player, msg_data []byte) int32 {
	var req msg_client_message.C2SChgExpedition
	err := proto.Unmarshal(msg_data, &req)
	if err != nil {
		log.Error("unmarshal msg failed %v", err.Error())
		return -1
	}

	unique_id := req.GetId()
	task_db := p.db.Expeditions.Get(unique_id)
	if nil == task_db {
		return int32(msg_client_message.E_ERR_EXPEDITION_NO_TASK)
	}

	task_cfg := expedition_table_mgr.Map[task_db.TaskId]
	if nil == task_cfg {
		return int32(msg_client_message.E_ERR_EXPEDITION_NO_TASK)
	}

	if PLAYER_EXPEDITION_TYPE_TIMELIMIT == task_cfg.TaskType {
		return int32(msg_client_message.E_ERR_EXPEDITION_CANT_RM_LIMIT_TASK)
	}

	cur_count := p.db.Info.GetDayChgExpeditionCount()
	log.Info("当前探险任务刷新次数%d  %d", cur_count, global_config.ExpeditionDayFreeChgCount)

	p.db.Info.SetDayChgExpeditionCount(cur_count + 1)
	p.db.Info.SetDayChgExpeditionUpDay(timer.GetDayFrom1970WithCfg(0))
	log.Info("设置新的任务刷新次数%d %d ", p.db.Info.GetDayChgExpeditionCount(), p.db.Info.GetDayChgExpeditionUpDay())

	cur_count, cur_cost := p.ChkDayExpeditionChgCount()
	if cur_cost > p.db.Info.GetDiamond() {
		return int32(msg_client_message.E_ERR_EXPEDITION_LESS_DIAMOND)
	}

	p.SubDiamond(cur_cost, "expedition_chg", "expedition")

	cats_id, _ := p.db.Expeditions.GetInCatIds(unique_id)
	p.db.Expeditions.Remove(unique_id)
	p.send_cats_update(cats_id)

	p.CheckUpdateExpedition()

	res_2cli := &msg_client_message.S2CRetAllExpedition{}
	p.db.Expeditions.FillAllClientMsg(res_2cli)

	res_2cli.CurChgCount = global_config.ExpeditionDayFreeChgCount - cur_count
	if res_2cli.GetCurChgCount() < 0 {
		res_2cli.CurChgCount = 0
	}
	res_2cli.CurChgCost = cur_cost

	if len(res_2cli.Tasks) < 1 {
		return 0
	}

	p.Send(uint16(msg_client_message.S2CRetAllExpedition_ProtoID), res_2cli)

	return 1
}

func C2SStopExpeditionHandler(p *Player, msg_data []byte) int32 {
	var req msg_client_message.C2SStopExpedition
	err := proto.Unmarshal(msg_data, &req)
	if err != nil {
		log.Error("unmarshal msg failed %v", err.Error())
		return -1
	}

	unique_id := req.GetId()

	task_db := p.db.Expeditions.Get(unique_id)
	if nil == task_db {
		return int32(msg_client_message.E_ERR_EXPEDITION_NO_TASK)
	}

	if PLAYER_EXPEDITION_STATE_DOING != task_db.CurState {
		return int32(msg_client_message.E_ERR_EXPEDITION_STOP_NOT_DOING)
	}

	cat_ids := task_db.InCatIds
	p.db.Expeditions.Stop(unique_id)
	// update cats state
	p.send_cats_update(cat_ids)

	p.CheckUpdateExpedition()

	res_2cli := &msg_client_message.S2CRetAllExpedition{}
	p.db.Expeditions.FillAllClientMsg(res_2cli)
	cur_count, cur_cost := p.ChkDayExpeditionChgCount()
	res_2cli.CurChgCount = cur_count
	res_2cli.CurChgCost = cur_cost

	if len(res_2cli.Tasks) < 1 {
		return 0
	}

	p.Send(uint16(msg_client_message.S2CRetAllExpedition_ProtoID), res_2cli)

	return 1
}

func C2SGetExpeditionRewardHandler(p *Player, msg_data []byte) int32 {
	var req msg_client_message.C2SGetExpeditionReward
	err := proto.Unmarshal(msg_data, &req)
	if err != nil {
		log.Error("unmarshal msg failed %v", err.Error())
		return -1
	}

	p.CheckUpdateExpedition()

	unique_id := req.GetId()
	task_db := p.db.Expeditions.Get(unique_id)
	if nil == task_db {
		return int32(msg_client_message.E_ERR_EXPEDITION_NO_TASK)
	}

	task_cfg := expedition_table_mgr.Map[task_db.TaskId]
	if nil == task_cfg {
		return int32(msg_client_message.E_ERR_EXPEDITION_NOT_TASK_CFG)
	}

	cat_num := int32(len(task_db.InCatIds))
	max_cfg_len := int32(len(global_config.ExpeditionMultiVal))
	if cat_num > max_cfg_len {
		cat_num = max_cfg_len
	}
	if cat_num < 1 {
		cat_num = 1
	}

	RewardTotoalPercent := global_config.ExpeditionMultiVal[cat_num-1]

	tmp_len := int32(len(task_cfg.FixRewards))
	var obj_id, obj_num int32
	res_2cli_fi := &msg_client_message.S2CGetExpeditionReward{}
	res_2cli_fi.Id = unique_id
	res_2cli_fi.Rewards = make([]*msg_client_message.IdNum, 0, task_cfg.FixRewardsNum)
	var msg_item *msg_client_message.IdNum
	for idx := int32(0); idx+1 < tmp_len; idx = idx + 2 {
		obj_id = task_cfg.FixRewards[idx]
		obj_num = task_cfg.FixRewards[idx+1]
		if PLAYER_EXPEDITION_RESULT_FAILED == task_db.Result {
			obj_num = obj_num * RewardTotoalPercent / 200
		} else {
			obj_num = obj_num * RewardTotoalPercent / 100
		}
		if nil != item_table_mgr.Map[obj_id] {
			p.AddItem(obj_id, obj_num, "expedition_finish", "expedition", true)
		} else if nil != building_table_mgr.Map[obj_id] {
			p.AddDepotBuilding(obj_id, obj_num, "expedition_finish", "expedition", true)
		} else if nil != cat_table_mgr.Map[obj_id] {
			p.AddCat(obj_id, "expedition_finish", "expedition", true)
		} else {
			p.AddItemResource(obj_id, obj_num, "expedition_finish", "expedition")
		}
		msg_item = &msg_client_message.IdNum{Id: obj_id, Num: obj_num}
		res_2cli_fi.Rewards = append(res_2cli_fi.Rewards, msg_item)
	}

	if PLAYER_EXPEDITION_RESULT_SUCCEED == task_db.Result {

		log.Info("发放特殊事件奖励")
		tmp_len := int32(len(task_db.EventIds))
		var event_cfg *tables.XmlExpeditionEventItem
		var tmp_event *dbExpeditionEventData
		res_2cli_fi.Specials = make([]*msg_client_message.IdNum, 0, task_db.TotalSpecials)
		for idx := int32(0); idx < tmp_len; idx++ {
			tmp_event = &task_db.EventIds[idx]
			event_cfg = expedition_table_mgr.EventMap[tmp_event.EventId]
			if nil == event_cfg {
				log.Info("发放特殊事件奖励 未找到事件配置[%d]", tmp_event.EventId)
				continue
			}

			log.Info("特殊奖励配置[%v]", tmp_event.DropIdNums)
			for jdx := int32(0); jdx+1 < int32(len(tmp_event.DropIdNums)); jdx += 2 {
				obj_id = tmp_event.DropIdNums[jdx]
				obj_num = tmp_event.DropIdNums[jdx+1]
				p.AddObj(obj_id, obj_num, "expediton_event", "expeditions", true)
				log.Info("添加特殊奖励 %d %d", obj_id, obj_num)
				msg_item = &msg_client_message.IdNum{Id: obj_id, Num: obj_num}
				res_2cli_fi.Specials = append(res_2cli_fi.Specials, msg_item)
			}
		}
	}

	p.Send(uint16(msg_client_message.S2CGetExpeditionReward_ProtoID), res_2cli_fi)
	p.SendItemsUpdate()
	p.SendCatsUpdate()
	p.SendBuildingUpdate()

	cats_id, _ := p.db.Expeditions.GetInCatIds(unique_id)
	p.db.Expeditions.Remove(unique_id)
	p.send_cats_update(cats_id)
	p.CheckUpdateExpedition()
	res_2cli_tasks := &msg_client_message.S2CRetAllExpedition{}
	p.db.Expeditions.FillAllClientMsg(res_2cli_tasks)
	p.Send(uint16(msg_client_message.S2CRetAllExpedition_ProtoID), res_2cli_tasks)

	p.TaskUpdate(tables.TASK_COMPLETE_TYPE_EXPLORE_NUM, false, 0, 1)
	p.TaskUpdate(tables.TASK_COMPLETE_TYPE_EXPLORE_TASK_NUM, false, 0, 1)

	return 1
}

func C2SChgExpeditionResultHandler(p *Player, msg_data []byte) int32 {
	var req msg_client_message.C2SChgExpeditionResult
	err := proto.Unmarshal(msg_data, &req)
	if err != nil {
		log.Error("unmarshal msg failed %v", err.Error())
		return -1
	}

	unique_id := req.GetId()
	task_db := p.db.Expeditions.Get(unique_id)
	if nil == task_db {
		log.Info("C2SChgExpeditionResultHandler no task[%d] !", unique_id)
		all_tasks := p.db.Expeditions.GetAll()
		for _, val := range all_tasks {
			log.Info("	Task:[%v]", val)
		}
		return int32(msg_client_message.E_ERR_EXPEDITION_NO_TASK)
	}

	task_cfg := expedition_table_mgr.Map[task_db.TaskId]
	if nil == task_cfg {
		return int32(msg_client_message.E_ERR_EXPEDITION_NOT_TASK_CFG)
	}

	if PLAYER_EXPEDITION_STATE_FAILED != task_db.CurState {
		return int32(msg_client_message.E_ERR_EXPEDITION_WRONG_STATE)
	}

	if !p.ChkResEnough(task_cfg.BuyBackCosts) {
		return int32(msg_client_message.E_ERR_EXPEDITION_LESS_RES)
	}

	p.CheckUpdateExpedition()

	cat_num := int32(len(task_db.InCatIds))
	max_cfg_len := int32(len(global_config.ExpeditionMultiVal))
	if cat_num > max_cfg_len {
		cat_num = max_cfg_len
	}
	if cat_num < 1 {
		cat_num = 1
	}

	RewardTotoalPercent := global_config.ExpeditionMultiVal[cat_num-1]

	tmp_len := int32(len(task_cfg.FixRewards))
	var obj_id, obj_num int32
	res_2cli_fi := &msg_client_message.S2CGetExpeditionReward{}
	res_2cli_fi.Id = unique_id
	res_2cli_fi.Rewards = make([]*msg_client_message.IdNum, 0, task_cfg.FixRewardsNum)
	var msg_item *msg_client_message.IdNum
	for idx := int32(0); idx+1 < tmp_len; idx = idx + 2 {
		obj_id = task_cfg.FixRewards[idx]
		obj_num = task_cfg.FixRewards[idx+1]
		if PLAYER_EXPEDITION_RESULT_FAILED == task_db.Result {
			obj_num = obj_num * RewardTotoalPercent / 200
		}
		if nil != item_table_mgr.Map[obj_id] {
			p.AddItem(obj_id, obj_num, "expedition_finish", "expedition", true)
		} else if nil != building_table_mgr.Map[obj_id] {
			p.AddDepotBuilding(obj_id, obj_num, "expedition_finish", "expedition", true)
		} else if nil != cat_table_mgr.Map[obj_id] {
			p.AddCat(obj_id, "expedition_finish", "expedition", true)
		} else {
			p.AddItemResource(obj_id, obj_num, "expedition_finish", "expedition")
		}
		msg_item = &msg_client_message.IdNum{Id: obj_id, Num: obj_num}
		res_2cli_fi.Rewards = append(res_2cli_fi.Rewards, msg_item)
	}

	if PLAYER_EXPEDITION_RESULT_SUCCEED == task_db.Result {

		tmp_len := int32(len(task_db.EventIds))

		var event_cfg *tables.XmlExpeditionEventItem
		var tmp_event *dbExpeditionEventData
		res_2cli_fi.Specials = make([]*msg_client_message.IdNum, 0, task_db.TotalSpecials)
		for idx := int32(0); idx < tmp_len; idx++ {
			tmp_event = &task_db.EventIds[idx]
			event_cfg = expedition_table_mgr.EventMap[tmp_event.EventId]
			if nil == event_cfg {
				continue
			}

			for idx = 0; idx+1 < int32(len(tmp_event.DropIdNums)); idx += 2 {
				obj_id = tmp_event.DropIdNums[idx]
				obj_num = tmp_event.DropIdNums[idx+1]
				p.AddObj(obj_id, obj_num, "expediton_event", "expeditions", true)
				msg_item = &msg_client_message.IdNum{Id: obj_id, Num: obj_num}
				res_2cli_fi.Specials = append(res_2cli_fi.Specials, msg_item)
			}
		}
	}

	p.Send(uint16(msg_client_message.S2CGetExpeditionReward_ProtoID), res_2cli_fi)
	p.SendItemsUpdate()
	p.SendCatsUpdate()
	p.SendBuildingUpdate()

	cats_id, _ := p.db.Expeditions.GetInCatIds(unique_id)
	p.db.Expeditions.Remove(unique_id)
	p.send_cats_update(cats_id)
	p.CheckUpdateExpedition()
	res_2cli_tasks := &msg_client_message.S2CRetAllExpedition{}
	p.db.Expeditions.FillAllClientMsg(res_2cli_tasks)
	p.Send(uint16(msg_client_message.S2CRetAllExpedition_ProtoID), res_2cli_tasks)

	p.TaskUpdate(tables.TASK_COMPLETE_TYPE_EXPLORE_NUM, false, 0, 1)
	p.TaskUpdate(tables.TASK_COMPLETE_TYPE_EXPLORE_TASK_NUM, false, 0, 1)

	return 1
}
