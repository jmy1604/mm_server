package main

import (
	"math/rand"
	"mm_server/libs/log"
	"mm_server/src/common"
	"mm_server/src/tables"

	"mm_server/proto/gen_go/client_message"
)

const (
	CAT_STATE_NONE          = iota
	CAT_STATE_IN_CATHOUSE   // 猫舍
	CAT_STATE_IN_EXPEDITION // 探索
	CAT_STATE_IN_FOSTER     // 寄养所
)

func is_cat_level_max(next_level int32, cat *tables.XmlCharacterItem) bool {
	if int(next_level) >= len(cat.UpgradeExps) {
		return true
	}
	return false
}

func cat_level_need_exp(level int32, cat *tables.XmlCharacterItem) int32 {
	return cat.GetLevelExp(level)
}

func (this *Player) IfCatBusy(cat_id int32) bool {
	if this.db.Expeditions.IfCatInExpedition(cat_id) {
		return true
	}

	if house, o := this.db.Cats.GetCathouseId(cat_id); o && house > 0 {
		return true
	}

	if this.db.FosterCats.HasIndex(cat_id) {
		return true
	}

	return false
}

func (this *Player) GetCatState(cat_id int32) int32 {
	state := CAT_STATE_NONE
	if this.db.Expeditions.IfCatInExpedition(cat_id) {
		state = CAT_STATE_IN_EXPEDITION
	} else if this.db.FosterCats.HasIndex(cat_id) {
		state = CAT_STATE_IN_FOSTER
	} else {
		house, o := this.db.Cats.GetCathouseId(cat_id)
		if o && house > 0 {
			state = CAT_STATE_IN_CATHOUSE
		}
	}
	return int32(state)
}

func (this *Player) cat_level_up(next_level int32, cat_id int32, cat *tables.XmlCharacterItem, curr_food int32) (ok bool, cost_food int32) {
	need_exp := cat.GetLevelExp(next_level)
	if need_exp < 0 {
		log.Error("玩家[%v]猫[%v]的下一等级[%v]不合法", this.Id, cat_id, next_level)
		return false, 0
	}

	if need_exp > curr_food {
		log.Debug("玩家[%v]猫[%v]不能升级到下一级[%v]，猫粮[%v]不够，需要[%v]", this.Id, cat_id, next_level, curr_food, need_exp)
		return false, 0
	}

	return true, need_exp
}

func (this *Player) feed_need_food(cat_id int32) (need_food int32, add_exp int32, is_critical bool) {
	cat_cid, o := this.db.Cats.GetCfgId(cat_id)
	if !o {
		return -1, -1, false
	}

	cat := cat_table_mgr.Map[cat_cid]
	if cat == nil {
		return -1, -1, false
	}

	// 受星级限制
	cat_star, _ := this.db.Cats.GetStar(cat_id)
	curr_level, _ := this.db.Cats.GetLevel(cat_id)
	max_level := cat.GetStarMaxLevel(cat_star)
	if curr_level >= max_level {
		return int32(msg_client_message.E_ERR_CAT_UPLEVL_NEED_UPSTAR), 0, false
	}

	feed_cost := cat.FeedCosts[curr_level-1]
	// 猫粮不够
	if feed_cost > this.db.Info.GetCatFood() {
		return int32(msg_client_message.E_ERR_CAT_FOOD_NOT_ENOUGH), 0, false
	}

	if cat.CriticalChances[curr_level] > rand.Int31n(10000) {
		is_critical = true
	}

	curr_exp, _ := this.db.Cats.GetExp(cat_id)
	next_level_need_exp := cat_level_need_exp(curr_level, cat)
	if is_critical || feed_cost > next_level_need_exp-curr_exp {
		if feed_cost >= next_level_need_exp-curr_exp {
			is_critical = false
		}
		add_exp = next_level_need_exp - curr_exp
	} else {
		add_exp = feed_cost
	}

	log.Debug("Player[%v] feed cat[%v] need food[%v] add_exp[%v] is_critical[%v]", this.Id, cat_id, feed_cost, add_exp, is_critical)
	return feed_cost, add_exp, is_critical
}

