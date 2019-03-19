package main

import (
	"mm_server/libs/log"
	"mm_server/proto/gen_go/client_message"
	"mm_server/src/tables"
	"time"
)

const (
	MAKING_BUILDING_DEFAULT_MAX_SLOT_ID = 2 // 默认槽位
	MAKING_BUILDING_MAX_SLOT_ID         = 4 // 最大槽位
)

func _make_formula_building_msg_data() *msg_client_message.MakingFormulaBuildingInfo {
	d := &msg_client_message.MakingFormulaBuildingInfo{}
	d.FormulaId = 0
	d.RemainSeconds = 0
	d.SlotId = 0
	return d
}

func (this *Player) get_formula_buildings() (slot_buildings []*msg_client_message.MakingFormulaBuildingInfo, maked_buildings []int32) {
	formula_buildings := this.db.MakingFormulaBuildings.GetAllIndex()
	var data dbPlayerMakingFormulaBuildingData
	if formula_buildings == nil || len(formula_buildings) == 0 {
		for i := 0; i < MAKING_BUILDING_DEFAULT_MAX_SLOT_ID; i++ {
			data.SlotId = int32(i + 1)
			data.FormulaId = 0
			this.db.MakingFormulaBuildings.Add(&data)
		}
		formula_buildings = this.db.MakingFormulaBuildings.GetAllIndex()
		this.db.Info.SetMakingBuildingQueue(make([]int32, 0))
	}

	slot_buildings = make([]*msg_client_message.MakingFormulaBuildingInfo, len(formula_buildings))
	for i := 0; i < len(slot_buildings); i++ {
		slot_buildings[i] = _make_formula_building_msg_data()
	}
	maked_buildings = this.db.Info.GetMakedBuildingQueue()

	// add no in making list slot id to msg
	dn := len(slot_buildings) - 1
	making_list := this.db.Info.GetMakingBuildingQueue()
	for i := 0; i < len(formula_buildings); i++ {
		j := 0
		for ; j < len(making_list); j++ {
			if formula_buildings[i] == making_list[j] {
				break
			}
		}
		if j == len(making_list) {
			slot_buildings[dn].SlotId = formula_buildings[i]
			dn -= 1
		}
	}

	var gn int32
	if making_list != nil && len(making_list) > 0 {
		now := int32(time.Now().Unix())
		start_time := int32(0)
		used_time := int32(0)
		left_seconds := int32(0)

		n := 0
		// 已经打造好的建筑和索引
		for ; n < len(making_list); n++ {
			fid, o := this.db.MakingFormulaBuildings.GetFormulaId(making_list[n])
			if !o {
				log.Warn("Player[%v] no slot[%v]", this.Id, making_list[n])
				continue
			}
			if fid <= 0 {
				log.Warn("Player[%v] making list index[%v] slot[%v] is empty", this.Id, n, making_list[n])
				break
			}

			f := formula_table_mgr.Map[fid]
			if f == nil {
				log.Warn("Player[%v] Slot[%v] FormulaId[%v] invalid", this.Id, making_list[n], fid)
				continue
			}

			if n == 0 {
				start_time, _ = this.db.MakingFormulaBuildings.GetStartTime(making_list[n])
				used_time = now - start_time
			}

			if used_time < f.Time {
				left_seconds = f.Time - used_time
				this.db.MakingFormulaBuildings.SetStartTime(making_list[n], now-used_time)
				break
			}
			used_time -= f.Time

			this.db.MakingFormulaBuildings.SetFormulaId(making_list[n], 0)
			this.db.MakingFormulaBuildings.SetStartTime(making_list[n], 0)

			slot_buildings[dn].SlotId = making_list[n]
			maked_buildings = append(maked_buildings, f.BuildID)
			dn -= 1

			this.TaskUpdate(tables.TASK_COMPLETE_TYPE_MAKING_FORMULA_NUM, false, f.Rarity, 1)
			this.TaskUpdate(tables.TASK_COMPLETE_TYPE_MAKING_FORMULA_BUILDING_NUM, false, f.Rarity, 1)
		}

		for i := n; i < len(making_list); i++ {
			fid, o := this.db.MakingFormulaBuildings.GetFormulaId(making_list[i])
			if !o {
				log.Warn("Player[%v] no slot[%v]", this.Id, making_list[i])
				continue
			}
			if fid <= 0 {
				log.Warn("Player[%v] making list index[%v] slot[%v] is empty", this.Id, i, making_list[i])
				break
			}

			f := formula_table_mgr.Map[fid]
			if f == nil {
				log.Warn("Player[%v] Slot[%v] FormulaId[%v] invalid", this.Id, making_list[i], fid)
				continue
			}

			slot_buildings[i-n].SlotId = making_list[i]
			slot_buildings[i-n].FormulaId = fid
			if i == n {
				slot_buildings[i-n].RemainSeconds = left_seconds
			} else {
				slot_buildings[i-n].RemainSeconds = f.Time * 60
			}
		}

		making_list = making_list[n:]
		this.db.Info.SetMakingBuildingQueue(making_list)
	}

	this.db.Info.SetMakedBuildingQueue(maked_buildings)

	log.Debug("slot_buildings: %v num", gn)
	for i := 0; i < len(slot_buildings); i++ {
		log.Debug("@@@@@ slot_id:%v, formula_id:%v, remain_seconds:%v", slot_buildings[i].GetSlotId(), slot_buildings[i].GetFormulaId(), slot_buildings[i].GetRemainSeconds())
	}
	log.Debug("maked_buildings: %v num", len(maked_buildings))
	for i := 0; i < len(maked_buildings); i++ {
		log.Debug("@@@@@ depot_building_id:%v", maked_buildings[i])
	}

	return slot_buildings, maked_buildings
}

