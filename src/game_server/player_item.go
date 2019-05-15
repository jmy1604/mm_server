package main

import (
	"mm_server/libs/log"
	"mm_server/proto/gen_go/client_message"
	"mm_server/src/common"
	"mm_server/src/tables"
	"sync"
	"time"
)

// 物品类型
const (
	ITEM_TYPE_NONE         = iota
	ITEM_TYPE_RESOURCE     = 1  // 资源类
	ITEM_TYPE_DRAW         = 2  // 抽卡券
	ITEM_TYPE_ELIMINATE    = 3  // 消除类
	ITEM_TYPE_OBSTACLE     = 4  // 障碍类
	ITEM_TYPE_DECORATION   = 5  // 装饰材料
	ITEM_TYPE_FOSTER       = 6  // 寄养卡
	ITEM_TYPE_CAT_FRAGMENT = 7  // 猫碎片
	ITEM_TYPE_SPIRIT       = 8  // 体力道具
	ITEM_TYPE_JUMP         = 9  // 跳跳道具
	ITEM_TYPE_CAT          = 10 // 猫道具
	ITEM_TYPE_HEAD         = 11 // 头像  临时定义
)

// 其他属性
const (
	ITEM_RESOURCE_ID_RMB          = 1  // 人民币
	ITEM_RESOURCE_ID_GOLD         = 2  // 金币
	ITEM_RESOURCE_ID_DIAMOND      = 3  // 钻石
	ITEM_RESOURCE_ID_CAT_FOOD     = 4  // 猫粮
	ITEM_RESOURCE_ID_SPIRIT       = 5  // 体力
	ITEM_RESOURCE_ID_FRIEND_POINT = 6  // 友情值
	ITEM_RESOURCE_ID_CHARM_VALUE  = 7  // 魅力值
	ITEM_RESOURCE_ID_EXP_VALUE    = 8  // 经验值
	ITEM_RESOURCE_ID_SOUL_STONE   = 9  // 魂石
	ITEM_RESOURCE_ID_CHARM_MEDAL  = 10 // 魅力勋章
	ITEM_RESOURCE_ID_TIME         = 11 // 时间
	ITEM_RESOURCE_ID_STAR         = 12 // 星数
	ITEM_RESOURCE_ID_CAT_EXP      = 13 // 猫经验
	ITEM_RESOURCE_ID_ACTION       = 14 // 行动力
)

type ItemCatBuildingChangeInfo struct {
	items_update                map[int32]*msg_client_message.ItemInfo     // 物品变化
	items_update_lock           *sync.RWMutex                              // 物品变化锁
	cats_add                    map[int32]*msg_client_message.CatInfo      // 猫增加
	cats_add_lock               *sync.RWMutex                              // 增加猫锁
	cats_remove                 map[int32]int32                            // 猫减少
	cats_remove_lock            *sync.RWMutex                              // 减少猫锁
	cats_update                 map[int32]*msg_client_message.CatInfo      // 猫变化
	cats_update_lock            *sync.RWMutex                              // 猫变化锁
	buildings_add               map[int32]*msg_client_message.BuildingInfo // 建筑物增加
	buildings_add_lock          *sync.RWMutex                              // 增减建筑物锁
	buildings_remove            map[int32]int32                            // 建筑物减少
	buildings_remove_lock       *sync.RWMutex                              // 减少建筑物锁
	buildings_update            map[int32]*msg_client_message.BuildingInfo // 建筑物变化
	buildings_update_lock       *sync.RWMutex                              // 建筑物变化锁
	depot_buildings_update      map[int32]int32                            // 仓库建筑物变化
	depot_buildings_update_lock *sync.RWMutex                              // 仓库建筑物变化锁
}

func (this *ItemCatBuildingChangeInfo) init() {
	this.items_update_lock = &sync.RWMutex{}
	this.cats_add_lock = &sync.RWMutex{}
	this.cats_remove_lock = &sync.RWMutex{}
	this.cats_update_lock = &sync.RWMutex{}
	this.buildings_add_lock = &sync.RWMutex{}
	this.buildings_remove_lock = &sync.RWMutex{}
	this.buildings_update_lock = &sync.RWMutex{}
	this.depot_buildings_update_lock = &sync.RWMutex{}
}

func (this *ItemCatBuildingChangeInfo) item_update(p *Player, item_id int32) {
	//this.items_update_lock.Lock()
	//defer this.items_update_lock.Unlock()

	if this.items_update == nil {
		this.items_update = make(map[int32]*msg_client_message.ItemInfo)
	}
	if this.items_update[item_id] == nil {
		this.items_update[item_id] = &msg_client_message.ItemInfo{}
	}

	this.items_update[item_id].ItemCfgId = item_id

	item := p.db.Items.Get(item_id)

	if item == nil {
		this.items_update[item_id].ItemNum = 0
	} else {
		this.items_update[item_id].ItemNum = item.ItemNum
		this.items_update[item_id].RemainSeconds = get_time_item_remain_seconds(item)
	}
}

func cat_values_assign(p *Player, dst_info *msg_client_message.CatInfo, cat_id int32) {
	if !p.db.Cats.HasIndex(cat_id) {
		return
	}
	cfg_id, _ := p.db.Cats.GetCfgId(cat_id)
	dst_info.CatCfgId = cfg_id
	dst_info.Id = cat_id
	exp, _ := p.db.Cats.GetExp(cat_id)
	dst_info.Exp = exp
	level, _ := p.db.Cats.GetLevel(cat_id)
	dst_info.Level = level
	star, _ := p.db.Cats.GetStar(cat_id)
	dst_info.Star = star
	nick, _ := p.db.Cats.GetNick(cat_id)
	dst_info.Nick = nick
	skill_level, _ := p.db.Cats.GetSkillLevel(cat_id)
	dst_info.SkillLevel = skill_level
	locked, _ := p.db.Cats.GetLocked(cat_id)
	is_lock := false
	if locked != 0 {
		is_lock = true
	}
	dst_info.Locked = is_lock
	coin_ability, _ := p.db.Cats.GetCoinAbility(cat_id)
	dst_info.CoinAbility = coin_ability
	explore_ability, _ := p.db.Cats.GetExploreAbility(cat_id)
	dst_info.ExploreAbility = explore_ability
	match_ability, _ := p.db.Cats.GetMatchAbility(cat_id)
	dst_info.MatchAbility = match_ability
	state := p.GetCatState(cat_id)
	dst_info.State = state
}