func (this *Player) feed_cat(cat_id int32, food int32, add_exp int32, is_critical bool) (int32, int32, int32) {
	cat_cid, o := this.db.Cats.GetCfgId(cat_id)
	if !o {
		log.Error("玩家[%v]没有猫实例[%v]", this.Id, cat_id)
		return 0, 0, -1
	}

	cat := cat_table_mgr.Map[cat_cid]
	if cat == nil {
		log.Error("玩家[%v]没有猫[%v]配置", this.Id, cat_cid)
		return 0, 0, int32(msg_client_message.E_ERR_CAT_TABLE_DATA_NOT_FOUND)
	}

	if food < 0 || food > this.db.Info.GetCatFood() {
		log.Error("玩家[%v]猫粮[%v]不够", this.Id, this.db.Info.GetCatFood())
		return 0, 0, int32(msg_client_message.E_ERR_CAT_FOOD_NOT_ENOUGH)
	}

	curr_level, _ := this.db.Cats.GetLevel(cat_id)
	curr_exp, _ := this.db.Cats.GetExp(cat_id)

	old_level := curr_level
	old_exp := curr_exp
	cat_star, _ := this.db.Cats.GetStar(cat_id)
	max_level := cat.GetStarMaxLevel(cat_star)

	total_cost_exp := int32(0)
	is_max_level := false
	limited_bystar := false
	for {
		if is_cat_level_max(curr_level, cat) {
			is_max_level = true
			log.Info("玩家[%v]猫[%v]已升到最高级[%v]", this.Id, cat_id, curr_level)
			break
		}
		if curr_level+1 > max_level {
			// 受星级限制
			limited_bystar = true
			log.Info("Player[%v] cat[%v] level up limited by star[%v], need up star", this.Id, cat_id, cat_star)
			break
		}
		o, cost_exp := this.cat_level_up(curr_level, cat_id, cat, curr_exp+add_exp-total_cost_exp)
		if !o {
			break
		}
		curr_level += 1
		total_cost_exp += cost_exp
	}

	if is_max_level {
		curr_exp = 0
	} else {
		curr_exp = curr_exp + add_exp - total_cost_exp
		if limited_bystar {
			next_level_need_exp := cat_level_need_exp(curr_level, cat)
			if curr_exp >= next_level_need_exp {
				total_cost_exp += (next_level_need_exp - 1)
				curr_exp = next_level_need_exp - 1
			}
		}
	}

	if curr_level == old_level && curr_exp == old_exp {
		if limited_bystar {
			return 0, 0, int32(msg_client_message.E_ERR_CAT_UPLEVL_NEED_UPSTAR)
		}
		log.Debug("玩家[%v]猫[%v]升级没有变化", this.Id, cat_id)
		return 0, 0, 0
	}

	if curr_level != old_level {
		this.db.Cats.SetLevel(cat_id, curr_level)
		log.Info("玩家[%v]猫[%v]升级到[%v]", this.Id, cat_id, curr_level)
	}
	if curr_exp != old_exp {
		this.db.Cats.SetExp(cat_id, curr_exp)
		log.Info("玩家[%v]猫[%v]当前经验[%v]", this.Id, cat_id, curr_exp)
	}

	this.SubCatFood(food, "cat_level_up", "cat")

	this.item_cat_building_change_info.cat_update(this, cat_id)
	this.item_cat_building_change_info.send_cats_update(this)

	// update task
	this.TaskUpdate(tables.TASK_COMPLETE_TYPE_CAT_FEED, false, 0, 1)
	if old_level != curr_level {
		this.TaskUpdate(tables.TASK_COMPLETE_TYPE_CAT_LEVEL_UP, true, curr_level, 1)
		// update ranking list
		this.update_ouqi(cat_id)
	}

	// 公告
	if is_max_level {
		anouncement_mgr.PushNew(ANOUNCEMENT_TYPE_CAT_FULL_LEVEL, true, this.Id, this.db.GetName(), this.db.GetLevel(), cat.Id, curr_level, 0, "")
	}

	return curr_level, curr_exp, 1
}