// 拉取打造的建筑
func (this *Player) pull_formula_building() int32 {
	response := &msg_client_message.S2CGetMakingFormulaBuildingsResult{}
	response.Buildings, response.MakedBuildings = this.get_formula_buildings()
	this.Send(uint16(msg_client_message.S2CGetMakingFormulaBuildingsResult_ProtoID), response)
	return 1
}

// 兑换建筑配方
func (this *Player) exchange_formula(formula_id int32) int32 {
	// 解锁关卡
	f := formula_table_mgr.Map[formula_id]
	if f == nil {
		log.Error("没有建筑配方[%v]配置", formula_id)
		return int32(msg_client_message.E_ERR_FORMULA_TABLE_DATA_NOT_FOUND)
	}

	// 已有该配方
	if this.db.DepotBuildingFormulas.HasIndex(formula_id) {
		log.Warn("玩家[%v]已有该建筑配方[%v]", this.Id, formula_id)
		return 0
	}

	// 解锁关卡
	if f.UnlockChapter > this.db.Info.GetMaxChapter() {
		log.Error("玩家[%v]未解锁关卡章节[%v]", this.Id, f.UnlockChapter)
		return int32(msg_client_message.E_ERR_FORMULA_EXCHANGE_NEED_UNLOCK_CHAPTER)
	}

	// 星星数
	if this.db.Info.GetTotalStars() < f.Star {
		log.Error("玩家[%v]星星数[%v]不足，需要[%v]", this.Id, this.db.Info.GetTotalStars(), f.Star)
		return int32(msg_client_message.E_ERR_FORMULA_EXCHANGE_NOT_ENOUGH_STAR)
	}

	var data dbPlayerDepotBuildingFormulaData
	data.Id = formula_id
	this.db.DepotBuildingFormulas.Add(&data)

	this.SubStar(f.Star, "exchange_building_formula", "formula")

	this.AddExp(f.Exp, "exchange_building_formula", "formula")

	response := &msg_client_message.S2CExchangeBuildingFormulaResult{}
	response.FormulaId = formula_id
	this.Send(uint16(msg_client_message.S2CExchangeBuildingFormulaResult_ProtoID), response)

	// 公告
	if f.Rarity >= 4 {
		anouncement_mgr.PushNew(ANOUNCEMENT_TYPE_GET_FORMULA, true, this.Id, this.db.GetName(), this.db.Info.GetLvl(), formula_id, 0, 0, "")
	}

	log.Debug("Player[%v] exchanged formula[%v]", this.Id, formula_id)

	return 1
}