func (this *ItemCatBuildingChangeInfo) cat_add(p *Player, cat_id int32) bool {
	//this.cats_add_lock.Lock()
	//defer this.cats_add_lock.Unlock()

	if !p.db.Cats.HasIndex(cat_id) {
		log.Error("玩家[%v]的猫[%v]不存在", p.Id, cat_id)
		return false
	}
	if this.cats_add == nil {
		this.cats_add = make(map[int32]*msg_client_message.CatInfo)
	}
	if this.cats_add[cat_id] == nil {
		this.cats_add[cat_id] = &msg_client_message.CatInfo{}
	}
	cat_values_assign(p, this.cats_add[cat_id], cat_id)
	log.Info("!!!!!!! 增加的猫[%v], level[%v], star[%v], skill_level[%v]", cat_id, this.cats_add[cat_id].GetLevel(), this.cats_add[cat_id].GetStar(), this.cats_add[cat_id].GetSkillLevel())
	return true
}

func (this *ItemCatBuildingChangeInfo) cat_remove(p *Player, cat_id int32) bool {
	//this.cats_remove_lock.Lock()
	//defer this.cats_remove_lock.Unlock()

	if this.cats_remove == nil {
		this.cats_remove = make(map[int32]int32)
	}
	if _, o := this.cats_remove[cat_id]; o {
		log.Error("玩家[%v]的猫[%v]已删除", p.Id, cat_id)
		return false
	}
	this.cats_remove[cat_id] = cat_id
	return true
}

func (this *ItemCatBuildingChangeInfo) cat_update(p *Player, cat_id int32) bool {
	//his.cats_update_lock.Lock()
	//defer this.cats_update_lock.Unlock()

	if !p.db.Cats.HasIndex(cat_id) {
		log.Error("找不到玩家[%v]的猫[%v]", p.Id, cat_id)
		return false
	}
	if this.cats_update == nil {
		this.cats_update = make(map[int32]*msg_client_message.CatInfo)
	}
	if this.cats_update[cat_id] == nil {
		this.cats_update[cat_id] = &msg_client_message.CatInfo{}
	}
	cat_values_assign(p, this.cats_update[cat_id], cat_id)
	this.cats_update[cat_id].State = p.GetCatState(cat_id)
	return true
}

func building_values_assign(dst_info *msg_client_message.BuildingInfo, src_info *dbPlayerBuildingData) {
	dst_info.CfgId = src_info.CfgId
	dst_info.Id = src_info.Id
	dst_info.X = src_info.X
	dst_info.Y = src_info.Y
	dst_info.Dir = src_info.Dir
}

func (this *ItemCatBuildingChangeInfo) building_add(p *Player, building_id int32) bool {
	//this.buildings_add_lock.Lock()
	//defer this.buildings_add_lock.Unlock()

	building := p.db.Buildings.Get(building_id)
	if building == nil {
		log.Error("找不到玩家[%v]建筑物[%v]", p.Id, building_id)
		return false
	}
	if this.buildings_add == nil {
		this.buildings_add = make(map[int32]*msg_client_message.BuildingInfo)
	}
	if this.buildings_add[building_id] == nil {
		this.buildings_add[building_id] = &msg_client_message.BuildingInfo{}
	}
	building_values_assign(this.buildings_add[building_id], building)
	return true
}

func (this *ItemCatBuildingChangeInfo) building_remove(p *Player, building_id int32) bool {
	//this.buildings_remove_lock.Lock()
	//defer this.buildings_remove_lock.Unlock()

	if this.buildings_remove == nil {
		this.buildings_remove = make(map[int32]int32)
	}
	if _, o := this.buildings_remove[building_id]; o {
		log.Error("玩家[%v]的猫[%v]已删除", p.Id, building_id)
		return false
	}
	this.buildings_remove[building_id] = building_id
	return true
}

func (this *ItemCatBuildingChangeInfo) building_update(p *Player, building_id int32) bool {
	//this.buildings_update_lock.Lock()
	//defer this.buildings_update_lock.Unlock()

	building := p.db.Buildings.Get(building_id)
	if building == nil {
		log.Error("找不到玩家[%v]的猫[%v]", p.Id, building_id)
		return false
	}
	if this.buildings_update == nil {
		this.buildings_update = make(map[int32]*msg_client_message.BuildingInfo)
	}
	if this.buildings_update[building_id] == nil {
		this.buildings_update[building_id] = &msg_client_message.BuildingInfo{}
	}
	building_values_assign(this.buildings_update[building_id], building)
	return true
}

func (this *ItemCatBuildingChangeInfo) depot_building_update(p *Player, depot_building_id int32) {
	//this.depot_buildings_update_lock.Lock()
	//defer this.depot_buildings_update_lock.Unlock()

	if this.depot_buildings_update == nil {
		this.depot_buildings_update = make(map[int32]int32)
	}
	depot_building := p.db.BuildingDepots.Get(depot_building_id)
	if depot_building == nil {
		this.depot_buildings_update[depot_building_id] = 0
	} else {
		this.depot_buildings_update[depot_building_id] = depot_building.Num
	}
}

func (this *ItemCatBuildingChangeInfo) send_items_update(p *Player) bool {
	//this.items_update_lock.Lock()
	//defer this.items_update_lock.Unlock()

	if this.items_update == nil || len(this.items_update) == 0 {
		return false
	}

	msg := &msg_client_message.S2CItemsInfoUpdate{}
	msg.Items = make([]*msg_client_message.ItemInfo, len(this.items_update))
	i := int32(0)
	for _, v := range this.items_update {
		msg.Items[i] = v
		i += 1
	}

	p.Send(uint16(msg_client_message.S2CItemsInfoUpdate_ProtoID), msg)

	this.items_update = nil

	return true
}

func (this *ItemCatBuildingChangeInfo) send_buildings_update(p *Player) bool {
	msg := &msg_client_message.S2CBuildingsInfoUpdate{}

	// 增加的建筑物
	//this.buildings_add_lock.Lock()
	if this.buildings_add != nil && len(this.buildings_add) > 0 {
		msg.AddBuildings = make([]*msg_client_message.BuildingInfo, len(this.buildings_add))
		i := int32(0)
		for _, v := range this.buildings_add {
			msg.AddBuildings[i] = v
			i += 1
		}

	}
	//this.buildings_add_lock.Unlock()

	// 删除的建筑物
	//this.buildings_remove_lock.Lock()
	if this.buildings_remove != nil && len(this.buildings_remove) > 0 {
		msg.RemoveBuildings = make([]int32, len(this.buildings_remove))
		i := int32(0)
		for k, _ := range this.buildings_remove {
			msg.RemoveBuildings[i] = k
			i += 1
		}

	}
	//this.buildings_remove_lock.Unlock()

	// 更新的建筑物
	//this.buildings_update_lock.Lock()
	if this.buildings_update != nil && len(this.buildings_update) > 0 {
		msg.UpdateBuildings = make([]*msg_client_message.BuildingInfo, len(this.buildings_update))
		i := int32(0)
		for _, v := range this.buildings_update {
			msg.UpdateBuildings[i] = v
			i += 1
		}

	}
	//this.buildings_update_lock.Unlock()

	if msg.AddBuildings == nil && msg.RemoveBuildings == nil && msg.UpdateBuildings == nil {
		return false
	}

	p.Send(uint16(msg_client_message.S2CBuildingsInfoUpdate_ProtoID), msg)

	this.buildings_add = nil
	this.buildings_remove = nil
	this.buildings_update = nil

	return true
}