func (this *Player) cat_upstar(cat_id int32, cost_cat_ids []int32) int32 {
	curr_star, o := this.db.Cats.GetStar(cat_id)
	if !o {
		log.Error("Player[%v] not found cat[%v]", this.Id, cat_id)
		return int32(msg_client_message.E_ERR_CAT_NOT_FOUND)
	}

	cat_cid, _ := this.db.Cats.GetCfgId(cat_id)
	cat := cat_table_mgr.Map[cat_cid]
	if cat == nil {
		log.Error("Player[%v] not found cat[%v] table data", this.Id, cat.Id)
		return int32(msg_client_message.E_ERR_CAT_NOT_FOUND)
	}

	if curr_star >= cat.GetMaxStar() {
		log.Error("Player[%v] cat[%v] star is max", this.Id, cat.Id)
		return int32(msg_client_message.E_ERR_CAT_STAR_IS_MAX)
	}

	level, _ := this.db.Cats.GetLevel(cat_id)
	if level < cat.UpstarMaxLevels[curr_star-1] {
		log.Error("Player[%v] cat[%v] level[%v] not enough to upstar[%v]", this.Id, cat_id, level, curr_star+1)
		return int32(msg_client_message.E_ERR_CAT_UPSTAR_LEVEL_NOT_ENOUGH)
	}

	// 需要消耗的猫数
	need_cat_num := cat.GetUpstarCostCatNum(curr_star)
	if need_cat_num > 0 {
		if len(cost_cat_ids) < int(need_cat_num) {
			log.Error("Player[%v] Cat[%v] cost cat num[%v] not enoguh, need[%v]", this.Id, cat_id, len(cost_cat_ids), need_cat_num)
			return int32(msg_client_message.E_ERR_CAT_UPSTAR_COST_CAT_NOT_ENOUGH)
		}
	}

	cost_coin := cat.GetStarCostCoin(curr_star)
	if cost_coin > this.GetGold() {
		log.Error("Player[%v] cat[%v] cost coin[%v] not enough", this.Id, cat_id, this.GetGold())
		return int32(msg_client_message.E_ERR_CAT_UPSTAR_FAILED)
	}

	// 检测将消耗的猫
	same_num := int32(0)
	for i := int32(0); i < need_cat_num; i++ {
		cid := cost_cat_ids[i]
		if !this.db.Cats.HasIndex(cid) {
			log.Error("Player[%v] will cost Cat[%v] not found", this.Id, cat_id)
			return int32(msg_client_message.E_ERR_CAT_UPSTAR_COST_CAT_NOT_FOUND)
		}
		if this.IfCatBusy(cid) {
			log.Error("Player[%v] Cat[%v] state[%v] is no idle", this.Id, cid)
			return int32(msg_client_message.E_ERR_CAT_UPSTAR_COST_CAT_IS_USING)
		}
		locked, _ := this.db.Cats.GetLocked(cid)
		if locked > 0 {
			log.Error("Player[%v] Cat[%v] locked", this.Id, cid)
			return int32(msg_client_message.E_ERR_CAT_UPSTAR_COST_CAT_NOT_UNLOCK)
		}

		cost_cat_star, _ := this.db.Cats.GetStar(cid)
		if cost_cat_star != curr_star {
			log.Error("Upstar Cat[%v] Stars not same to Cost Cat[%v]", cat_id, cid)
			return int32(msg_client_message.E_ERR_CAT_UPSTAR_COST_CAT_STAR_DIFF)
		}

		cost_cat_cid, _ := this.db.Cats.GetCfgId(cid)
		if cost_cat_cid == cat_cid {
			same_num += 1
		}
	}

	curr_star += 1
	this.db.Cats.SetStar(cat_id, curr_star)

	// 消耗
	for i := int32(0); i < need_cat_num; i++ {
		cid := cost_cat_ids[i]
		if !this.SubCat(cid, "up_star_cat", "cat") {
			log.Warn("Player[%v] Cost Cat[%v] failed", this.Id, cid)
		}
	}

	this.SubGold(cost_coin, "upgrade_cat_star", "cat")

	// 消耗的猫相同升级技能
	for i := int32(0); i < same_num; i++ {
		cat_skill_level, _ := this.db.Cats.GetSkillLevel(cat_id)
		if cat_skill_level >= cat.GetMaxSkillLevel() {
			break
		}
		this.cat_skill_levelup(cat_id, nil)
	}

	this.item_cat_building_change_info.cat_update(this, cat_id)
	this.SendCatsUpdate()

	msg := &msg_client_message.S2CCatUpgradeStarResult{}
	msg.CatId = cat_id
	msg.CatStar = curr_star
	this.Send(uint16(msg_client_message.S2CCatUpgradeStarResult_ProtoID), msg)

	// update task
	this.TaskUpdate(tables.TASK_COMPLETE_TYPE_CAT_UP_STAR, false, curr_star, 1)

	// update ouqi
	this.update_ouqi(cat_id)

	log.Info("Player[%v] Cat[%v] UpStar[%v]", this.Id, cat_id, curr_star)

	return 1
}