// 拉取配方
func (this *Player) get_formulas() int32 {
	formulas := this.db.DepotBuildingFormulas.GetAllIndex()
	if formulas == nil {
		formulas = make([]int32, 0)
	}

	response := &msg_client_message.S2CGetFormulasResult{}
	response.Formulas = formulas
	this.Send(uint16(msg_client_message.S2CGetFormulasResult_ProtoID), response)

	log.Debug("Player[%v] pull formulas %v", this.Id, formulas)
	return 1
}

// 打造
func (this *Player) make_formula_building(formula_id /*, slot_id*/ int32) int32 {

	all_idx := this.db.MakingFormulaBuildings.GetAllIndex()
	if all_idx == nil || len(all_idx) == 0 {
		return -1
	}
	idx := -1
	for i := 0; i < len(all_idx); i++ {
		d := this.db.MakingFormulaBuildings.Get(all_idx[i])
		if d == nil {
			continue
		}
		if d.FormulaId == 0 {
			idx = i
			break
		}
	}

	// 没有空槽位
	if idx < 0 {
		log.Error("Player[%v] no empty slot to making formula building[%v]", this.Id, formula_id)
		return int32(msg_client_message.E_ERR_FORMULA_NO_SLOT_TO_MAKING)
	}

	f := formula_table_mgr.Map[formula_id]
	if f == nil {
		log.Error("配方[%v]不存在", formula_id)
		return int32(msg_client_message.E_ERR_FORMULA_TABLE_DATA_NOT_FOUND)
	}

	// 是否有建筑物配方
	if !this.db.DepotBuildingFormulas.HasIndex(formula_id) {
		log.Error("玩家[%v]没有兑换该配方[%v]", this.Id, formula_id)
		return int32(msg_client_message.E_ERR_FORMULA_NOT_EXCHANGED)
	}

	// 需要消耗的货币和材料
	if this.GetGold() < f.Cost {
		log.Error("玩家[%v]货币数量[%v]不足需要[%v]，打造失败", this.Id, this.GetGold(), f.Cost)
		return int32(msg_client_message.E_ERR_FORMULA_MAKING_NOT_ENOUGH_COIN)
	}

	for i := 0; i < len(f.CostItems); i++ {
		item_id := f.CostItems[i].Id
		need_num := f.CostItems[i].Num
		num, o := this.db.Items.GetItemNum(item_id)
		if !o || num < need_num {
			if item_id == tables.ITEM_MATERIAL_ID_BOARD {
				log.Error("玩家[%v]木板数量[%v]不足需要[%v]，打造失败", this.Id, num, need_num)
			} else if item_id == tables.ITEM_MATERIAL_ID_BRICK {
				log.Error("玩家[%v]砖块数量[%v]不足需要[%v]，打造失败", this.Id, num, need_num)
			} else if item_id == tables.ITEM_MATERIAL_ID_IRON {
				log.Error("玩家[%v]生铁数量[%v]不足需要[%v]，打造失败", this.Id, num, need_num)
			} else if item_id == tables.ITEM_MATERIAL_ID_GOLD {
				log.Error("玩家[%v]金块数量[%v]不足需要[%v]，打造失败", this.Id, num, need_num)
			} else if item_id == tables.ITEM_MATERIAL_ID_LEAVES {
				log.Error("玩家[%v]叶子数量[%v]不足需要[%v]，打造失败", this.Id, num, need_num)
			} else if item_id == tables.ITEM_MATERIAL_ID_CLOTH {
				log.Error("玩家[%v]布料数量[%v]不足需要[%v]，打造失败", this.Id, num, need_num)
			} else if item_id == tables.ITEM_MATERIAL_ID_RUBBER {
				log.Error("玩家[%v]橡胶数量[%v]不足需要[%v]，打造失败", this.Id, num, need_num)
			} else if item_id == tables.ITEM_MATERIAL_ID_PAINT {
				log.Error("玩家[%v]油漆数量[%v]不足需要[%v]，打造失败", this.Id, num, need_num)
			} else {
				log.Error("")
			}
			return int32(msg_client_message.E_ERR_FORMULA_MAKING_NOT_ENOUGH_RESOURCE)
		}
	}

	slot_id := all_idx[idx]
	if slot_id < 1 || slot_id > MAKING_BUILDING_MAX_SLOT_ID {
		log.Error("玩家[%v]打造配方建筑槽位[%v]错误", this.Id, slot_id)
		return int32(msg_client_message.E_ERR_FORMULA_MAKING_SLOT_ID_INVALID)
	}

	// 槽位
	making_building := this.db.MakingFormulaBuildings.Get(slot_id)
	if making_building == nil {
		log.Error("Player[%v] making slot[%v] cant use, making failed", this.Id, slot_id)
		return int32(msg_client_message.E_ERR_FORMULA_MAKING_SLOT_ID_INVALID)
	}

	// 正在打造
	if making_building.FormulaId > 0 {
		log.Error("Player[%v] making slot[%v] already using", this.Id, slot_id)
		return int32(msg_client_message.E_ERR_FORMULA_MAKING_SLOT_IS_USING)
	}

	this.db.MakingFormulaBuildings.SetFormulaId(slot_id, formula_id)
	this.db.MakingFormulaBuildings.SetStartTime(slot_id, int32(time.Now().Unix()))
	making_list := this.db.Info.GetMakingBuildingQueue()
	making_list = append(making_list, slot_id)
	this.db.Info.SetMakingBuildingQueue(making_list)

	this.SubGold(f.Cost, "make_formula_building", "formula")
	for i := 0; i < len(f.CostItems); i++ {
		this.RemoveItem(f.CostItems[i].Id, f.CostItems[i].Num, true)
	}

	this.SendItemsUpdate()

	response := &msg_client_message.S2CMakeFormulaBuildingResult{}
	response.FormulaId = formula_id
	this.Send(uint16(msg_client_message.S2CMakeFormulaBuildingResult_ProtoID), response)

	this.pull_formula_building()

	// update task
	this.TaskUpdate(tables.TASK_COMPLETE_TYPE_MAKING_FORMULA_NUM, false, 0, 1)
	this.TaskUpdate(tables.TASK_COMPLETE_TYPE_MAKING_FORMULA_BUILDING_NUM, false, f.Rarity, 1)

	log.Debug("Player[%v] Making formula[%v] building in slot[%v], making_list[%v]", this.Id, formula_id, slot_id, making_list)

	return 1
}