func (this *ItemCatBuildingChangeInfo) send_cats_update(p *Player) bool {
	msg := &msg_client_message.S2CCatsInfoUpdate{}

	// 增加的猫
	//this.cats_add_lock.Lock()
	if this.cats_add != nil && len(this.cats_add) > 0 {
		msg.AddCats = make([]*msg_client_message.CatInfo, len(this.cats_add))
		i := int32(0)
		for _, v := range this.cats_add {
			msg.AddCats[i] = v
			i += 1
		}

	}
	//this.cats_add_lock.Unlock()

	// 删除的猫
	//this.cats_remove_lock.Lock()
	if this.cats_remove != nil && len(this.cats_remove) > 0 {
		msg.RemoveCats = make([]int32, len(this.cats_remove))
		i := int32(0)
		for k, _ := range this.cats_remove {
			msg.RemoveCats[i] = k
			i += 1
		}

	}
	//this.cats_remove_lock.Unlock()

	// 更新的猫
	//this.cats_update_lock.Lock()
	if this.cats_update != nil && len(this.cats_update) > 0 {
		msg.UpdateCats = make([]*msg_client_message.CatInfo, len(this.cats_update))
		i := int32(0)
		for _, v := range this.cats_update {
			msg.UpdateCats[i] = v
			i += 1
		}

	}
	//this.cats_update_lock.Unlock()

	if msg.AddCats == nil && msg.RemoveCats == nil && msg.UpdateCats == nil {
		return false
	}

	p.Send(uint16(msg_client_message.S2CCatsInfoUpdate_ProtoID), msg)

	this.cats_add = nil
	this.cats_remove = nil
	this.cats_update = nil

	return true
}

func (this *ItemCatBuildingChangeInfo) send_depot_building_update(p *Player) bool {
	//this.depot_buildings_update_lock.Lock()
	//defer this.depot_buildings_update_lock.Unlock()

	if this.depot_buildings_update == nil || len(this.depot_buildings_update) == 0 {
		return false
	}

	msg := &msg_client_message.S2CDepotBuildingInfoUpdate{}
	msg.Buildings = make([]*msg_client_message.DepotBuildingInfo, len(this.depot_buildings_update))
	i := int32(0)
	for k, v := range this.depot_buildings_update {
		msg.Buildings[i] = &msg_client_message.DepotBuildingInfo{}
		msg.Buildings[i].CfgId = k
		msg.Buildings[i].Num = v
		i += 1
	}

	p.Send(uint16(msg_client_message.S2CDepotBuildingInfoUpdate_ProtoID), msg)

	this.depot_buildings_update = nil

	return true
}

// 计算计时物品剩余时间
func get_time_item_remain_seconds(item *dbPlayerItemData) int32 {
	if item.StartTimeUnix == 0 {
		return 0
	}

	now_time := int32(time.Now().Unix())
	cost_seconds := now_time - item.StartTimeUnix
	// 剩余时间小于等于3秒一律算到时
	left_seconds := item.RemainSeconds - cost_seconds
	if left_seconds <= 3 {
		return 0
	}
	return left_seconds
}

//////////////////////////////////////////////////////////////////////////////////
func (this *Player) SendItemsUpdate() {
	this.item_cat_building_change_info.send_items_update(this)
}

func (this *Player) SendCatsUpdate() {
	this.item_cat_building_change_info.send_cats_update(this)
}

func (this *Player) SendCatUpdate(cat_id int32) {
	this.item_cat_building_change_info.cat_update(this, cat_id)
	this.item_cat_building_change_info.send_cats_update(this)
}

func (this *Player) SendCatAdd(cat_id int32) {
	this.item_cat_building_change_info.cat_add(this, cat_id)
	this.item_cat_building_change_info.send_cats_update(this)
}

func (this *Player) SendCatRemove(cat_id int32) {
	this.item_cat_building_change_info.cat_remove(this, cat_id)
	this.item_cat_building_change_info.send_cats_update(this)
}

func (this *Player) SendBuildingUpdate() {
	this.item_cat_building_change_info.send_buildings_update(this)
}

func (this *Player) SendDepotBuildingUpdate() {
	this.item_cat_building_change_info.send_depot_building_update(this)
}

// 体力增长计算
func (this *Player) CalcSpirit() int32 {
	curr_stamina := this.db.Info.GetSpirit()
	cp := player_level_table_mgr.Map[this.db.GetLevel()]
	if cp == nil {
		return curr_stamina
	}

	last_save := this.db.Info.GetSaveLastSpiritPointTime()
	now := time.Now().Unix()
	used_seconds := int32(now) - last_save
	if curr_stamina < cp.MaxPower && used_seconds > global_config.SpiritGrowPointNeedMinute*60 {
		y := used_seconds % global_config.SpiritGrowPointNeedMinute
		grow_points := used_seconds / (global_config.SpiritGrowPointNeedMinute * 60)
		if curr_stamina+grow_points > cp.MaxPower {
			grow_points = cp.MaxPower - curr_stamina
		}
		if grow_points > 0 {
			this.db.Info.IncbySpirit(grow_points)
			this.db.Info.SetSaveLastSpiritPointTime(int32(now) - y)
		}
	}
	return this.db.Info.GetSpirit()
}