func (this *Player) cat_skill_levelup(cat_id int32, cost_cat_ids []int32) int32 {
	cat_cid, o := this.db.Cats.GetCfgId(cat_id)
	if !o {
		log.Error("Player[%v] not found cat[%v]", this.Id, cat_id)
		return int32(msg_client_message.E_ERR_CAT_NOT_FOUND)
	}
	cat := cat_table_mgr.GetCat(cat_cid)
	if cat == nil {
		log.Error("Not found cat[%v] table data", cat_cid)
		return int32(msg_client_message.E_ERR_CAT_TABLE_DATA_NOT_FOUND)
	}

	skill_level, _ := this.db.Cats.GetSkillLevel(cat_id)
	if skill_level >= cat.GetMaxSkillLevel() {
		log.Error("Player[%v] cat[%v] skill level[%v] is max", this.Id, cat_id, skill_level)
		return int32(msg_client_message.E_ERR_CAT_SKILL_LEVEL_IS_MAX)
	}

	cost_coin := cat.GetSkillLevelCostCoin(skill_level)
	if cost_coin > this.GetGold() {
		log.Error("Player[%v] cat[%v] up skill cost coin[%v] not enough", this.Id, cat_id, this.GetGold())
		return int32(msg_client_message.E_ERR_CAT_UPSKILL_NOT_ENOUGH_COIN)
	}

	up_level := skill_level
	if cost_cat_ids != nil {
		if len(cost_cat_ids) == 0 {
			log.Error("Player[%v] cat[%v] cost cats is empty", this.Id, cat_id)
			return int32(msg_client_message.E_ERR_CAT_UPSKILL_COST_CAT_CANT_EMPTY)
		}

		for _, cost_cat_id := range cost_cat_ids {
			if cat_id == cost_cat_id {
				log.Error("Player[%v] up level use the cost cat is self", this.Id)
				return int32(msg_client_message.E_ERR_CAT_UPSKILL_COST_CAT_CANT_SELF)
			}

			if up_level >= cat.GetMaxSkillLevel() {
				break
			}

			if cost_cat_id > 0 {
				if !this.db.Cats.HasIndex(cost_cat_id) {
					log.Error("Player[%v] have no cat[%v]", this.Id, cost_cat_id)
					return int32(msg_client_message.E_ERR_CAT_UPSKILL_COST_CAT_NOT_FOUND)
				}
				if this.IfCatBusy(cost_cat_id) {
					log.Error("Player[%v] is busy", this.Id)
					return int32(msg_client_message.E_ERR_CAT_UPSKILL_COST_CAT_IS_USING)
				}
				locked, _ := this.db.Cats.GetLocked(cost_cat_id)
				if locked > 0 {
					log.Error("Player[%v] cat[%v] locked", this.Id, cost_cat_id)
					return int32(msg_client_message.E_ERR_CAT_UPSKILL_COST_CAT_LOCKED)
				}
				cfg_id, _ := this.db.Cats.GetCfgId(cost_cat_id)

				// 是否有需要消耗的猫
				found := false
				for _, id := range cat.UpSkills {
					if cfg_id == id {
						found = true
						break
					}
				}
				if !found {
					log.Error("Player[%v] cat[%v] UpSkill cost cat[%v] is not include in configure", this.Id, cat_id, cost_cat_id)
					return int32(msg_client_message.E_ERR_CAT_UPSKILL_NO_VALID_COST_CAT)
				}
			}

			up_level += 1
		}
	} else {
		up_level += 1
	}

	this.db.Cats.SetSkillLevel(cat_id, up_level)

	// 消耗猫
	if cost_cat_ids != nil && len(cost_cat_ids) > 0 {
		for _, cost_cat_id := range cost_cat_ids {
			if cost_cat_id > 0 && !this.SubCat(cost_cat_id, "up_cat_skill_level", "cat") {
				log.Warn("Player[%v] sub Cat[%v] failed for Cat[%v] up Skill[%v]", this.Id, cost_cat_id, cat_id, cat.SkillId)
			}
		}
	}

	if cost_coin > 0 {
		this.SubGold(cost_coin, "up_cat_skill_level", "cat")
	}

	this.item_cat_building_change_info.cat_update(this, cat_id)
	this.SendCatsUpdate()

	msg := &msg_client_message.S2CCatSkillLevelUpResult{}
	msg.CatId = cat_id
	msg.SkillLevel = up_level
	this.Send(uint16(msg_client_message.S2CCatSkillLevelUpResult_ProtoID), msg)

	// update task
	this.TaskUpdate(tables.TASK_COMPLETE_TYPE_CAT_UP_SKILL_LEVEL, true, up_level, 1)

	log.Info("Player[%v] Cat[%v] skill level up to %v", this.Id, cat_id, up_level)

	return up_level
}

