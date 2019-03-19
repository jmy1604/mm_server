package main

import (
	"mm_server/libs/log"
	"mm_server/proto/gen_go/client_message"
	"mm_server/src/rpc_proto"
	"mm_server/src/tables"
	"time"
)

func (this *dbPlayerCatHouseColumn) Get4RPC(building_id int32) (data *rpc_proto.H2H_CatHouseData) {
	this.m_row.m_lock.UnSafeRLock("dbPlayerCatHouseColumn.Get4RPC")
	defer this.m_row.m_lock.UnSafeRUnlock()

	d := this.m_data[building_id]
	if d == nil {
		return nil
	}

	is_done := false
	if d.IsDone != 0 {
		is_done = true
	}

	var cat_ids []int32
	if d.CatIds == nil || len(d.CatIds) == 0 {
		cat_ids = make([]int32, 0)
	} else {
		cat_ids = make([]int32, len(d.CatIds))
		copy(cat_ids, d.CatIds)
	}
	data = &rpc_proto.H2H_CatHouseData{
		CatIds:        cat_ids,
		CatHouseLevel: d.Level,
		IsDone:        is_done,
	}
	return
}

func get_cathouse_table_data(cathouse_cid, level int32) *tables.XmlCatHouseItem {
	cathouse := cathouse_table_mgr.Map[cathouse_cid]
	if cathouse == nil {
		log.Error("Cathouse[%v] table data not found", cathouse_cid)
		return nil
	}
	if level < 1 || level > int32(len(cathouse)) {
		log.Error("Cathouse[%v] table no level[%v] data", cathouse_cid, level)
		return nil
	}

	return cathouse[level-1]
}

func (this *Player) get_cathouse_curr_gold(building_id int32) int32 {

	now_time := int32(time.Now().Unix())
	last_gold_time, o := this.db.CatHouses.GetLastGetGoldTime(building_id)
	if !o {
		return -1
	}
	level, _ := this.db.CatHouses.GetLevel(building_id)
	if level == 0 {
		return 0
	}
	cathouse_cid, _ := this.db.CatHouses.GetCfgId(building_id)
	cathouse := get_cathouse_table_data(cathouse_cid, level)
	if cathouse == nil {
		return -1
	}
	gold, _ := this.db.CatHouses.GetCurrGold(building_id)
	if gold >= cathouse.CoinStorage {
		return cathouse.CoinStorage
	}

	max_add := cathouse.CoinStorage - gold
	cost_seconds := int32(0)
	if last_gold_time > 0 {
		cost_seconds = now_time - last_gold_time
	}

	cat_ids, _ := this.db.CatHouses.GetCatIds(building_id)
	v := int32(0)
	for _, cid := range cat_ids {
		cat_cid, o := this.db.Cats.GetCfgId(cid)
		if !o {
			log.Warn("Player[%v] have not cat[%v]", cid)
			continue
		}

		cat := cat_table_mgr.GetCat(cat_cid)
		if cat == nil {
			log.Error("Cat[%v] table data not found", cid)
			continue
		}

		level, _ := this.db.Cats.GetLevel(cid)
		coin_ability, _ := this.db.Cats.GetCoinAbility(cid)
		star, _ := this.db.Cats.GetStar(cid)
		t := int32(cat.GrowthRate*level/100 + coin_ability + coin_ability*cat.InitialRate*(level-1)/100 + cat.AddCoins[star-1])
		v += t
		log.Debug("!!!!! player[%v] cat[%v] level[%v] growth_rate[%v] initial_rate[%v] coin_ability[%v,%v] cost_seconds[%v]", this.Id, cid, level, cat.GrowthRate, cat.InitialRate, coin_ability, t, cost_seconds)
	}
	add_gold := int32(cost_seconds * v / 60)
	if add_gold > max_add {
		add_gold = max_add
	}
	curr_gold := this.db.CatHouses.IncbyCurrGold(building_id, add_gold)
	this.db.CatHouses.SetLastGetGoldTime(building_id, now_time)
	return curr_gold
}