func (this *Player) GetItemResourceValue(other_id int32) int32 {
	switch other_id {
	case ITEM_RESOURCE_ID_RMB:
		{
			return 0
		}
	case ITEM_RESOURCE_ID_GOLD:
		{
			return this.db.Info.GetGold()
		}
	case ITEM_RESOURCE_ID_DIAMOND:
		{
			return this.db.Info.GetDiamond()
		}
	case ITEM_RESOURCE_ID_CAT_FOOD:
		{
			return this.db.Info.GetCatFood()
		}
	case ITEM_RESOURCE_ID_SPIRIT:
		{
			// 体力要即时计算
			return this.CalcSpirit()
		}
	case ITEM_RESOURCE_ID_FRIEND_POINT:
		{
			return this.db.Info.GetFriendPoints()
		}
	case ITEM_RESOURCE_ID_CHARM_VALUE:
		{
			return this.db.Info.GetCharmVal()
		}
	case ITEM_RESOURCE_ID_EXP_VALUE:
		{
			return this.db.Info.GetExp()
		}
	case ITEM_RESOURCE_ID_SOUL_STONE:
		{
			return this.db.Info.GetSoulStone()
		}
	case ITEM_RESOURCE_ID_CHARM_MEDAL:
		{
			return this.db.Info.GetCharmMedal()
		}
	default:
		{
			num, o := this.db.Items.GetItemNum(other_id)
			if !o {
				return 0
			}
			return num
		}
	}
}

func (this *Player) get_item(item_id int32) int32 {
	return 0
}

func (this *Player) add_diamond(diamond int32) {

}

func (this *Player) get_resource(resource_id int32) int32 {
	return 0
}

func (this *Player) add_resource(item_id, item_num int32) {

}

func (this *Player) add_resources(resources []int32) {

}

func (this *Player) AddItemResource(cid, num int32, reason, mod string) int32 {
	switch cid {
	case ITEM_RESOURCE_ID_RMB:
		{
			log.Debug("rmb is not supported")
		}
	case ITEM_RESOURCE_ID_GOLD:
		{
			return this.AddGold(num, reason, mod)
		}
	case ITEM_RESOURCE_ID_DIAMOND:
		{
			return this.AddDiamond(num, reason, mod)
		}
	case ITEM_RESOURCE_ID_CAT_FOOD:
		{
			return this.AddCatFood(num, reason, mod)
		}
	case ITEM_RESOURCE_ID_CHARM_MEDAL:
		{
			return this.AddCharmMedal(num, reason, mod)
		}
	case ITEM_RESOURCE_ID_CHARM_VALUE:
		{
			return this.AddCharmVal(num, reason, mod)
		}
	case ITEM_RESOURCE_ID_EXP_VALUE:
		{
			this.AddExp(num, reason, mod)
		}
	case ITEM_RESOURCE_ID_FRIEND_POINT:
		{
			return this.AddFriendPoints(num, reason, mod)
		}
	case ITEM_RESOURCE_ID_SOUL_STONE:
		{
			return this.AddSoulStone(num, reason, mod)
		}
	case ITEM_RESOURCE_ID_SPIRIT:
		{
			return this.AddSpirit(num, reason, mod)
		}
	default:
		{
			if this.AddItem(cid, num, reason, mod, true) == nil {
				return -1
			}
		}
	}
	return 1
}

func (this *Player) RemoveItemResource(cid, num int32, reason, mod string) int32 {
	switch cid {
	case ITEM_RESOURCE_ID_RMB:
		{
			log.Debug("rmb is not supported")
		}
	case ITEM_RESOURCE_ID_GOLD:
		{
			return this.SubGold(num, reason, mod)
		}
	case ITEM_RESOURCE_ID_DIAMOND:
		{
			return this.SubDiamond(num, reason, mod)
		}
	case ITEM_RESOURCE_ID_CAT_FOOD:
		{
			return this.SubCatFood(num, reason, mod)
		}
	case ITEM_RESOURCE_ID_CHARM_MEDAL:
		{
			return this.SubCharmMedal(num, reason, mod)
		}
	case ITEM_RESOURCE_ID_CHARM_VALUE:
		{
			return this.SubCharmVal(num, reason, mod)
		}
	case ITEM_RESOURCE_ID_EXP_VALUE:
		{

		}
	case ITEM_RESOURCE_ID_FRIEND_POINT:
		{
			return this.SubFriendPoints(num, reason, mod)
		}
	case ITEM_RESOURCE_ID_SOUL_STONE:
		{
			return this.SubSoulStone(num, reason, mod)
		}
	case ITEM_RESOURCE_ID_SPIRIT:
		{
			return this.SubSpirit(num, reason, mod)
		}
	default:
		{
			if this.RemoveItem(cid, num, true) == nil {
				return -1
			}
		}
	}
	return 1
}

func (this *Player) ChkResEnough(resources []int32) bool {
	tmp_len := int32(len(resources))
	var item_type, item_val int32
	for idx := int32(0); idx < tmp_len; idx += 2 {
		item_type = resources[idx]
		item_val = resources[idx+1]
		if this.GetItemResourceValue(item_type) < item_val {
			return false
		}
	}

	return true
}

func (this *Player) RemoveResources(resources []int32, reason, mod string) {
	tmp_len := int32(len(resources))
	var item_type, item_val int32
	for idx := int32(0); idx < tmp_len; idx += 2 {
		item_type = resources[idx]
		item_val = resources[idx+1]
		this.RemoveItemResource(item_type, item_val, reason, mod)
	}

	return
}

func (this *Player) AddResources(resources []int32, reason, mod string) {
	if resources == nil {
		return
	}
	for i := 0; i < len(resources)/2; i++ {
		this.AddItemResource(resources[2*i], resources[2*i+1], reason, mod)
	}
}

// 各个属性设置函数

// 玩家添加猫或者物品或建筑库
func (this *Player) AddObj(objcfgid, addnum int32, reason, mod string, bslience bool) int32 {
	new_num := this.AddItemResource(objcfgid, addnum, reason, mod)
	if new_num >= 0 {
		return new_num
	}

	cat_cfg := cat_table_mgr.Map[objcfgid]
	if nil != cat_cfg {
		if nil != this.AddCat(objcfgid, reason, mod, bslience) {
			return 1
		}
		return 0
	}

	building_cfg := building_table_mgr.Map[objcfgid]
	if nil != building_cfg {
		this.AddDepotBuilding(objcfgid, addnum, reason, mod, bslience)
		new_num, _ = this.db.BuildingDepots.GetNum(objcfgid)
	}

	return new_num
}

// 玩家猫
func (this *Player) AddCat(catcfgid int32, reason, mod string, bslience bool) *msg_client_message.CatInfo {
	return this.AddCatWithLevelStarSkill(catcfgid, 1, 1, 1, reason, mod, bslience)
}