// 购买空位
func (this *Player) buy_new_making_building_slot() int32 {
	all_index := this.db.MakingFormulaBuildings.GetAllIndex()
	if all_index == nil || len(all_index) == 0 {
		log.Error("Player[%v] making building data not init", this.Id)
		return -1
	}

	if this.GetDiamond() < global_config.FormulaAddNewSlotCostDiamond {
		log.Error("Player[%v] buy new making slot failed, diamond not enough", this.Id)
		return int32(msg_client_message.E_ERR_FORMULA_MAKING_SLOT_ID_INVALID)
	}

	new_slot_id := int32(0)
	for slot_id := MAKING_BUILDING_DEFAULT_MAX_SLOT_ID + 1; slot_id <= MAKING_BUILDING_MAX_SLOT_ID; slot_id++ {
		d := this.db.MakingFormulaBuildings.Get(int32(slot_id))
		if d == nil {
			var data dbPlayerMakingFormulaBuildingData
			data.SlotId = int32(slot_id)
			this.db.MakingFormulaBuildings.Add(&data)
			new_slot_id = int32(slot_id)
			break
		}
	}

	if new_slot_id == 0 {
		log.Warn("Player[%v] Making Formula building slots all used", this.Id)
		return 0
	}

	this.SubDiamond(global_config.FormulaAddNewSlotCostDiamond, "buy_new_slot", "formula")

	response := &msg_client_message.S2CBuyMakeBuildingSlotResult{}
	response.SlotId = new_slot_id
	this.Send(uint16(msg_client_message.S2CBuyMakeBuildingSlotResult_ProtoID), response)

	this.pull_formula_building()

	log.Debug("Player[%v] buy new slot[%v]", this.Id, new_slot_id)
	return 1
}