func (this *Player) update_cathouse_level(building_id int32) (bool, int32, int32) {
	levelup_starttime, o := this.db.CatHouses.GetLevelupStartTime(building_id)
	if !o {
		return false, 0, 0
	}

	level, _ := this.db.CatHouses.GetLevel(building_id)
	if levelup_starttime <= 0 {
		log.Info("Player[%v] cathouse[%v] not doing level up", this.Id, building_id)
		return false, level, 0
	}

	cathouse_cid, _ := this.db.CatHouses.GetCfgId(building_id)
	cathouse := get_cathouse_table_data(cathouse_cid, level+1)
	if cathouse == nil {
		log.Error("Cathouse[%v] table data not found", cathouse_cid)
		return false, level, 0
	}

	if levelup_starttime > 0 {
		now_time := int32(time.Now().Unix())
		cost_time := now_time - levelup_starttime
		if cost_time < cathouse.Time {
			return false, level, cathouse.Time - cost_time
		}

		level += 1
		this.db.CatHouses.SetLevel(building_id, level)
		this.db.CatHouses.SetLevelupStartTime(building_id, 0)
		//this.db.CatHouses.SetIsDone(building_id, 0)
	}

	return true, level, 0
}

func (this *Player) if_cathouse_can_create(cathouse_cid int32) bool {
	building := building_table_mgr.Map[cathouse_cid]
	if building == nil {
		return false
	}

	cathouse := get_cathouse_table_data(cathouse_cid, 1)
	if cathouse == nil {
		return false
	}

	if cathouse.UnlockStar > this.db.Info.GetTotalStars() {
		return false
	}

	return true
}

func (this *Player) send_cathouse_info(building_id int32, on_update bool) {
	cathouse_data := this.db.CatHouses.Get(building_id)
	if cathouse_data == nil {
		return
	}

	level, _ := this.db.CatHouses.GetLevel(building_id)
	remain_seconds := int32(0)
	curr_gold, _ := this.db.CatHouses.GetCurrGold(building_id)
	if on_update {
		curr_gold = this.get_cathouse_curr_gold(building_id)
		_, level, remain_seconds = this.update_cathouse_level(building_id)
	}
	is_done, _ := this.db.CatHouses.GetIsDone(building_id)
	cathouse_cid, _ := this.db.CatHouses.GetCfgId(building_id)

	msg := &msg_client_message.S2CGetCatHouseInfoResult{}
	msg.House = &msg_client_message.CatHouseInfo{}
	msg.House.CatHouseId = building_id
	msg.House.CatIds, _ = this.db.CatHouses.GetCatIds(building_id)
	msg.House.Level = level
	msg.House.Gold = curr_gold
	msg.House.NextLevelRemainSeconds = remain_seconds
	msg.House.BuildingConfigId = cathouse_cid
	msg.House.IsDone = is_done

	log.Debug("Player[%v] get cathouse[%v] info: cat_ids[%v] level[%v] gold[%v] is_done[%v]", this.Id, building_id, msg.House.GetCatIds(), level, curr_gold, is_done)

	this.Send(uint16(msg_client_message.S2CGetCatHouseInfoResult_ProtoID), msg)
}

func (this *Player) create_cathouse(building_id int32) *dbPlayerCatHouseData {
	building_data := this.db.Buildings.Get(building_id)
	if building_data == nil {
		log.Error("Player[%v] have not building[%v]", this.Id, building_id)
		return nil
	}
	cathouse_data := this.db.CatHouses.Get(building_id)
	if cathouse_data == nil {
		// 创建猫舍
		var data dbPlayerCatHouseData
		data.BuildingId = building_id
		data.CfgId = building_data.CfgId
		data.Level = 0
		data.CatIds = make([]int32, 0)
		data.CurrGold = 0
		data.LastGetGoldTime = int32(time.Now().Unix())
		data.IsDone = 1
		this.db.CatHouses.Add(&data)
		cathouse_data = this.db.CatHouses.Get(building_id)
		//this.send_cathouse_info(building_id, false)
		if this.cathouse_start_levelup(building_id, false) < 0 {
			return nil
		}
		//this.db.CatHouses.SetIsDone(building_id, 0)
	}
	this.send_cathouse_info(building_id, true)
	return cathouse_data
}