func (this *Player) rename_cat(cat_id int32, new_nick string) int32 {
	nick, o := this.db.Cats.GetNick(cat_id)
	if !o {
		log.Error("Player[%v] cat[%v] not found", this.Id, cat_id)
		return int32(msg_client_message.E_ERR_CAT_NOT_FOUND)
	}

	if new_nick == nick {
		log.Error("Player[%v] cat[%v] old nick same to new nick", this.Id, cat_id)
		return int32(msg_client_message.E_ERR_CAT_RENAME_CANT_USE_OLD)
	}

	this.db.Cats.SetNick(cat_id, new_nick)

	this.item_cat_building_change_info.cat_update(this, cat_id)
	this.SendCatsUpdate()

	msg := &msg_client_message.S2CCatRenameNickResult{}
	msg.CatId = cat_id
	msg.NewNick = new_nick
	this.Send(uint16(msg_client_message.S2CCatRenameNickResult_ProtoID), msg)

	return 1
}

func (this *Player) lock_cat(cat_id int32, is_lock bool) int32 {
	curr_lock, o := this.db.Cats.GetLocked(cat_id)
	if !o {
		log.Error("Player[%v] cat[%v] not found", this.Id, cat_id)
		return int32(msg_client_message.E_ERR_CAT_NOT_FOUND)
	}

	is_curr_lock := true
	if curr_lock == 0 {
		is_curr_lock = false
	}
	if is_curr_lock == is_lock {
		log.Error("Player[%v] cat[%v] is already lock[%v]", this.Id, cat_id, is_curr_lock)
		return int32(msg_client_message.E_ERR_CAT_UPSKILL_COST_CAT_LOCKED)
	}

	if is_lock {
		curr_lock = 1
	} else {
		curr_lock = 0
	}

	this.db.Cats.SetLocked(cat_id, curr_lock)

	this.item_cat_building_change_info.cat_update(this, cat_id)
	this.item_cat_building_change_info.send_cats_update(this)

	msg := &msg_client_message.S2CCatLockResult{}
	msg.CatId = cat_id
	msg.Locked = is_lock
	this.Send(uint16(msg_client_message.S2CCatLockResult_ProtoID), msg)

	return 1
}