// 加速打造
func (this *Player) speedup_making_building( /*slot_id int32*/ ) int32 {
	slot_buildings, maked_buildings := this.get_formula_buildings()
	if slot_buildings == nil || len(slot_buildings) == 0 {
		log.Error("Player[%v] no formula making building to speed up", this.Id)
		return int32(msg_client_message.E_ERR_FORMULA_NO_MAKING_BUILDING)
	}

	making_list := this.db.Info.GetMakingBuildingQueue()
	cost_diamond := int32(0)
	if slot_buildings[0].GetSlotId() > 0 && slot_buildings[0].GetFormulaId() > 0 {
		remain_seconds := slot_buildings[0].GetRemainSeconds()
		this.db.MakingFormulaBuildings.SetFormulaId(slot_buildings[0].GetSlotId(), 0)
		this.db.MakingFormulaBuildings.SetStartTime(slot_buildings[0].GetSlotId(), 0)

		this.db.Info.SetMakingBuildingQueue(making_list[1:])
		formula := formula_table_mgr.Map[slot_buildings[0].GetFormulaId()]
		if formula == nil {
			log.Error("Player[%v] get formula[%v] failed, cant get maked build id", this.Id, formula.Id)
			return -1
		}
		maked_buildings = append(maked_buildings, formula.BuildID)
		this.db.Info.SetMakedBuildingQueue(maked_buildings)

		this.TaskUpdate(tables.TASK_COMPLETE_TYPE_MAKING_FORMULA_NUM, false, formula.Rarity, 1)
		this.TaskUpdate(tables.TASK_COMPLETE_TYPE_MAKING_FORMULA_BUILDING_NUM, false, formula.Rarity, 1)

		slot_buildings[0].FormulaId = 0
		slot_buildings[0].RemainSeconds = 0
		tmp_slot_id := slot_buildings[0].GetSlotId()
		n := 0
		for ; n < len(slot_buildings)-1; n++ {
			slot_buildings[n] = slot_buildings[n+1]
		}
		slot_buildings[n].SlotId = tmp_slot_id

		cost_diamond = (remain_seconds + (global_config.FormulaSpeedupMakingBuildingCostDiamond - 1)) / global_config.FormulaSpeedupMakingBuildingCostDiamond
		this.SubDiamond(cost_diamond, "speedup_making_building", "formula")
	}

	response := &msg_client_message.S2CSpeedupMakeBuildingResult{}
	response.SlotId = making_list[0]
	this.Send(uint16(msg_client_message.S2CSpeedupMakeBuildingResult_ProtoID), response)

	data_msg := &msg_client_message.S2CGetMakingFormulaBuildingsResult{}
	data_msg.Buildings = slot_buildings
	data_msg.MakedBuildings = maked_buildings
	this.Send(uint16(msg_client_message.S2CGetMakingFormulaBuildingsResult_ProtoID), data_msg)

	this.pull_formula_building()

	log.Debug("Player[%v] speed up making building in slot[%v], cost diamond %v", this.Id, cost_diamond)

	return 1
}

// 收取配方建筑
func (this *Player) get_completed_formula_building( /*slot_id int32*/ ) int32 {
	all_index := this.db.MakingFormulaBuildings.GetAllIndex()
	if all_index == nil || len(all_index) == 0 {
		log.Error("Player[%v] no making formula building slots", this.Id)
		return int32(msg_client_message.E_ERR_FORMULA_NO_MAKING_BUILDING)
	}

	maked_buildings := this.db.Info.GetMakedBuildingQueue()
	if maked_buildings == nil || len(maked_buildings) == 0 {
		return -1
	}

	this.db.Info.SetMakedBuildingQueue(make([]int32, 0))
	for i := 0; i < len(maked_buildings); i++ {
		this.AddDepotBuilding(maked_buildings[i], 1, "make_building", "formula", false)
	}
	this.SendDepotBuildingUpdate()

	response := &msg_client_message.S2CGetCompletedFormulaBuildingResult{}
	response.DepotBuildingId = maked_buildings
	this.Send(uint16(msg_client_message.S2CGetCompletedFormulaBuildingResult_ProtoID), response)

	this.pull_formula_building()

	log.Debug("Player[%v] get completed formula buildings[%v]", this.Id, maked_buildings)
	return 1
}