func (this *Player) AddCatWithLevelStarSkill(cat_cid int32, level int32, star int32, skill_level int32, reason, mod string, bslience bool) *msg_client_message.CatInfo {
	catcfg := cat_table_mgr.Map[cat_cid]
	if nil == catcfg {
		log.Error("Player Addcat failed to find catcfg [%d]", cat_cid)
		return nil
	}

	next_cid := this.db.Info.IncbyNextCatId(1)
	new_cat_db := &dbPlayerCatData{}
	new_cat_db.CfgId = cat_cid
	new_cat_db.Id = next_cid
	new_cat_db.Exp = 0
	new_cat_db.Level = level
	new_cat_db.Star = star
	new_cat_db.SkillLevel = skill_level
	new_cat_db.Locked = 0

	var o bool
	// 总属性
	var all_ability int32
	o, all_ability = rand31n_from_range(catcfg.RangeMin, catcfg.RangeMax)
	if !o {
		log.Error("Player AddCat[%v] gen all_ability failed", cat_cid)
		return nil
	}

	// 产金权重
	var coin_ability_weight int32
	o, coin_ability_weight = rand31n_from_range(catcfg.CoinAbilityRangeMin, catcfg.CoinAbilityRangeMax)
	if !o {
		log.Error("Player AddCat[%v] gen coin_ability_weight failed", cat_cid)
		return nil
	}

	var explore_ability_weight int32
	o, explore_ability_weight = rand31n_from_range(catcfg.ExploreAbilityRangeMin, catcfg.ExploreAbilityRangeMax)
	if !o {
		log.Error("Player AddCat[%v] gen explore_ability failed", cat_cid)
		return nil
	}

	var match_ability_weight int32
	o, match_ability_weight = rand31n_from_range(catcfg.MatchAbilityRangeMin, catcfg.MatchAbilityRangeMax)
	if !o {
		log.Error("Player AddCat[%v] gen match_ability failed", cat_cid)
		return nil
	}

	total_weight := coin_ability_weight + explore_ability_weight + match_ability_weight
	new_cat_db.CoinAbility = GetRoundValue(float32(all_ability * coin_ability_weight / total_weight))
	new_cat_db.ExploreAbility = GetRoundValue(float32(all_ability * explore_ability_weight / total_weight))
	new_cat_db.MatchAbility = all_ability - (new_cat_db.CoinAbility + new_cat_db.ExploreAbility)
	this.db.Cats.Add(new_cat_db)

	// 更新猫状态变化
	if !this.item_cat_building_change_info.cat_add(this, next_cid) {
		return nil
	}

	msg := &msg_client_message.CatInfo{}
	msg.CatCfgId = cat_cid
	msg.Id = next_cid
	msg.Level = level
	msg.Star = star
	msg.SkillLevel = skill_level
	msg.Exp = 0
	msg.Locked = false
	msg.CoinAbility = new_cat_db.CoinAbility
	msg.ExploreAbility = new_cat_db.ExploreAbility
	msg.MatchAbility = new_cat_db.MatchAbility
	msg.State = CAT_STATE_NONE

	if !bslience {
		this.Send(uint16(msg_client_message.CatInfo_ProtoID), msg)
	}

	// task update
	this.TaskUpdate(tables.TASK_COMPLETE_TYPE_COLLECT_SSR, false, catcfg.Rarity, 1)

	// 图鉴
	this.AddHandbookItem(cat_cid)

	// 头像
	this.AddHead(catcfg.AvatarId)

	// update ranking list
	this.update_ouqi(next_cid)

	// 公告
	if catcfg.Rarity >= 4 {
		anouncement_mgr.PushNew(ANOUNCEMENT_TYPE_GET_SSR_CAT, true, this.Id, this.db.GetName(), this.db.GetLevel(), cat_cid, 0, 0, "")
	}

	return msg
}

func (this *Player) SubCat(cat_id int32, reason, mod string) bool {
	this.db.Cats.Remove(cat_id)
	// 更新猫状态变化
	if !this.item_cat_building_change_info.cat_remove(this, cat_id) {
		return false
	}

	// update ranking list
	/*if this.rpc_call_delete_rank(4, cat_id) == nil {
		log.Warn("Player[%v] delete cat[%v] to update ouqi ranking list failed")
	}*/
	return true
}

// 玩家经验
func (this *Player) AddExp(add_val int32, reason, mod string) (int32, int32) {
	old_lvl := this.db.GetLevel()
	if add_val < 0 {
		log.Error("Player AddExp add_val[%d] < 0 ", add_val)
		return old_lvl, this.db.Info.GetExp()
	}

	old_exp := this.db.Info.GetExp()
	cur_exp := old_exp + add_val
	if old_lvl < 1 {
		return -1, -1
	}
	cur_lvl := old_lvl
	if cur_lvl+1 <= player_level_table_mgr.MaxLevel {
		blvl_chg := false
		for i := cur_lvl; i < player_level_table_mgr.MaxLevel; i++ {
			next_exp := player_level_table_mgr.Array[i-1].MaxExp
			if cur_exp >= next_exp {
				cur_lvl = i + 1
				cur_exp = cur_exp - next_exp
				blvl_chg = true
			} else {
				break
			}
		}

		if blvl_chg {
			log.Info("玩家[%d] 升级了[%d]", this.Id, cur_lvl)
			this.db.SetLevel(cur_lvl)
			this.db.Info.SetExp(cur_exp)
		} else {
			this.db.Info.SetExp(cur_exp)
		}
	}

	this.b_base_prop_chg = true

	if cur_lvl > old_lvl {
		this.rpc_player_base_info_update()
	}
	return cur_lvl, cur_exp
}

// 玩家金币 ====================================

func (this *Player) GetGold() int32 {
	return this.db.Info.GetGold()
}

func (this *Player) AddGold(val int32, reason, mod string) int32 {
	if val < 0 {
		log.Error("Player AddGold %d", val)
		return this.db.Info.GetGold()
	}

	if this.db.Info.GetGold()+val < 0 {
		this.db.Info.SetGold(0x7fffffff)
		return 0x7fffffff
	}

	cur_coin := this.db.Info.IncbyGold(val)
	this.b_base_prop_chg = true
	return cur_coin
}

func (this *Player) SubGold(val int32, reason, mod string) int32 {
	if val < 0 {
		log.Error("Player SubGold %d", val)
		return this.db.Info.GetGold()
	}

	cur_coin := this.db.Info.IncbyGold(-val)

	//this.TaskAchieveOnConditionAdd(TASK_ACHIEVE_FINISH_COIN_COST, val)

	this.b_base_prop_chg = true
	return cur_coin
}

// 玩家钻石 ====================================

func (this *Player) GetDiamond() int32 {
	return this.db.Info.GetDiamond()
}