func (this *Player) remove_cathouse(building_id int32) bool {
	if !this.db.CatHouses.HasIndex(building_id) {
		return false
	}
	this.db.CatHouses.Remove(building_id)
	return true
}

func (this *Player) cathouse_add_cat(cat_id int32, building_id int32) int32 {
	if !this.db.Cats.HasIndex(cat_id) {
		log.Error("Player[%v] cat[%] not found", this.Id, cat_id)
		return int32(msg_client_message.E_ERR_CAT_NOT_FOUND)
	}

	cathouse_id, _ := this.db.Cats.GetCathouseId(cat_id)
	if cathouse_id > 0 {
		log.Error("Player[%v] cat[%v] is in cathouse[%v]", this.Id, cat_id, cathouse_id)
		return int32(msg_client_message.E_ERR_CATHOUSE_CAT_ALREADY_IN_HOUSE)
	}

	cat_cid, _ := this.db.Cats.GetCfgId(cat_id)
	cat := cat_table_mgr.Map[cat_cid]
	if cat == nil {
		log.Error("Cat[%v] table data not found", cat_cid)
		return int32(msg_client_message.E_ERR_CAT_TABLE_DATA_NOT_FOUND)
	}

	cathouse_data := this.create_cathouse(building_id)
	if cathouse_data == nil {
		return int32(msg_client_message.E_ERR_CATHOUSE_NOT_FOUND)
	}

	cathouse_cid, _ := this.db.CatHouses.GetCfgId(building_id)
	cathouse_level, _ := this.db.CatHouses.GetLevel(building_id)
	cathouse := get_cathouse_table_data(cathouse_cid, cathouse_level)
	if cathouse == nil {
		return int32(msg_client_message.E_ERR_CATHOUSE_TABLE_DATA_NOT_FOUND)
	}

	cat_ids, _ := this.db.CatHouses.GetCatIds(building_id)
	if cathouse.CatStorage <= int32(len(cat_ids)) {
		log.Error("Player[%v] Cathouse[%v] storage[%v] is full", this.Id, building_id, cathouse.CatStorage)
		return int32(msg_client_message.E_ERR_CATHOUSE_IS_FULL)
	}

	if (cathouse.Color & cat.Color) == 0 {
		log.Error("Player[%v] cat color[%v] not same to cat house[%v]", this.Id, cat.Color, cathouse.Color)
		return int32(msg_client_message.E_ERR_CATHOUSE_CAT_MUST_SAME_COLOR)
	}

	// 是否已有该猫
	for _, cid := range cat_ids {
		if cid == cat_id {
			log.Error("Player[%v] Cathouse[%v] already have cat[%v]", this.Id, building_id, cat_id)
			return int32(msg_client_message.E_ERR_CATHOUSE_CAT_ALREADY_IN_HOUSE)
		}
	}

	curr_gold := this.get_cathouse_curr_gold(building_id)
	//this.AddCoin(curr_gold, "add_cat", "cathouse")
	//this.db.CatHouses.SetCurrGold(building_id, 0)

	cat_ids = append(cat_ids, cat_id)
	this.db.CatHouses.SetCatIds(building_id, cat_ids)
	this.db.Cats.SetCathouseId(cat_id, building_id)

	response := &msg_client_message.S2CCatHouseAddCatResult{}
	response.CatHouseId = building_id
	response.CatId = cat_id
	response.Gold = curr_gold
	this.Send(uint16(msg_client_message.S2CCatHouseAddCatResult_ProtoID), response)

	this.send_cathouse_info(building_id, false)
	this.item_cat_building_change_info.cat_update(this, cat_id)
	this.SendCatsUpdate()

	log.Debug("Player[%v] add cat[%v] to cathouse[%v], curr gold[%v], cat_ids[%v]", this.Id, cat_id, building_id, curr_gold, cat_ids)

	return 1
}