func (this *Player) decompose_cat(cat_ids []int32) int32 {
	if cat_ids == nil || len(cat_ids) == 0 {
		return -1
	}
	add_stone := int32(0)
	for i := 0; i < len(cat_ids); i++ {
		cat_cid, o := this.db.Cats.GetCfgId(cat_ids[i])
		if !o {
			log.Error("Player[%v] cat[%v] not found", this.Id, cat_ids[i])
			return int32(msg_client_message.E_ERR_CAT_NOT_FOUND)
		}

		if this.IfCatBusy(cat_ids[i]) {
			log.Error("Player[%v] cat[%v] is busy, cant to decompose", this.Id, cat_ids[i])
			return int32(msg_client_message.E_ERR_CAT_IS_BUSY)
		}

		cat := cat_table_mgr.GetCat(cat_cid)
		if cat == nil {
			log.Error("cat[%v] table data not found", cat_cid)
			return int32(msg_client_message.E_ERR_CAT_TABLE_DATA_NOT_FOUND)
		}
		add_stone += cat.Stone

	}

	for i := 0; i < len(cat_ids); i++ {
		this.SubCat(cat_ids[i], "decompose_cat", "cat")
	}
	this.SendCatsUpdate()
	this.AddSoulStone(add_stone, "decompose_cat", "cat")

	response := &msg_client_message.S2CCatDecomposeResult{}
	response.CatId = cat_ids
	response.GetSoulStone = add_stone
	this.Send(uint16(msg_client_message.S2CCatDecomposeResult_ProtoID), response)

	return 1
}

func (this *Player) send_cats_update(cat_ids []int32) int32 {
	if cat_ids == nil || len(cat_ids) == 0 {
		return 0
	}

	l := int32(len(cat_ids))
	for i := int32(0); i < l; i++ {
		this.item_cat_building_change_info.cat_update(this, cat_ids[i])
	}
	this.SendCatsUpdate()

	return l
}

func get_cat_coin_ability(cat *tables.XmlCharacterItem, d *dbPlayerCatData) int32 {
	add_coin_ouqi := int32(0)
	if cat.AddCoins != nil && len(cat.AddCoins) >= int(d.Star) && d.Star >= 1 {
		add_coin_ouqi = cat.AddCoins[d.Star-1]
	}
	return (cat.GrowthRate*d.Level/100 + d.CoinAbility + d.CoinAbility*cat.InitialRate*(d.Level-1)/100) + add_coin_ouqi
}

func get_cat_match_ability(cat *tables.XmlCharacterItem, d *dbPlayerCatData) int32 {
	add_match_ouqi := int32(0)
	if cat.AddMatchs != nil && len(cat.AddMatchs) >= int(d.Star) && d.Star >= 1 {
		add_match_ouqi = cat.AddMatchs[d.Star-1]
	}
	return (cat.GrowthRate*d.Level/100 + d.MatchAbility + d.MatchAbility*cat.InitialRate*(d.Level-1)/100) + add_match_ouqi
}

func get_cat_explore_ability(cat *tables.XmlCharacterItem, d *dbPlayerCatData) int32 {
	add_explore_ouqi := int32(0)
	if cat.AddExplores != nil && len(cat.AddExplores) >= int(d.Star) && d.Star >= 1 {
		add_explore_ouqi = cat.AddExplores[d.Star-1]
	}
	return (cat.GrowthRate*d.Level/100 + d.ExploreAbility + d.ExploreAbility*cat.InitialRate*(d.Level-1)/100) + add_explore_ouqi
}