func (this *Player) SubDiamond(sub_val int32, reason, mod string) int32 {
	if sub_val < 0 {
		log.Error("Player SubDiamond sub_val[%d] < 0, reason[%s] mod[%s]", sub_val, reason, mod)
		return this.db.Info.GetDiamond()
	}

	cur_diamond := this.db.Info.SubDiamond(sub_val)

	//this.TaskAchieveOnConditionAdd(TASK_ACHIEVE_FINISH_DIAMOND_COST, sub_val)

	this.b_base_prop_chg = true
	return cur_diamond
}

func (this *Player) AddDiamond(add_val int32, reason, mod string) int32 {
	if add_val < 0 {
		log.Error("Player AddDiamod add_val[%d] < 0, reason[%s] mod[%s]", add_val, reason, mod)
		return this.db.Info.GetDiamond()
	}

	if this.db.Info.GetDiamond()+add_val < 0 {
		this.db.Info.SetDiamond(0x7fffffff)
		return 0x7fffffff
	}

	this.b_base_prop_chg = true
	return this.db.Info.IncbyDiamond(add_val)
}

// 玩家魅力 =====================================

func (this *Player) SubCharmVal(sub_val int32, reason, mod string) int32 {
	if sub_val < 0 {
		log.Error("Player SubCharamVal sub_val(%d) < 0 reason(%s) mod(%s)", sub_val, reason, mod)
		return this.db.Info.GetCharmVal()
	}

	cur_charmval := this.db.Info.IncbyCharmVal(-sub_val)
	this.b_base_prop_chg = true

	// update ranking list
	if this.rpc_rank_list_update_data(common.RANK_LIST_TYPE_CHARM, []int32{cur_charmval}) == nil {
		log.Warn("Player[%v] update charm[%v] rank list failed", this.Id, cur_charmval)
	}

	return cur_charmval
}

func (this *Player) AddCharmVal(add_val int32, reason, mod string) int32 {
	if add_val < 0 {
		log.Error("Player AddCharmVal add_val(%d)< 0 reason(%s) mod(%s)", add_val, reason, mod)
		return this.db.Info.GetCharmVal()
	}

	if this.db.Info.GetCharmVal()+add_val < 0 {
		this.db.Info.SetCharmVal(0x7fffffff)
		return 0x7fffffff
	}

	cur_charmval := this.db.Info.IncbyCharmVal(add_val)
	this.b_base_prop_chg = true

	// update task
	this.TaskUpdate(tables.TASK_COMPLETE_TYPE_CHARM_VALUE, false, 0, cur_charmval)

	// update ranking list
	if this.rpc_rank_list_update_data(common.RANK_LIST_TYPE_CHARM, []int32{cur_charmval}) == nil {
		log.Warn("Player[%v] update charm[%v] rank list failed", this.Id, cur_charmval)
	}

	return cur_charmval
}

// 玩家友情点 ====================================
func (this *Player) SubFriendPoints(sub_val int32, reason, mod string) int32 {
	if sub_val < 0 {
		log.Error("Player SubFriendPoints sub_val(%v) < 0 reason(%s) mod(%s)", sub_val, reason, mod)
		return this.db.Info.GetFriendPoints()
	}

	cur_friendpoints := this.db.Info.IncbyFriendPoints(-sub_val)
	this.b_base_prop_chg = true
	return cur_friendpoints
}

func (this *Player) AddFriendPoints(add_val int32, reason, mod string) int32 {
	if add_val < 0 {
		log.Error("Player AddFriendPoints add_val(%d) < 0 reason(%s) mod(%s)", add_val, reason, mod)
		return this.db.Info.GetFriendPoints()
	}

	if this.db.Info.GetFriendPoints()+add_val < 0 {
		this.db.Info.SetFriendPoints(0x7fffffff)
		return 0x7fffffff
	}

	cur_friendpoints := this.db.Info.IncbyFriendPoints(add_val)
	this.b_base_prop_chg = true
	return cur_friendpoints
}

// 玩家体力 =====================================
func (this *Player) AddSpirit(spirit int32, reason, mod string) int32 {
	this.CalcSpirit()
	if spirit < 0 {
		log.Error("Player AddSpirit spirit(%v) < 0  reason(%v) mod(%s)", spirit, reason, mod)
		return this.db.Info.GetSpirit()
	}
	if this.db.Info.GetSpirit()+spirit < 0 {
		this.db.Info.SetSpirit(0x7fffffff)
		return 0x7fffffff
	}
	cur_spirit := this.db.Info.IncbySpirit(spirit)
	this.b_base_prop_chg = true
	return cur_spirit
}

func (this *Player) SubSpirit(spirit int32, reason, mod string) int32 {
	this.CalcSpirit()
	if spirit < 0 {
		log.Error("Player SubSpirit spirit(%v) < 0  reason(%v) mod(%s)", spirit, reason, mod)
		return this.db.Info.GetSpirit()
	}
	cur_spirit := this.db.Info.IncbySpirit(-spirit)
	this.b_base_prop_chg = true
	return cur_spirit
}

// 玩家猫粮 =====================================
func (this *Player) AddCatFood(food int32, reason, mod string) int32 {
	if food < 0 {
		log.Error("Player AddCatFood food(%v) < 0  reason(%v) mod(%v)", food, reason, mod)
		return this.db.Info.GetCatFood()
	}
	if this.db.Info.GetCatFood()+food < 0 {
		this.db.Info.SetCatFood(0x7fffffff)
		return 0x7fffffff
	}
	curr_food := this.db.Info.IncbyCatFood(food)
	this.b_base_prop_chg = true
	return curr_food
}

func (this *Player) SubCatFood(food int32, reason, mod string) int32 {
	if food < 0 {
		log.Error("Player SubCatFood food(%v) < 0  reason(%v) mod(%v)", food, reason, mod)
		return this.db.Info.GetCatFood()
	}
	if food > this.db.Info.GetCatFood() {
		return this.db.Info.GetCatFood()
	}
	cur_food := this.db.Info.IncbyCatFood(-food)
	this.b_base_prop_chg = true
	return cur_food
}

// 玩家魂石 =====================================
func (this *Player) AddSoulStone(stone int32, reason, mod string) int32 {
	if stone < 0 {
		log.Error("Player AddSoulStone stone(%v) < 0  reason(%v) mod(%v)", stone, reason, mod)
		return this.db.Info.GetSoulStone()
	}
	if this.db.Info.GetSoulStone()+stone < 0 {
		this.db.Info.SetSoulStone(0x7fffffff)
		return 0x7fffffff
	}
	curr_stone := this.db.Info.IncbySoulStone(stone)
	this.b_base_prop_chg = true
	return curr_stone
}