func (this *Player) get_cathouses_info() int32 {
	response := &msg_client_message.S2CGetCatHousesInfoResult{}
	all_index := this.db.CatHouses.GetAllIndex()
	if all_index == nil || len(all_index) == 0 {
		response.Houses = make([]*msg_client_message.CatHouseInfo, 0)
	} else {
		response.Houses = make([]*msg_client_message.CatHouseInfo, len(all_index))
		c := int32(0)
		for _, id := range all_index {
			// 是否有该建筑
			if !this.db.Buildings.HasIndex(id) {
				this.db.CatHouses.Remove(id)
				log.Error("Player[%v] not found building[%v] for cathouse", this.Id, id)
				continue
			}
			cat_ids, o := this.db.CatHouses.GetCatIds(id)
			if !o {
				log.Error("Player[%v] not found cathouse[%v]", this.Id, id)
				continue
			}
			n := 0
			for i := 0; i < len(cat_ids); i++ {
				if this.db.Cats.HasIndex(cat_ids[i]) {
					cat_ids[n] = cat_ids[i]
					n += 1
				} else {
					log.Error("Player[%v] cathouse[%v] cat[%v] not found", this.Id, id, cat_ids[i])
				}
			}
			if n < len(cat_ids) {
				cat_ids = cat_ids[:n]
				this.db.CatHouses.SetCatIds(id, cat_ids)
			}
			cathouse_cid, _ := this.db.CatHouses.GetCfgId(id)

			response.Houses[c] = &msg_client_message.CatHouseInfo{}
			response.Houses[c].CatHouseId = id
			response.Houses[c].CatIds = cat_ids
			curr_gold := this.get_cathouse_curr_gold(id)
			_, level, remain_seconds := this.update_cathouse_level(id)
			response.Houses[c].Level = level
			response.Houses[c].Gold = curr_gold
			response.Houses[c].NextLevelRemainSeconds = remain_seconds
			response.Houses[c].BuildingConfigId = cathouse_cid
			is_done, _ := this.db.CatHouses.GetIsDone(id)
			response.Houses[c].IsDone = is_done
			c += 1
			log.Debug("@@@@ Player[%v] cathouse[%v] cat_ids[%v] level[%v] gold[%v] is_done[%v]", this.Id, id, cat_ids, level, curr_gold, is_done)
		}
		response.Houses = response.Houses[:c]
	}
	this.Send(uint16(msg_client_message.S2CGetCatHouseInfoResult_ProtoID), response)
	return 1
}