// 取消打造
func (this *Player) cancel_making_formula_building(slot_id int32) int32 {
	this.get_formula_buildings()
	making_list := this.db.Info.GetMakingBuildingQueue()
	if making_list == nil || len(making_list) < 2 {
		log.Error("Player[%v] no building to cancel in making list", this.Id)
		return int32(msg_client_message.E_ERR_FORMULA_NOT_MAKING)
	}

	n := 1
	for ; n < len(making_list); n++ {
		if making_list[n] == slot_id {
			break
		}
	}

	if n == len(making_list) {
		log.Error("Player[%v] no slot[%v] to making building", this.Id, slot_id)
		return int32(msg_client_message.E_ERR_FORMULA_MAKING_SLOT_ID_INVALID)
	}

	d := this.db.MakingFormulaBuildings.Get(slot_id)
	if d == nil {
		log.Error("Player[%v] no have slot[%v] for making formula building", this.Id, slot_id)
		return int32(msg_client_message.E_ERR_FORMULA_NOT_MAKING)
	}

	if d.FormulaId == 0 {
		log.Error("Player[%v] making building slot[%v] is no use", this.Id, slot_id)
		return int32(msg_client_message.E_ERR_FORMULA_NOT_MAKING)
	}

	f := formula_table_mgr.Map[d.FormulaId]
	if f == nil {
		log.Error("No formula[%v] configure", d.FormulaId)
		return int32(msg_client_message.E_ERR_FORMULA_TABLE_DATA_NOT_FOUND)
	}

	this.db.MakingFormulaBuildings.SetFormulaId(slot_id, 0)
	this.db.MakingFormulaBuildings.SetStartTime(slot_id, 0)
	for i := n; i < len(making_list)-1; i++ {
		making_list[i] = making_list[i+1]
	}
	making_list = making_list[:len(making_list)-1]
	this.db.Info.SetMakingBuildingQueue(making_list)

	/*var tmp_slot_building *msg_client_message.MakingFormulaBuildingInfo
	for i := 0; i < len(slot_buildings)-1; i++ {
		if slot_id == slot_buildings[i].GetSlotId() {
			tmp_slot_building = slot_buildings[i]
			n = i
			log.Debug("!!!!!!!!@@@@@@@ n:%v  tmp_slot_building:%v", n, *tmp_slot_building)
			break
		}
	}

	if tmp_slot_building == nil {
		log.Error("Player[%v] making building data is invalid", this.Id)
		return -1
	}

	for i := n; i < len(slot_buildings)-1; i++ {
		slot_buildings[i] = slot_buildings[i+1]
		log.Debug("@@@@@@@@!!!!!!!! slot_buildings[%v] slot_id[%v] formula_id[%v]", i, slot_buildings[i].GetSlotId(), slot_buildings[i].GetFormulaId())
	}
	tmp_slot_building.FormulaId = proto.Int32(0)
	tmp_slot_building.RemainSeconds = proto.Int32(0)
	slot_buildings[len(slot_buildings)-1] = tmp_slot_building*/

	response := &msg_client_message.S2CCancelMakingFormulaBuildingResult{}
	response.SlotId = slot_id
	response.ReturnMaterials = make([]*msg_client_message.ItemInfo, len(f.CostItems))
	for i := 0; i < len(f.CostItems); i++ {
		response.ReturnMaterials[i] = &msg_client_message.ItemInfo{}
		response.ReturnMaterials[i].ItemCfgId = f.CostItems[i].Id
		item_num := int32(f.CostItems[i].Num * global_config.CancelMakingFormulaReturnMaterial / 100)
		response.ReturnMaterials[i].ItemNum = item_num

		this.AddItem(f.CostItems[i].Id, item_num, "cancel_making_formula_building", "formula", true)
	}
	this.Send(uint16(msg_client_message.S2CCancelMakingFormulaBuildingResult_ProtoID), response)
	this.SendItemsUpdate()

	this.pull_formula_building()

	log.Debug("Player[%v] cancel making formula building in slot[%v], making_list[%v]", this.Id, slot_id, making_list)

	return 1
}