func (this *Player) SubSoulStone(stone int32, reason, mod string) int32 {
	if stone < 0 {
		log.Error("Player SubSoulStone stone(%v) < 0  reason(%v) mod(%v)", stone, reason, mod)
		return this.db.Info.GetSoulStone()
	}

	curr_stone := this.db.Info.IncbySoulStone(-stone)
	this.b_base_prop_chg = true
	return curr_stone
}

// 玩家星数
func (this *Player) AddStar(star int32, reason, mod string) int32 {
	if star < 0 {
		log.Error("Player AddStar star(%v) < 0  reason(%v) mod(%v)", star, reason, mod)
		return this.db.Info.GetTotalStars()
	}
	curr_star := this.db.Info.IncbyTotalStars(star)
	this.b_base_prop_chg = true

	// update task
	this.TaskUpdate(tables.TASK_COMPLETE_TYPE_COLLECT_STAR_NUM, false, 0, curr_star)

	return curr_star
}

func (this *Player) SubStar(star int32, reason, mod string) int32 {
	if star < 0 {
		log.Error("Player SubStar star(%v) < 0  reason(%v) mod(%v)", star, reason, mod)
		return this.db.Info.GetTotalStars()
	}
	curr_star := this.db.Info.IncbyTotalStars(-star)
	this.b_base_prop_chg = true
	return curr_star
}

// 玩家赞数
func (this *Player) AddZan(zan int32, reason, mod string) int32 {
	if zan < 0 {
		log.Error("Player AddZan zan(%v) < 0  reason(%v) mod(%v)", zan, reason, mod)
		return this.db.Info.GetZan()
	}
	if this.db.Info.GetZan()+zan < 0 {
		this.db.Info.SetZan(0x7fffffff)
		return 0x7fffffff
	}
	cur_zan := this.db.Info.IncbyZan(zan)
	this.b_base_prop_chg = true
	return cur_zan
}

func (this *Player) SubZan(zan int32, reason, mod string) int32 {
	if zan < 0 {
		log.Error("Player SubZan zan(%v) < 0  reason(%v) mod(%v)", zan, reason, mod)
		return this.db.Info.GetZan()
	}
	cur_zan := this.db.Info.IncbyZan(-zan)
	this.b_base_prop_chg = true
	return cur_zan
}

// 玩家魅力勋章
func (this *Player) AddCharmMedal(charm_medal int32, reason, mod string) int32 {
	if charm_medal < 0 {
		log.Error("Player AddCharmMedal charm_medal(%v) < 0  reason(%v) mod(%v)", charm_medal, reason, mod)
		return this.db.Info.GetCharmMedal()
	}
	if this.db.Info.GetCharmMedal()+charm_medal < 0 {
		this.db.Info.SetCharmMedal(0x7fffffff)
		return 0x7fffffff
	}
	cur_medal := this.db.Info.IncbyCharmMedal(charm_medal)
	this.b_base_prop_chg = true
	return cur_medal
}

func (this *Player) SubCharmMedal(charm_medal int32, reason, mod string) int32 {
	if charm_medal < 0 {
		log.Error("Player SubCharmMedal charm_madal(%v) < 0  reason(%v) mod(%v)", charm_medal, reason, mod)
		return this.db.Info.GetCharmMedal()
	}

	cur_medal := this.db.Info.IncbyCharmMedal(-charm_medal)
	this.b_base_prop_chg = true
	return cur_medal
}

// 玩家物品
func (this *Player) AddItem(itemcfgid, addnum int32, reason, mod string, bslience bool) *msg_client_message.ItemInfo {
	itemcfg := item_table_mgr.Map[itemcfgid]
	if nil == itemcfg {
		log.Error("Player AddItem failed to find itemcfg[%v] reason[%v] mod[%v]", itemcfgid, reason, mod)
		return nil
	}

	new_num := this.db.Items.ChkAddItemByNum(itemcfgid, addnum)

	// 更新物品变化状态
	this.item_cat_building_change_info.item_update(this, itemcfgid)

	msg := &msg_client_message.ItemInfo{}
	msg.ItemCfgId = itemcfgid
	msg.ItemNum = new_num
	if !bslience {
		this.Send(uint16(msg_client_message.ItemInfo_ProtoID), msg)
	}

	// 公告寄养卡
	foster := foster_table_mgr.Get(itemcfgid)
	if foster != nil && foster.Rarity >= 4 {
		anouncement_mgr.PushNew(ANOUNCEMENT_TYPE_GET_FOSTER_CARD, true, this.Id, this.db.GetName(), this.db.GetLevel(), itemcfgid, 0, 0, "")
	}

	return msg
}

func (this *Player) RemoveItem(cfgid, num int32, bsilence bool) *msg_client_message.ItemInfo {
	item := item_table_mgr.Map[cfgid]
	if item == nil {
		log.Error("Not found item[%v] in config", cfgid)
		return nil
	}
	o, _ := this.db.Items.ChkRemoveItem(cfgid, num)
	if !o {
		log.Error("remove item[%v,%v] for player[%v] failed", cfgid, num, this.Id)
		return nil
	}

	// 更新物品变化状态
	this.item_cat_building_change_info.item_update(this, cfgid)

	msg := &msg_client_message.ItemInfo{}
	msg.ItemCfgId = cfgid
	msg.ItemNum = num
	if !bsilence {
		this.Send(uint16(msg_client_message.ItemInfo_ProtoID), msg)
	}
	return msg
}

func (this *Player) RemoveItemAll(item_id int32, silence bool) {
	n, o := this.db.Items.GetItemNum(item_id)
	if !o {
		return
	}
	this.db.Items.Remove(item_id)

	// 更新物品变化状态
	this.item_cat_building_change_info.item_update(this, item_id)

	msg := &msg_client_message.ItemInfo{}
	msg.ItemCfgId = item_id
	msg.ItemNum = n
	if !silence {
		this.Send(uint16(msg_client_message.ItemInfo_ProtoID), msg)
	}
}

func (this *Player) add_handbook_data(item_id int32) {
	var d dbPlayerHandbookItemData
	d.Id = item_id
	this.db.HandbookItems.Add(&d)

	msg := &msg_client_message.S2CNewHandbookItemNotify{}
	msg.ItemId = item_id
	this.Send(uint16(msg_client_message.S2CNewHandbookItemNotify_ProtoID), msg)
}

func (this *Player) AddHandbookItem(item_id int32) {
	if handbook_table_mgr.Get(item_id) == nil {
		return
	}
	if this.db.HandbookItems.HasIndex(item_id) {
		return
	}

	this.add_handbook_data(item_id)

	// 是否为建筑
	/*building := cfg_building_mgr.Map[item_id]
	if building != nil {
		if building.SuitId > 0 && suit_table_mgr.Map[building.SuitId] != nil {
			suits := cfg_building_mgr.Suits[building.SuitId]
			if suits != nil {
				c := true
				for _, v := range suits.Items {
					if !this.db.HandbookItems.HasIndex(v) {
						c = false
						break
					}
				}
				if c {
					this.add_handbook_data(building.SuitId)
				}
			}
		}
	}*/
}