func (this *Player) cathouse_remove_cat(cat_id, building_id int32) int32 {
	cathouse_data := this.db.CatHouses.Get(building_id)
	if cathouse_data == nil {
		log.Error("Player[%v] not have cathouse[%v]", this.Id, building_id)
		return int32(msg_client_message.E_ERR_CATHOUSE_NOT_FOUND)
	}

	if cathouse_data.IsDone == 0 {
		//this.get_cathouse_info(building_id)
		log.Error("Player[%v] cathouse[%v] not set done", this.Id, building_id)
		return -1
	}

	cat_data := this.db.Cats.Get(cat_id)
	if cat_data == nil {
		log.Error("Player[%v] Cat[%v] not found", this.Id, cat_id)
		return int32(msg_client_message.E_ERR_CAT_NOT_FOUND)
	}

	if cat_data.CathouseId != building_id {
		log.Error("Player[%v] cat[%v] not in cathouse[%v], but in cathouse[%v]", this.Id, cat_id, building_id, cat_data.CathouseId)
		return int32(msg_client_message.E_ERR_CATHOUSE_CAT_NOT_IN_THE_HOUSE)
	}

	found := false
	i := 0
	for ; i < len(cathouse_data.CatIds); i++ {
		if found {
			cathouse_data.CatIds[i-1] = cathouse_data.CatIds[i]
			continue
		}

		cid := cathouse_data.CatIds[i]
		if cat_id == cid {
			found = true
		}
	}

	if !found {
		log.Error("Player[%v] cat house[%v] not have cat[%v]", this.Id, building_id, cat_id)
		return int32(msg_client_message.E_ERR_CATHOUSE_NOT_FOUND_CAT_IN_HOUSE)
	}

	gold := this.get_cathouse_curr_gold(building_id)
	//this.AddCoin(gold, "remove_cat", "cathouse")
	//this.db.CatHouses.SetCurrGold(building_id, 0)
	cats_num := int32(len(cathouse_data.CatIds) - 1)
	this.db.CatHouses.SetCatIds(building_id, cathouse_data.CatIds[:cats_num])
	this.db.Cats.SetCathouseId(cat_id, 0)

	this.item_cat_building_change_info.cat_update(this, cat_id)
	this.SendCatsUpdate()

	response := &msg_client_message.S2CCatHouseRemoveCatResult{}
	response.CatHouseId = building_id
	response.CatId = cat_id
	response.Gold = gold
	this.Send(uint16(msg_client_message.S2CCatHouseRemoveCatResult_ProtoID), response)

	this.send_cathouse_info(building_id, false)

	log.Debug("Player[%v] removed cat[%v] from cathouse[%v], cat_ids[%v]", this.Id, cat_id, building_id, cathouse_data.CatIds[:cats_num])

	return 1
}

func (this *Player) cathouse_collect_gold(building_id int32) int32 {
	is_done, o := this.db.CatHouses.GetIsDone(building_id)
	if !o {
		log.Error("Player[%v] cathouse[%v] not found", this.Id, building_id)
		return int32(msg_client_message.E_ERR_CATHOUSE_NOT_FOUND)
	}

	if is_done == 0 {
		//this.get_cathouse_info(building_id)
		log.Error("Player[%v] cathouse[%v] not set done", this.Id, building_id)
		return -1
	}

	gold := this.get_cathouse_curr_gold(building_id)
	this.db.CatHouses.SetCurrGold(building_id, 0)
	this.AddGold(gold, "cathouse_collect_gold", "cathouse")

	response := &msg_client_message.S2CCatHouseGetGoldResult{}
	response.CatHouseId = building_id
	response.Gold = gold
	this.Send(uint16(msg_client_message.S2CCatHouseGetGoldResult_ProtoID), response)

	this.send_cathouse_info(building_id, false)

	log.Debug("Player[%v] collect cathouse[%v] gold[%v]", this.Id, building_id, gold)
	return gold
}

func (this *Player) cathouses_collect_gold(building_ids []int32) int32 {
	gold := int32(0)

	for i := 0; i < len(building_ids); i++ {
		res := this.cathouse_collect_gold(building_ids[i])
		if res < 0 {
			return res
		}
		gold += res
	}
	response := &msg_client_message.S2CCatHousesGetGoldResult{}
	response.CatHouseIds = building_ids
	response.Gold = gold
	this.Send(uint16(msg_client_message.S2CCatHousesGetGoldResult_ProtoID), response)
	return 1
}