func (this *Player) get_cat_match_ability(cat_id int32) int32 {
	cat_db := this.db.Cats.Get(cat_id)
	if cat_db == nil {
		return 0
	}
	cat_cid, o := this.db.Cats.GetCfgId(cat_id)
	if !o {
		return 0
	}
	cat_cfg := cat_table_mgr.GetCat(cat_cid)
	return get_cat_match_ability(cat_cfg, cat_db)
}

func (this *dbPlayerCatColumn) CalcOuqi(cat_id int32) int32 {
	this.m_row.m_lock.UnSafeRLock("dbPlayerCatColumn.CalcOuqi")
	defer this.m_row.m_lock.UnSafeRUnlock()

	d := this.m_data[cat_id]
	if d == nil {
		return 0
	}

	cat := cat_table_mgr.GetCat(d.CfgId)
	if cat == nil {
		return 0
	}

	add_skill_ouqi := int32(0)
	if cat.SkillLevelScores != nil && len(cat.SkillLevelScores) >= int(d.Star) && d.Star >= 1 {
		add_skill_ouqi = cat.SkillLevelScores[d.SkillLevel-1]
	}

	//ouqi := get_cat_coin_ability(cat, d) + get_cat_match_ability(cat, d) + get_cat_explore_ability(cat, d) + add_skill_ouqi

	return d.CoinAbility + d.MatchAbility + d.Exp + add_skill_ouqi
}

func (this *Player) update_ouqi(cat_id int32) {
	ouqi := this.db.Cats.CalcOuqi(cat_id)
	if this.rpc_rank_list_update_data(common.RANK_LIST_TYPE_CAT_OUQI, []int32{cat_id, ouqi}) == nil {
		log.Warn("Player[%v] remote update cat[%v] ouqi[%v] failed", this.Id, cat_id, ouqi)
	}
}

func (this *Player) get_player_cat_info(player_id int32, cat_id int32) int32 {
	var cat_level, cat_exp, cat_star, cat_skill_level, cat_add_coin, cat_add_match, cat_add_explore int32
	player := player_mgr.GetPlayerById(player_id)
	if player == nil {
		/*result := this.rpc_call_player_cat_info(player_id, cat_id)
		if result.Error < 0 {
			return result.Error
		}
		cat_exp = result.ToPlayerCatExp
		cat_level = result.ToPlayerCatLevel
		cat_star = result.ToPlayerCatStar
		cat_skill_level = result.ToPlayerCatSkillLevel
		cat_add_coin = result.ToPlayerCatAddCoin
		cat_add_match = result.ToPlayerCatAddMatch
		cat_add_explore = result.ToPlayerCatAddExplore*/
	} else {
		o := false
		cat_level, o = player.db.Cats.GetLevel(cat_id)
		if !o {
			log.Error("Player[%v] no cat[%v]", player_id, cat_id)
			return int32(msg_client_message.E_ERR_CAT_NOT_FOUND)
		}
		cat_exp, _ = player.db.Cats.GetExp(cat_id)
		cat_star, _ = player.db.Cats.GetStar(cat_id)
		cat_skill_level, _ = player.db.Cats.GetSkillLevel(cat_id)
		cat_add_coin, _ = player.db.Cats.GetCoinAbility(cat_id)
		cat_add_match, _ = player.db.Cats.GetMatchAbility(cat_id)
		cat_add_explore, _ = player.db.Cats.GetExploreAbility(cat_id)
	}

	response := &msg_client_message.S2CPlayerCatInfoResult{
		PlayerId:      player_id,
		CatId:         cat_id,
		CatLevel:      cat_level,
		CatExp:        cat_exp,
		CatStar:       cat_star,
		CatSkillLevel: cat_skill_level,
		CatAddCoin:    cat_add_coin,
		CatAddMatch:   cat_add_match,
		CatAddExplore: cat_add_explore,
	}
	this.Send(uint16(msg_client_message.S2CPlayerCatInfoResult_ProtoID), response)

	return 1
}