func (this *Player) AddHead(item_id int32) {
	if this.db.HeadItems.HasIndex(item_id) {
		return
	}
	var d dbPlayerHeadItemData
	d.Id = item_id
	this.db.HeadItems.Add(&d)
	msg := &msg_client_message.S2CNewHeadNotify{}
	msg.ItemId = item_id
	this.Send(uint16(msg_client_message.S2CNewHeadNotify_ProtoID), msg)
}

func (this *Player) use_item(item_id int32, item_count int32) int32 {
	if item_count <= 0 {
		return -1
	}

	item := item_table_mgr.Map[item_id]
	if item == nil {
		log.Error("没有ID为%v的物品配置", item_id)
		return -1
	}

	num, o := this.db.Items.GetItemNum(item_id)
	if !o {
		log.Error("没有物品[%v]", item_id)
		return -1
	}

	if num < item_count {
		log.Error("物品[%v]数量[%v]不够", item_id, item_count)
		return -1
	}

	// 先判断是否为限时道具
	if item.ValidTime > 0 {
		item_data := this.db.Items.Get(item_id)
		if item_data != nil {
			if get_time_item_remain_seconds(item_data) == 0 {
				log.Error("玩家[%v]限时道具[%v]已过期", this.Id, item_id)
				return -1
			}
		}
	}

	// 体力道具
	if item.Type == ITEM_TYPE_SPIRIT {
		if len(item.Numbers) < 2 {
			log.Error("物品[%v]数据配置错误", item_id)
			return -1
		}
		//this.AddSpirit(item.Numbers[1], "use_spirit_item", "use_item")
		this.RemoveItem(item_id, item_count, false)
		this.AddItemResource(item.Numbers[0], item.Numbers[1]*item_count, "use_spirit_item", "use_item")
	}

	// 发送物品变化
	this.item_cat_building_change_info.send_items_update(this)

	msg := &msg_client_message.S2CUseItem{}
	msg.CostItem = &msg_client_message.ItemInfo{}
	msg.CostItem.ItemCfgId = item_id
	msg.CostItem.ItemNum = item_count
	this.Send(uint16(msg_client_message.S2CUseItem_ProtoID), msg)

	return 1
}

func (this *Player) sell_item(item_id int32, item_count int32) int32 {
	item := item_table_mgr.Map[item_id]
	if item == nil {
		log.Error("没有ID为%v的物品", item_id)
		return -1
	}

	if this.RemoveItem(item_id, item_count, false) == nil {
		return -1
	}

	// 发送物品变化
	this.item_cat_building_change_info.send_items_update(this)

	this.AddGold(item.SaleCoin*item_count, "sell item", "item")

	msg := &msg_client_message.S2CSellItemResult{}
	msg.ItemId = item_id
	msg.ItemNum = item_count
	this.Send(uint16(msg_client_message.S2CSellItemResult_ProtoID), msg)

	return 1
}

func (this *Player) ChkItemsEnough(itemidnums []int32) bool {
	tmp_len := int32(len(itemidnums))
	var item_id, item_num, db_num int32
	for idx := int32(0); idx < tmp_len; idx += 2 {
		item_id = itemidnums[idx]
		item_num = itemidnums[idx+1]
		db_num, _ = this.db.Items.GetItemNum(item_id)
		if db_num < item_num {
			return false
		}
	}
	return true
}

func (this *Player) RemoveItems(itemidnums []int32, reason, mod string) {
	tmp_len := int32(len(itemidnums))
	var item_id, item_num int32
	for idx := int32(0); idx < tmp_len; idx += 2 {
		item_id = itemidnums[idx]
		item_num = itemidnums[idx+1]
		this.RemoveItem(item_id, item_num, true)
	}
	return
}

func (this *Player) compose_cat(cat_id int32) int32 {
	cat := cat_table_mgr.Map[cat_id]
	if cat == nil {
		log.Error("没有配置ID为[%v]的猫", cat_id)
		return -1
	}

	piece_item := this.db.Items.Get(cat.PieceId)
	if piece_item == nil {
		log.Error("没有碎片物品[%v]", cat.PieceId)
		return -1
	}

	if piece_item.ItemNum < cat.PieceNum {
		log.Error("物品碎片[%v]数量[%v]不足，合成失败", cat.PieceId, cat.PieceNum)
		return -1
	}

	this.RemoveItem(cat.PieceId, cat.PieceNum, true)
	cat_add := this.AddCat(cat.Id, "compose", "item", true)
	if cat_add == nil {
		log.Error("合成添加猫[%v]失败", cat_id)
		return -1
	}

	// 发送物品变化
	this.item_cat_building_change_info.send_items_update(this)
	// 发送猫变化
	this.item_cat_building_change_info.send_cats_update(this)

	response := &msg_client_message.S2CComposeCatResult{}
	response.Cat = cat_add
	response.UsedFragment = &msg_client_message.ItemInfo{}
	response.UsedFragment.ItemCfgId = cat.PieceId
	response.UsedFragment.ItemNum = -cat.PieceNum

	this.Send(uint16(msg_client_message.S2CComposeCatResult_ProtoID), response)

	return 1
}

func (this *Player) is_today_zan(player_id int32, now_time time.Time) bool {
	zan_time, o := this.db.Zans.GetZanTime(player_id)
	if !o {
		return false
	}

	tt := time.Unix(int64(zan_time), 0)

	if tt.Year() != now_time.Year() || tt.YearDay() != now_time.YearDay() {
		return false
	}

	return true
}

func (p *Player) zan_player(player_id int32) int32 {
	now_time := time.Now()
	o := p.db.Zans.HasIndex(player_id)
	if o {
		if p.is_today_zan(player_id, now_time) {
			log.Warn("Player[%v] zan player[%v] today yet", p.Id, player_id)
			return int32(msg_client_message.E_ERR_PLAYER_ALREADY_ZAN_TODAY)
		}
		p.db.Zans.IncbyZanNum(player_id, 1)
		p.db.Zans.SetZanTime(player_id, int32(now_time.Unix()))
	} else {
		d := &dbPlayerZanData{
			PlayerId: player_id,
			ZanTime:  int32(now_time.Unix()),
			ZanNum:   1,
		}
		p.db.Zans.Add(d)
	}
	return 1
}