func (this *Player) cathouse_start_levelup(building_id int32, send_cathouse_info bool) int32 {
	is_done, o := this.db.CatHouses.GetIsDone(building_id)
	if !o {
		log.Error("Player[%v] cathouse[%v] not found", this.Id, building_id)
		return int32(msg_client_message.E_ERR_CATHOUSE_NOT_FOUND)
	}

	if is_done == 0 {
		//this.get_cathouse_info(building_id)
		log.Error("Player[%v] cathouse[%v] already set done", this.Id, building_id)
		return -1
	}

	level, _ := this.db.CatHouses.GetLevel(building_id)
	levelup_starttime, _ := this.db.CatHouses.GetLevelupStartTime(building_id)
	if levelup_starttime > 0 {
		log.Error("Player[%v] is doing up level[%v] to next level[%v]", this.Id, level, level+1)
		return int32(msg_client_message.E_ERR_CATHOUSE_IS_DOING_LEVEL_UP)
	}

	cathouse_cid, _ := this.db.CatHouses.GetCfgId(building_id)
	cathouse := cathouse_table_mgr.Map[cathouse_cid]
	if cathouse == nil {
		log.Error("Cathouse[%v] table data not found", cathouse_cid)
		return int32(msg_client_message.E_ERR_CATHOUSE_TABLE_DATA_NOT_FOUND)
	}

	if level >= int32(len(cathouse)) {
		log.Warn("Player[%v] cathouse[%v] level[%v] is max", this.Id, building_id, level)
		return int32(msg_client_message.E_ERR_CATHOUSE_LEVEL_IS_MAX)
	}

	next_level := level + 1
	if this.GetGold() < cathouse[next_level-1].Cost {
		log.Error("Player[%v] levelup cathouse[%v] failed, coin[%v] not enough, need[%v]", this.Id, building_id, this.GetGold(), cathouse[next_level-1].Cost)
		return int32(msg_client_message.E_ERR_CATHOUSE_LEVELUP_COST_NOT_ENOUGH)
	}

	//this.db.CatHouses.SetLevel(building_id, next_level)
	this.db.CatHouses.SetLevelupStartTime(building_id, int32(time.Now().Unix()))
	this.db.CatHouses.SetIsDone(building_id, 0)
	this.SubGold(cathouse[next_level-1].Cost, "cathouse_start_levelup", "cathouse")

	response := &msg_client_message.S2CCatHouseStartLevelupResult{}
	response.CatHouseId = building_id
	response.RemainSeconds = cathouse[next_level-1].Time
	this.Send(uint16(msg_client_message.S2CCatHouseStartLevelupResult_ProtoID), response)

	if send_cathouse_info {
		this.send_cathouse_info(building_id, true)
	}

	log.Debug("Player[%v] cathouse[%v] level up to [%v]", this.Id, building_id, next_level)

	return 1
}

func (this *Player) cathouse_speed_levelup(building_id int32) int32 {
	/*is_done, o := this.db.CatHouses.GetIsDone(building_id)
	if !o {
		log.Error("Player[%v] cathouse[%v] not exist", this.Id, building_id)
		return int32(msg_client_message.E_ERR_CATHOUSE_NOT_FOUND)
	}

	if is_done == 0 {
		//this.get_cathouse_info(building_id)
		log.Error("Player[%v] cathouse[%v] not set done", this.Id, building_id)
		return -1
	}*/

	levelup_starttime, _ := this.db.CatHouses.GetLevelupStartTime(building_id)
	if levelup_starttime <= 0 {
		log.Error("Player[%v] cathouse[%v] is not doing levelup", this.Id, building_id)
		return int32(msg_client_message.E_ERR_CATHOUSE_NOT_DOING_LEVELUP)
	}

	level, _ := this.db.CatHouses.GetLevel(building_id)
	next_level := level + 1
	cathouse_cid, _ := this.db.CatHouses.GetCfgId(building_id)
	cathouse := get_cathouse_table_data(cathouse_cid, next_level)
	remain_seconds := GetRemainSeconds(levelup_starttime, cathouse.Time)
	if remain_seconds == 0 {
		log.Warn("Player[%v] cathouse[%v] already leveled up to [%v]", this.Id, building_id, next_level)
		return 1
	}

	cost_diamond := (remain_seconds + (global_config.CatHouseSpeedupLevelCostDiamond - 1)) / global_config.CatHouseSpeedupLevelCostDiamond

	if this.GetDiamond() < cost_diamond {
		log.Error("Player[%v] cathouse[%v] speedup level up not enough diamond, need[%v]", this.Id, building_id, cost_diamond)
		return int32(msg_client_message.E_ERR_CATHOUSE_SPEEDUP_LEVELUP_NOT_ENOUGH_DIAMOND)
	}

	// 更新当前猫舍金币
	this.get_cathouse_curr_gold(building_id)
	//this.AddCoin(gold, "speedup_levelup", "cathouse")
	//this.db.CatHouses.SetCurrGold(building_id, 0)

	this.db.CatHouses.SetLevel(building_id, next_level)
	this.db.CatHouses.SetLevelupStartTime(building_id, 0)
	this.db.CatHouses.SetIsDone(building_id, 0)

	this.SubDiamond(cost_diamond, "cathouse_speed_levelup", "cathouse")

	response := &msg_client_message.S2CCatHouseSpeedLevelupResult{}
	response.CatHouseId = building_id
	response.CostDiamond = cost_diamond
	this.Send(uint16(msg_client_message.S2CCatHouseSpeedLevelupResult_ProtoID), response)

	this.send_cathouse_info(building_id, false)

	log.Debug("Player[%v] cathouse[%v] cost diamond[%v] to speed up to level[%v]", this.Id, building_id, cost_diamond, next_level)

	return 1
}

func (this *Player) cathouse_can_remove(building_id int32) int32 {
	cathouse_data := this.db.CatHouses.Get(building_id)
	if cathouse_data == nil {
		log.Error("Player[%v] cathouse[%v] not exist", this.Id, building_id)
		return int32(msg_client_message.E_ERR_CATHOUSE_NOT_FOUND)
	}

	if len(cathouse_data.CatIds) > 0 {
		log.Error("Player[%v] cathouse[%v] has cat in, cant sell", this.Id, building_id)
		return int32(msg_client_message.E_ERR_CATHOUSE_CANT_SELL)
	}

	return 1
}

func (this *Player) cathouse_remove(building_id int32, is_sell bool) int32 {
	cathouse_cid, o := this.db.CatHouses.GetCfgId(building_id)
	if !o {
		log.Error("Player[%v] cathouse[%v] not exist", this.Id, building_id)
		return int32(msg_client_message.E_ERR_CATHOUSE_NOT_FOUND)
	}
	level, _ := this.db.CatHouses.GetLevel(building_id)
	cathouse := get_cathouse_table_data(cathouse_cid, level)
	if cathouse == nil {
		log.Error("Cathouse[%v] table data[%v] not found", building_id, cathouse_cid)
		return int32(msg_client_message.E_ERR_CATHOUSE_TABLE_DATA_NOT_FOUND)
	}
	if is_sell {
		add_gold := this.get_cathouse_curr_gold(building_id)
		this.AddGold(cathouse.SalePrice+add_gold, "cathouse_sell", "cathouse")
	}
	this.db.CatHouses.Remove(building_id)
	return 1
}

func (this *Player) cathouse_setdone(building_id int32) int32 {
	this.update_cathouse_level(building_id)
	is_done, o := this.db.CatHouses.GetIsDone(building_id)
	if !o {
		log.Error("Player[%v] cathouse[%v] not exist", this.Id, building_id)
		return int32(msg_client_message.E_ERR_CATHOUSE_NOT_FOUND)
	}

	if is_done > 0 {
		log.Warn("Player[%v] cathouse[%v] is set done yet", this.Id, building_id)
	}

	this.db.CatHouses.SetIsDone(building_id, 1)
	response := &msg_client_message.S2CCatHouseSetDoneResult{}
	response.IsDone = 1
	response.CatHouseId = building_id
	this.Send(uint16(msg_client_message.S2CCatHouseSetDoneResult_ProtoID), response)

	this.send_cathouse_info(building_id, false)

	return 1
}
