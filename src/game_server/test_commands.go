package main

import (
	"mm_server/libs/log"
	"mm_server/libs/utils"
	"mm_server/proto/gen_go/client_message"
	"mm_server/src/common"
	"strconv"

	"github.com/golang/protobuf/proto"
)

func player_info_cmd(p *Player, args []string) int32 {
	log.Info("### 玩家基础信息如下：")
	log.Info("###### Name: %v", p.db.GetName())
	log.Info("###### Level: %v", p.db.GetLevel())
	log.Info("###### Exp: %v", p.db.Info.GetExp())
	log.Info("###### Diamond: %v", p.db.Info.GetDiamond())
	log.Info("###### Gold: %v", p.db.Info.GetGold())
	log.Info("###### CharmVal: %v", p.db.Info.GetCharmVal())
	log.Info("###### CharmMedal: %v", p.db.Info.GetCharmMedal())
	log.Info("###### MaxUnlockStage: %v", p.db.Info.GetMaxUnlockStage())
	log.Info("###### CurMaxStage: %v", p.db.Info.GetCurMaxStage())
	log.Info("###### Star: %v", p.db.Info.GetTotalStars())
	log.Info("###### FriendPoints: %v", p.db.Info.GetFriendPoints())
	log.Info("###### Zan: %v", p.db.Info.GetZan())
	log.Info("###### CatFood: %v", p.db.Info.GetCatFood())
	log.Info("###### SoulStone: %v", p.db.Info.GetSoulStone())
	log.Info("###### Spirit: %v", p.CalcSpirit())
	return 0
}

func add_exp_cmd(p *Player, args []string) int32 {
	if len(args) < 1 {
		log.Error("参数数量[%v]不够", len(args))
		return -1
	}

	var exp int
	var err error
	exp, err = strconv.Atoi(args[0])
	if err != nil {
		log.Error("经验[%v]转换失败[%v]", exp, err.Error())
		return -1
	}

	p.AddExp(int32(exp), "test_add_exp", "test_command")
	return 1
}

func set_level_cmd(p *Player, args []string) int32 {
	if len(args) < 1 {
		log.Error("参数数量[%v]不够", len(args))
		return -1
	}

	var level int
	var err error
	level, err = strconv.Atoi(args[0])
	if err != nil {
		return -1
	}

	p.db.SetLevel(int32(level))
	p.rpc_player_base_info_update()
	p.send_info()
	return 1
}

func add_item_cmd(p *Player, args []string) int32 {
	if len(args) < 2 {
		log.Error("参数数量[%v]不够", len(args))
		return -1
	}

	item_id, err := strconv.Atoi(args[0])
	if err != nil {
		log.Error("物品ID[%v]转换失败[%v]", args[0], err.Error())
		return -1
	}

	item := item_table_mgr.Map[int32(item_id)]
	if item == nil {
		log.Error("没有物品[%v]配置", item_id)
		return -1
	}

	item_count, err2 := strconv.Atoi(args[1])
	if err2 != nil {
		log.Error("物品数量[%v]转换失败[%v]", args[1], err2.Error())
		return -1
	}

	if item_count < 0 {
		p.RemoveItem(int32(item_id), int32(item_count), false)
	} else {
		p.AddItem(int32(item_id), int32(item_count), "test_add_item", "test_command", false)
	}
	p.SendItemsUpdate()
	return 1
}

func add_all_item_cmd(p *Player, args []string) int32 {
	p.add_all_items()
	p.SendItemsUpdate()
	return 1
}

func use_item_cmd(p *Player, args []string) int32 {
	if len(args) < 1 {
		log.Error("参数数量[%v]不够", len(args))
		return -1
	}

	var item_id int
	var err error
	item_id, err = strconv.Atoi(args[0])
	if err != nil {
		log.Error("物品ID[%v]转换失败[%v]", args[0], err.Error())
		return -1
	}

	item := item_table_mgr.Map[int32(item_id)]
	if item == nil {
		log.Error("没有物品[%v]配置", item_id)
		return -1
	}

	item_num := 1
	if len(args) > 1 {
		item_num, err = strconv.Atoi(args[1])
		if err != nil {
			log.Error("物品数量[%v]转换失败[%v]", args[1], err.Error())
			return -1
		}
	}
	return p.use_item(int32(item_id), int32(item_num))
}

func list_item_cmd(p *Player, args []string) int32 {
	ids := p.db.Items.GetAllIndex()
	if ids == nil || len(ids) == 0 {
		log.Warn("玩家[%v]没有物品", p.Id)
		return -1
	}
	log.Info("@@@ 玩家[%v]物品列表如下：", p.Id)
	for i, id := range ids {
		item_data := p.db.Items.Get(id)
		if item_data == nil {
			log.Warn("玩家[%v]没有物品[%v]", p.Id, id)
			continue
		}
		log.Info("@@@@@@ [%v] Id[%v] CfgId[%v] Num[%v] RemainSeconds[%v]", i, id, item_data.ItemCfgId, item_data.ItemNum, item_data.RemainSeconds)
	}
	return 0
}

func add_coin_cmd(p *Player, args []string) int32 {
	if len(args) < 1 {
		log.Error("参数[%v]不够", len(args))
		return -1
	}
	coin, err := strconv.Atoi(args[0])
	if err != nil {
		log.Error("金币数量[%v]转换失败[%v]", args[0], err.Error())
		return -1
	}

	p.AddGold(int32(coin), "test_add_coin", "test_command")
	return 1
}

func set_coin_cmd(p *Player, args []string) int32 {
	if len(args) < 1 {
		log.Error("参数[%v]不够", len(args))
		return -1
	}
	coin, err := strconv.Atoi(args[0])
	if err != nil {
		log.Error("金币数量[%v]转换失败[%v]", args[0], err.Error())
		return -1
	}
	if coin < 0 {
		return -1
	}
	p.db.Info.SetGold(int32(coin))
	return 1
}

func add_diamond_cmd(p *Player, args []string) int32 {
	if len(args) < 1 {
		log.Error("参数[%v]不够", len(args))
		return -1
	}
	diamond, err := strconv.Atoi(args[0])
	if err != nil {
		log.Error("钻石数量[%v]转换失败[%v]", args[0], err.Error())
		return -1
	}
	p.AddDiamond(int32(diamond), "test_add_diamond", "test_command")
	return 1
}

func set_diamond_cmd(p *Player, args []string) int32 {
	if len(args) < 1 {
		log.Error("参数[%v]不够", len(args))
		return -1
	}
	diamond, err := strconv.Atoi(args[0])
	if err != nil {
		log.Error("钻石数量[%v]转换失败[%v]", args[0], err.Error())
		return -1
	}
	if diamond < 0 {
		return -1
	}
	p.db.Info.SetDiamond(int32(diamond))
	return 1
}

func add_cat_cmd(p *Player, args []string) int32 {
	if len(args) < 2 {
		log.Error("参数[%v]不够", len(args))
		return -1
	}
	var cat_cid, num int
	var err error
	cat_cid, err = strconv.Atoi(args[0])
	if err != nil {
		log.Error("增加猫配置ID[%v]转换失败[%v]", args[0], err.Error())
		return -1
	}

	num, err = strconv.Atoi(args[1])
	if err != nil {
		log.Error("增加猫数量[%v]转换失败[%v]", args[1], err.Error())
		return -1
	}

	for i := 0; i < num; i++ {
		p.AddCat(int32(cat_cid), "add_cat_cmd", "test_command", true)
	}

	p.item_cat_building_change_info.send_cats_update(p)

	return 1
}

func add_cat_with_level_star_skill_cmd(p *Player, args []string) int32 {
	if len(args) < 1 {
		log.Error("参数[%v]不够", len(args))
		return -1
	}
	var cat_cid, level, star, skill_level, num int
	var err error
	cat_cid, err = strconv.Atoi(args[0])
	if err != nil {
		log.Error("增加猫ID[%v]转换失败[%v]", args[0], err.Error())
		return -1
	}

	cat := cat_table_mgr.GetCat(int32(cat_cid))
	if cat == nil {
		log.Error("增加猫的配置ID[%v]不合法", cat_cid)
		return -1
	}

	if len(args) > 1 {
		level, err = strconv.Atoi(args[1])
		if err != nil {
			log.Error("增加猫等级[%v]转换失败[%v]", args[1], err.Error())
			return -1
		}
	}

	if cat.GetMaxLevel() < int32(level) {
		log.Error("猫的等级不能超过[%v]级", level)
		return -1
	}

	star = 1
	if len(args) > 2 {
		star, err = strconv.Atoi(args[2])
		if err != nil {
			log.Error("增加猫星级[%v]转换失败[%v]", args[2], err.Error())
			return -1
		}
	}

	if cat.GetMaxStar() < int32(star) {
		log.Error("猫的星级不能超过[%v]级", star)
		return -1
	}

	skill_level = 1
	if len(args) > 3 {
		skill_level, err = strconv.Atoi(args[3])
		if err != nil {
			log.Error("增加猫技能等级[%v]转换失败[%v]", args[3], err.Error())
			return -1
		}
	}

	if cat.GetMaxSkillLevel() < int32(skill_level) {
		log.Error("猫的技能等级不能超过[%v]级", skill_level)
		return -1
	}

	num = 1
	if len(args) > 4 {
		num, err = strconv.Atoi(args[4])
		if err != nil {
			log.Error("增加猫数量[%v]转换失败[%v]", args[4], err.Error())
			return -1
		}
	}

	for i := 0; i < num; i++ {
		p.AddCatWithLevelStarSkill(int32(cat_cid), int32(level), int32(star), int32(skill_level), "add_cat_with_level_star_skill_cmd", "test_command", false)
	}

	p.item_cat_building_change_info.send_cats_update(p)

	return 1
}

func list_cat_cmd(p *Player, args []string) int32 {
	ids := p.db.Cats.GetAllIndex()
	if ids == nil || len(ids) == 0 {
		log.Error("玩家[%v]没有猫", p.Id)
		return -1
	}
	log.Info("+++玩家[%v]猫列表", p.Id)
	for i, id := range ids {
		cat_data := p.db.Cats.Get(id)
		if cat_data == nil {
			continue
		}
		log.Info("+++++ [%v] CfgId[%v] Id[%v] Level[%v] Exp[%v] Star[%v] SkillLevel[%v] Nick[%v], CoinAbility[%v], ExploreAbility[%v], MatchAbility[%v]",
			i, cat_data.CfgId, cat_data.Id, cat_data.Level, cat_data.Exp, cat_data.Star, cat_data.SkillLevel, cat_data.Nick, cat_data.CoinAbility, cat_data.ExploreAbility, cat_data.MatchAbility)
	}
	return 0
}

func draw_card_cmd(p *Player, args []string) int32 {
	if len(args) < 2 {
		log.Error("参数[%v]不够", len(args))
		return -1
	}
	draw_type, err := strconv.Atoi(args[0])
	if err != nil {
		log.Error("抽奖类型[%v]转换失败[%v]", args[0], err.Error())
		return -1
	}
	draw_num, err2 := strconv.Atoi(args[1])
	if err2 != nil {
		log.Error("抽奖次数[%v]转换失败[%v]", args[1], err.Error())
		return -1
	}
	return p.DrawCard(int32(draw_type), int32(draw_num))
}

func drop_items_cmd(p *Player, args []string) int32 {
	if len(args)%2 != 0 {
		log.Error("参数[%v]不够", len(args))
		return -1
	}

	var err error
	var n int
	a := make([]int32, len(args))
	for i := 0; i < len(args); i++ {
		n, err = strconv.Atoi(args[i])
		if err != nil {
			log.Error("掉落参数[%v]错误[%v]", args[i], err.Error())
			return -1
		}
		a[i] = int32(n)
	}

	b, items, cats, buildings := p.DropItems2([]int32(a), true)
	if !b {
		return -1
	}
	log.Debug("@@@@@@ droped items[%v], cats[%v], buildings[%v]", items, cats, buildings)
	return 1
}

func compose_cat_cmd(p *Player, args []string) int32 {
	if len(args) < 1 {
		log.Error("参数[%v]不够", len(args))
		return -1
	}

	cat_config_id, err := strconv.Atoi(args[0])
	if err != nil {
		log.Error("猫配置ID[%v]转换失败[%v]", args[0], err.Error())
		return -1
	}
	return p.compose_cat(int32(cat_config_id))
}

func get_shop_items_cmd(p *Player, args []string) int32 {
	if len(args) < 1 {
		log.Error("参数[%v]不够", len(args))
		return -1
	}

	shop_id, err := strconv.Atoi(args[0])
	if err != nil {
		log.Error("商店配置ID[%v]转换失败[%v]", args[0], err.Error())
		return -1
	}
	/*if p.check_shop_refresh(false) {
		log.Info("商店[%v]刷新", shop_id)
	}*/
	return p.fetch_shop_limit_items(int32(shop_id), true)
}

func buy_shop_item_cmd(p *Player, args []string) int32 {
	if len(args) < 1 {
		log.Error("参数[%v]不够", len(args))
		return -1
	}

	item_id, err := strconv.Atoi(args[0])
	if err != nil {
		log.Error("商品配置[%v]转换失败[%v]", args[0], err.Error())
		return -1
	}

	if p.check_shop_limited_days_items_refresh_by_shop_itemid(int32(item_id), true) {
		log.Info("商店刷新")
	}

	return p.buy_item(int32(item_id), 1, true)
}

func sell_item_cmd(p *Player, args []string) int32 {
	if len(args) < 2 {
		log.Error("参数[%v]不够", len(args))
		return -1
	}

	var item_id, item_num int
	var err error
	item_id, err = strconv.Atoi(args[0])
	if err != nil {
		log.Error("物品ID[%v]转换失败[%v]", args[0], err.Error())
		return -1
	}

	item_num, err = strconv.Atoi(args[1])
	if err != nil {
		log.Error("物品[%v]数量转换失败[%v]", args[1], err.Error())
		return -1
	}

	return p.sell_item(int32(item_id), int32(item_num))
}

func refresh_shop_cmd(p *Player, args []string) int32 {
	if p.check_all_shop_items_refresh(true) {
		return 1
	}
	return -1
}

func cat_feed_cmd(p *Player, args []string) int32 {
	if len(args) < 2 {
		log.Error("参数[%v]不够", len(args))
		return -1
	}

	cat_id, err := strconv.Atoi(args[0])
	if err != nil {
		log.Error("猫id[%v]转换失败[%v]", args[0], err.Error())
		return -1
	}

	food, err2 := strconv.Atoi(args[1])
	if err2 != nil {
		log.Error("猫粮[%v]转换失败[%v]", args[0], err2.Error())
		return -1
	}

	_, _, res := p.feed_cat(int32(cat_id), int32(food), int32(food), false)
	return res
}

func cat_upstar_cmd(p *Player, args []string) int32 {
	if len(args) < 2 {
		log.Error("参数[%v]不够", len(args))
		return -1
	}

	var cat_id int
	var err error
	cat_id, err = strconv.Atoi(args[0])
	if err != nil {
		log.Error("猫ID[%v]转换失败[%v]", args[0], err.Error())
		return -1
	}
	cost_cats_len := len(args) - 1
	if cost_cats_len == 0 {
		log.Error("消耗猫的数量不能为0")
		return -1
	}
	cost_cat_ids := make([]int32, cost_cats_len)
	for i, a := range args[1:] {
		var cost_cat_id int
		cost_cat_id, err = strconv.Atoi(a)
		if err != nil {
			log.Error("升星消耗猫ID[%v]转换失败[%v]", a, err.Error())
			return -1
		}
		cost_cat_ids[i] = int32(cost_cat_id)
	}
	return p.cat_upstar(int32(cat_id), cost_cat_ids)
}

func cat_upskill_cmd(p *Player, args []string) int32 {
	if len(args) < 2 {
		log.Error("参数[%v]不够", len(args))
		return -1
	}

	var cat_id, cost_cat_id int
	var err error
	cat_id, err = strconv.Atoi(args[0])
	if err != nil {
		log.Error("猫ID[%v]转换失败[%v]", args[0], err.Error())
		return -1
	}
	cost_cat_id, err = strconv.Atoi(args[1])
	if err != nil {
		log.Error("消耗猫ID[%v]转换失败[%v]", args[1], err.Error())
		return -1
	}
	return p.cat_skill_levelup(int32(cat_id), []int32{int32(cost_cat_id)})
}

func add_cat_food_cmd(p *Player, args []string) int32 {
	if len(args) < 1 {
		log.Error("参数[%v]不够", len(args))
		return -1
	}

	var food int
	var err error
	food, err = strconv.Atoi(args[0])
	if err != nil {
		log.Error("猫粮[%v]转换失败[%v]", args[0], err.Error())
		return -1
	}

	return p.AddCatFood(int32(food), "test_add_cat_food", "test")
}

func add_friend_points_cmd(p *Player, args []string) int32 {
	if len(args) < 1 {
		log.Error("参数[%v]不够", len(args))
		return -1
	}

	var friend_points int
	var err error
	friend_points, err = strconv.Atoi(args[0])
	if err != nil {
		log.Error("友情点[%v]转换失败[%v]", args[0], err.Error())
		return -1
	}

	return p.AddFriendPoints(int32(friend_points), "test_add_friend_points", "test")
}

func add_soul_stone_cmd(p *Player, args []string) int32 {
	if len(args) < 1 {
		log.Error("参数[%v]不够", len(args))
		return -1
	}

	var soul_stone int
	var err error
	soul_stone, err = strconv.Atoi(args[0])
	if err != nil {
		log.Error("魂石[%v]转换失败[%v]", args[0], err.Error())
		return -1
	}

	return p.AddSoulStone(int32(soul_stone), "test_add_soulstone", "test")
}

func add_charm_cmd(p *Player, args []string) int32 {
	if len(args) < 1 {
		log.Error("参数[%v]不够", len(args))
		return -1
	}

	var charm int
	var err error
	charm, err = strconv.Atoi(args[0])
	if err != nil {
		log.Error("魅力值[%v]转换失败[%v]", args[0], err.Error())
		return -1
	}

	if charm < 0 {
		return p.SubCharmVal(int32(-charm), "test_add_charm", "test")
	} else {
		return p.AddCharmVal(int32(charm), "test_add_charm", "test")
	}
}

func add_charm_medal_cmd(p *Player, args []string) int32 {
	if len(args) < 1 {
		log.Error("参数[%v]不够", len(args))
		return -1
	}

	medal, err := strconv.Atoi(args[0])
	if err != nil {
		log.Error("魅力勋章[%v]转换失败[%v]", args[0], err.Error())
		return -1
	}

	return p.AddCharmMedal(int32(medal), "test_add_charm_medal", "test")
}

func add_zan_cmd(p *Player, args []string) int32 {
	if len(args) < 1 {
		log.Error("参数[%v]不够", len(args))
		return -1
	}

	zan, err := strconv.Atoi(args[0])
	if err != nil {
		log.Error("赞[%v]转换失败[%v]", args[0], err.Error())
		return -1
	}

	return p.AddZan(int32(zan), "test_add_charm_medal", "test")
}

func add_star_cmd(p *Player, args []string) int32 {
	if len(args) < 1 {
		log.Error("参数[%v]不够", len(args))
		return -1
	}

	star, err := strconv.Atoi(args[0])
	if err != nil {
		log.Error("星星数[%v]转换失败[%v]", args[0], err.Error())
		return -1
	}

	return p.AddStar(int32(star), "test_add_star", "test")
}

func see_cat_cmd(p *Player, args []string) int32 {
	if len(args) < 1 {
		log.Error("参数[%v]不够", len(args))
		return -1
	}

	var cat_id int
	var err error
	cat_id, err = strconv.Atoi(args[0])
	if err != nil {
		log.Error("猫ID[%v]转换失败[%v]", args[0], err.Error())
		return -1
	}

	cat_data := p.db.Cats.Get(int32(cat_id))
	if cat_data == nil {
		log.Error("玩家[%v]没有猫[%v]", p.Id, cat_id)
		return -1
	}

	log.Info("!!! 玩家[%v]猫[%v]数据: CfgId[%v] Level[%v] Exp[%v] Star[%v] SkillLevel[%v] Nick[%v] IsLock[%v]",
		p.Id, cat_id, cat_data.CfgId, cat_data.Level, cat_data.Exp, cat_data.Star, cat_data.SkillLevel, cat_data.Nick, cat_data.Locked)
	return 0
}

func get_formulas_cmd(p *Player, args []string) int32 {
	return p.get_formulas()
}

func get_making_buildings_cmd(p *Player, args []string) int32 {
	return p.pull_formula_building()
}

func exchange_formulas_cmd(p *Player, args []string) int32 {
	if len(args) < 1 {
		log.Error("参数[%v]不够", len(args))
		return -1
	}

	var formula_id int
	var err error
	formula_id, err = strconv.Atoi(args[0])
	if err != nil {
		log.Error("配方ID[%v]转换失败[%v]", args[0], err.Error())
		return -1
	}

	return p.exchange_formula(int32(formula_id))
}

func making_formula_building_cmd(p *Player, args []string) int32 {
	if len(args) < 1 {
		log.Error("参数[%v]不够", len(args))
		return -1
	}

	var /*slot_id, */ formula_id int
	var err error
	/*slot_id, err = strconv.Atoi(args[0])
	if err != nil {
		log.Error("配方槽位[%v]转换失败[%v]", args[0], err.Error())
		return -1
	}*/

	formula_id, err = strconv.Atoi(args[0])
	if err != nil {
		log.Error("配方ID[%v]转换失败[%v]", args[0], err.Error())
		return -1
	}

	return p.make_formula_building(int32(formula_id) /*, int32(slot_id)*/)
}

func buy_formula_slot_cmd(p *Player, args []string) int32 {
	return p.buy_new_making_building_slot()
}

func speedup_making_formula_building_cmd(p *Player, args []string) int32 {
	/*if len(args) < 1 {
		log.Error("参数[%v]不够", len(args))
		return -1
	}

	var slot_id int
	var err error
	slot_id, err = strconv.Atoi(args[0])
	if err != nil {
		log.Error("配方槽位[%v]转换失败[%v]", args[0], err.Error())
		return -1
	}*/

	return p.speedup_making_building( /*int32(slot_id)*/ )
}

func get_completed_formula_building_cmd(p *Player, args []string) int32 {
	/*if len(args) < 1 {
		log.Error("参数[%v]不够", len(args))
		return -1
	}

	var slot_id int
	var err error
	slot_id, err = strconv.Atoi(args[0])
	if err != nil {
		log.Error("配方槽位[%v]转换失败[%v]", args[0], err.Error())
		return -1
	}*/
	return p.get_completed_formula_building( /*int32(slot_id)*/ )
}

func cancel_making_formula_building_cmd(p *Player, args []string) int32 {
	if len(args) < 1 {
		log.Error("参数[%v]不够", len(args))
		return -1
	}

	var slot_id int
	var err error
	slot_id, err = strconv.Atoi(args[0])
	if err != nil {
		log.Error("配方槽位[%v]转换失败[%v]", args[0], err.Error())
		return -1
	}

	return p.cancel_making_formula_building(int32(slot_id))
}

func get_crops_cmd(p *Player, args []string) int32 {
	return p.get_crops()
}

func plant_crop_cmd(p *Player, args []string) int32 {
	if len(args) < 2 {
		log.Error("参数[%v]不够", len(args))
		return -1
	}

	var crop_id, building_id int
	var err error
	crop_id, err = strconv.Atoi(args[0])
	if err != nil {
		log.Error("作物[%v]转换失败[%v]", args[0], err.Error())
		return -1
	}

	building_id, err = strconv.Atoi(args[1])
	if err != nil {
		log.Error("农田ID[%v]转换失败[%v]", args[1], err.Error())
		return -1
	}

	return p.plant_crop(int32(crop_id), int32(building_id))
}

func speedup_crop_cmd(p *Player, args []string) int32 {
	if len(args) < 1 {
		log.Error("参数[%v]不够", len(args))
		return -1
	}

	var farm_building_id int
	var err error
	farm_building_id, err = strconv.Atoi(args[0])
	if err != nil {
		log.Error("农田ID[%v]转换失败[%v]", farm_building_id, err.Error())
		return -1
	}
	return p.speedup_crop(int32(farm_building_id))
}

func harvest_crop_cmd(p *Player, args []string) int32 {
	if len(args) < 1 {
		log.Error("参数[%v]不够", len(args))
		return -1
	}

	var building_id int
	var err error
	building_id, err = strconv.Atoi(args[0])
	if err != nil {
		log.Error("农田id[%v]转换失败[%v]", args[0], err.Error())
		return -1
	}

	is_speedup := 0
	if len(args) > 1 {
		is_speedup, err = strconv.Atoi(args[1])
		if err != nil {
			log.Error("是否加速[%v]转换失败[%v]", args[1])
			return -1
		}
	}

	b_speedup := false
	if is_speedup > 0 {
		b_speedup = true
	}
	return p.harvest_crop(int32(building_id), b_speedup)
}

func add_depot_building_cmd(p *Player, args []string) int32 {
	if len(args) < 1 {
		log.Error("参数[%v]不够", len(args))
		return -1
	}

	var depot_building_id int
	var err error
	depot_building_id, err = strconv.Atoi(args[0])
	if err != nil {
		log.Error("仓库建筑ID[%v]转换失败[%v]", depot_building_id, err.Error())
		return -1
	}

	var num int = 1
	p.AddDepotBuilding(int32(depot_building_id), int32(num), "test_add_depot_building", "test", true)

	return 1
}

func all_depot_building_cmd(p *Player, args []string) int32 {
	for k, _ := range building_table_mgr.Map {
		p.AddDepotBuilding(int32(k), int32(10), "test_all_depot_building", "test", true)
	}
	p.SendDepotBuildingUpdate()
	return 1
}

func list_depot_building_cmd(p *Player, args []string) int32 {
	all_index := p.db.BuildingDepots.GetAllIndex()
	if all_index == nil || len(all_index) == 0 {
		log.Info("玩家[%v]仓库建筑为空")
		return 0
	}

	log.Info("@@@@ 仓库建筑如下:")
	for i, id := range all_index {
		d := p.db.BuildingDepots.Get(id)
		if d == nil {
			continue
		}
		log.Info("@@@@@@@@ [%v] [Id:%v, Num:%v]", i, d.CfgId, d.Num)
	}

	return 0
}

func set_building_cmd(p *Player, args []string) int32 {
	if len(args) < 1 {
		log.Error("参数[%v]不够", len(args))
		return -1
	}

	var depot_building_id int
	var err error
	depot_building_id, err = strconv.Atoi(args[0])
	if err != nil {
		log.Error("仓库建筑id[%v]转换错误[%v]", depot_building_id, err.Error())
		return -1
	}

	building := building_table_mgr.Map[int32(depot_building_id)]
	if building == nil {
		log.Error("仓库建筑ID[%v]找不到配置", depot_building_id)
		return -1
	}

	x := int32(-500)
	y := int32(-500)
	b := false
	var iret int32
	for ; x < 2800; x++ {
		for ; y < 2800; y++ {
			iret = p.if_pos_can_set_building(x, y, building.MapSizes[0], building.MapSizes[1], 0, building.Geography)
			if iret > 0 {
				b = true
				break
			}
		}
		if b {
			break
		}
	}

	if !b {
		log.Error("设置仓库建筑[%v]失败", depot_building_id)
		return -1
	}

	/*
		new_building_db := &dbPlayerBuildingData{}
		new_building_db.Id = p.db.Info.IncbyNextBuildingId(1)
		new_building_db.CfgId = int32(depot_building_id)
		new_building_db.X = x
		new_building_db.Y = y
		new_building_db.Dir = table_config.BUILDING_DIR_BIG_X_DIR

		p.db.Buildings.Add(new_building_db)
	*/

	p.TrySetMapBuildingDefDir(int32(depot_building_id))

	p.RemoveDepotBuilding(int32(depot_building_id), 1, "test_set_building", "test")

	return 0
}

func list_building_cmd(p *Player, args []string) int32 {
	all_index := p.db.Buildings.GetAllIndex()
	if all_index == nil || len(all_index) == 0 {
		log.Info("玩家[%v]地图建筑为空", p.Id)
		return 0
	}

	log.Info("@@@@ 地图建筑如下: ")
	for i, id := range all_index {
		d := p.db.Buildings.Get(id)
		if d == nil {
			log.Warn("玩家[%v]地图建筑[%v]不存在", p.Id, id)
			continue
		}
		log.Info("@@@@@@@@ [%v] [building_id:%v, depot_build_ing:%v, x:%v, y:%v, dir:%v]", i, d.Id, d.CfgId, d.X, d.Y, d.Dir)
	}

	return 0
}

func add_cathouse_cat_cmd(p *Player, args []string) int32 {
	if len(args) < 2 {
		log.Error("参数[%v]不够", len(args))
		return -1
	}

	var cathouse_id, cat_id int
	var err error
	cat_id, err = strconv.Atoi(args[0])
	if err != nil {
		log.Error("猫ID[%v]转换失败[%v]", args[0], err.Error())
		return -1
	}
	cathouse_id, err = strconv.Atoi(args[1])
	if err != nil {
		log.Error("猫舍ID[%v]转换失败[%v]", args[1], err.Error())
		return -1
	}

	return p.cathouse_add_cat(int32(cat_id), int32(cathouse_id))
}

func remove_cathouse_cat_cmd(p *Player, args []string) int32 {
	if len(args) < 2 {
		log.Error("参数[%v]不够", len(args))
		return -1
	}

	var cathouse_id, cat_id int
	var err error
	cat_id, err = strconv.Atoi(args[0])
	if err != nil {
		log.Error("猫ID[%v]转换失败[%v]", args[0], err.Error())
		return -1
	}

	cathouse_id, err = strconv.Atoi(args[1])
	if err != nil {
		log.Error("猫舍ID[%v]转换失败[%v]", args[1], err.Error())
		return -1
	}

	return p.cathouse_remove_cat(int32(cat_id), int32(cathouse_id))
}

func list_cathouses_cmd(p *Player, args []string) int32 {
	/*all_index := p.db.CatHouses.GetAllIndex()
	if all_index == nil || len(all_index) == 0 {
		log.Info("玩家[%v]没有猫舍", p.Id)
		return -1
	}

	log.Info("@@@@ 玩家猫舍如下: ")
	for i, id := range all_index {
		cathouse_data := p.db.CatHouses.Get(id)
		if cathouse_data == nil {
			continue
		}
		log.Info("@@@@@@@@ [%v] cathouse_id[%v] cid[%v] cat_ids[%v], level[%v]", i, id, cathouse_data.CfgId, cathouse_data.CatIds, cathouse_data.Level)
	}*/

	return p.get_cathouses_info()
}

func cathouse_levelup_cmd(p *Player, args []string) int32 {
	if len(args) < 1 {
		log.Error("参数[%v]不够", len(args))
		return -1
	}

	var cathouse_id int
	var err error
	cathouse_id, err = strconv.Atoi(args[0])
	if err != nil {
		log.Error("猫舍ID[%v]转换失败[%v]", cathouse_id, err.Error())
		return -1
	}

	return p.cathouse_start_levelup(int32(cathouse_id), true)
}

func cathouse_speedup_cmd(p *Player, args []string) int32 {
	if len(args) < 1 {
		log.Error("参数[%v]不够", len(args))
		return -1
	}

	var cathouse_id int
	var err error
	cathouse_id, err = strconv.Atoi(args[0])
	if err != nil {
		log.Error("猫舍ID[%v]转换失败[%v]", cathouse_id, err.Error())
		return -1
	}

	return p.cathouse_speed_levelup(int32(cathouse_id))
}

func cathouse_sell_cmd(p *Player, args []string) int32 {
	if len(args) < 1 {
		log.Error("参数[%v]不够", len(args))
		return -1
	}

	var cathouse_id int
	var err error
	if err != nil {
		log.Error("猫舍ID[%v]转换失败[%v]", cathouse_id, err.Error())
		return -1
	}

	return p.cathouse_speed_levelup(int32(cathouse_id))
}

func cathouse_produce_gold_cmd(p *Player, args []string) int32 {
	if len(args) < 1 {
		log.Error("参数[%v]不够", len(args))
		return -1
	}

	var cathouse_id int
	var err error
	cathouse_id, err = strconv.Atoi(args[0])
	if err != nil {
		return -1
	}

	return p.cathouse_produce_gold(int32(cathouse_id))
}

func cathouse_collect_gold_cmd(p *Player, args []string) int32 {
	if len(args) < 1 {
		log.Error("参数[%v]不够", len(args))
		return -1
	}

	var cathouse_id int
	var err error
	cathouse_id, err = strconv.Atoi(args[0])
	if err != nil {
		log.Error("猫舍ID[%v]转换失败[%v]", args[0], err.Error())
		return -1
	}

	return p.cathouse_collect_gold(int32(cathouse_id))
}

func get_dailys_cmd(p *Player, args []string) int32 {
	return 1
}

func get_achieves_cmd(p *Player, args []string) int32 {
	return 1
}

func complete_task_cmd(p *Player, args []string) int32 {
	/*if len(args) < 1 {
		log.Error("参数[%v]不够", len(args))
		return -1
	}

	var task_id int
	var err error
	task_id, err = strconv.Atoi(args[0])
	if err != nil {
		log.Error("任务ID[%v]转换失败[%v]", args[0], err.Error())
		return -1
	}

	return p.complete_task(int32(task_id))*/
	return 1
}

func get_daily_reward_cmd(p *Player, args []string) int32 {
	if len(args) < 1 {
		log.Error("参数[%v]不够", len(args))
		return -1
	}

	var task_id int
	var err error
	task_id, err = strconv.Atoi(args[0])
	if err != nil {
		log.Error("日常任务ID[%v]转换失败[%v]", args[0], err.Error())
		return -1
	}

	return p.task_get_reward(int32(task_id))
}

func search_friend_id_cmd(p *Player, args []string) int32 {
	if len(args) < 1 {
		log.Error("参数[%v]不够", len(args))
		return -1
	}

	var friend_id int
	var err error
	friend_id, err = strconv.Atoi(args[0])
	if err != nil {
		log.Error("好友ID[%v]转换失败[%v]", args[0], err.Error())
		return -1
	}

	return p.search_friend_by_id(int32(friend_id))
}

func search_friend_name_cmd(p *Player, args []string) int32 {
	if len(args) < 1 {
		log.Error("参数[%v]不够", len(args))
		return -1
	}

	return p.search_friend(args[0])
}

func add_friend_cmd(p *Player, args []string) int32 {
	if len(args) < 1 {
		log.Error("参数[%v]不够", len(args))
		return -1
	}

	var friend_id int
	var err error
	friend_id, err = strconv.Atoi(args[0])
	if err != nil {
		log.Error("好友ID[%v]转换失败[%v]", args[0], err.Error())
		return -1
	}

	return p.add_friend_by_id(int32(friend_id))
}

func agree_friend_cmd(p *Player, args []string) int32 {
	if len(args) < 1 {
		log.Error("参数[%v]不够", len(args))
		return -1
	}

	var friend_id int
	var err error
	friend_id, err = strconv.Atoi(args[0])
	if err != nil {
		log.Error("好友ID[%v]转换失败[%v]", args[0], err.Error())
		return -1
	}

	return p.agree_add_friend(int32(friend_id))
}

func refuse_friend_cmd(p *Player, args []string) int32 {
	if len(args) < 1 {
		log.Error("参数[%v]不够", len(args))
		return -1
	}

	var friend_id int
	var err error
	friend_id, err = strconv.Atoi(args[0])
	if err != nil {
		log.Error("好友ID[%v]转换失败[%v]", args[0], err.Error())
		return -1
	}

	return p.refuse_add_friend(int32(friend_id))
}

func remove_friend_cmd(p *Player, args []string) int32 {
	if len(args) < 1 {
		log.Error("参数[%v]不够", len(args))
		return -1
	}

	var friend_id int
	var err error
	friend_id, err = strconv.Atoi(args[0])
	if err != nil {
		log.Error("好友[%v]转换失败[%v]", args[0], err.Error())
		return -1
	}

	return p.remove_friend(int32(friend_id))
}

func get_friends_cmd(p *Player, args []string) int32 {
	return p.get_friend_list()
}

func get_friend_info_cmd(p *Player, args []string) int32 {

	return 1
}

func give_friend_points_cmd(p *Player, args []string) int32 {
	if len(args) < 1 {
		log.Error("参数[%v]不够", len(args))
		return -1
	}

	friend_id := make([]int32, len(args))
	var err error
	var fid int
	for i, _ := range args {
		fid, err = strconv.Atoi(args[i])
		if err != nil {
			log.Error("好友ID[%v]转换失败[%v]", args[i], err.Error())
			return -1
		}
		friend_id[i] = int32(fid)
	}
	return p.give_friend_points(friend_id)
}

func get_friend_points_cmd(p *Player, args []string) int32 {
	if len(args) < 1 {
		log.Error("参数[%v]不够", len(args))
		return -1
	}

	friend_id := make([]int32, len(args))
	var err error
	var fid int
	for i, _ := range args {
		fid, err = strconv.Atoi(args[i])
		if err != nil {
			log.Error("好友ID[%v]转换失败[%v]", args[i], err.Error())
			return -1
		}
		friend_id[i] = int32(fid)
	}
	return p.get_friend_points(friend_id)
}

func friend_chat_cmd(p *Player, args []string) int32 {
	if len(args) < 2 {
		log.Error("参数[%v]不够", len(args))
		return -1
	}

	var err error
	var fid int
	fid, err = strconv.Atoi(args[0])
	if err != nil {
		log.Error("好友ID[%v]转换失败[%v]", args[0], err.Error())
		return -1
	}

	return p.friend_chat(int32(fid), []byte(args[1]))
}

func friend_get_unread_message_num_cmd(p *Player, args []string) int32 {
	if len(args) < 1 {
		log.Error("参数[%v]不够", len(args))
		return -1
	}

	var err error
	var fid int
	fid, err = strconv.Atoi(args[0])
	if err != nil {
		log.Error("好友ID[%v]转换失败[%v]", args[0], err.Error())
		return -1
	}

	return p.friend_get_unread_message_num([]int32{int32(fid)})
}

func friend_pull_unread_cmd(p *Player, args []string) int32 {
	if len(args) < 1 {
		log.Error("参数[%v]不够", len(args))
		return -1
	}

	var err error
	var fid int
	fid, err = strconv.Atoi(args[0])
	if err != nil {
		log.Error("好友ID[%v]转换失败[%v]", args[0], err.Error())
		return -1
	}

	return p.friend_pull_unread_message(int32(fid))
}

func friend_confirm_unread_cmd(p *Player, args []string) int32 {
	if len(args) < 1 {
		log.Error("参数[%v]不够", len(args))
		return -1
	}

	var err error
	var fid int
	fid, err = strconv.Atoi(args[0])
	if err != nil {
		log.Error("好友Id[%v]转换失败[%v]", args[0], err.Error())
		return -1
	}

	num := 0
	if len(args) > 1 {
		num, err = strconv.Atoi(args[1])
		if err != nil {
			log.Error("确认未读消息数目[%v]转换失败[%v]", args[1], err.Error())
			return -1
		}
	}

	return p.friend_confirm_unread_message(int32(fid), int32(num))
}

func finish_stage_cmd(p *Player, args []string) int32 {
	if len(args) < 2 {
		log.Error("参数[%v]不够", len(args))
		return -1
	}

	var result, stage_id int
	var err error
	result, err = strconv.Atoi(args[0])
	if err != nil {
		log.Error("通关结果[%v]转换失败[%v]", args[0], err.Error())
		return -1
	}

	stage_id, err = strconv.Atoi(args[1])
	if err != nil {
		log.Error("关卡ID[%v]转换失败[%v]", args[1], err.Error())
		return -1
	}

	var stars = 3
	if len(args) >= 3 {
		stars, err = strconv.Atoi(args[2])
		if nil != err {
			log.Info("填写的星星数有问题[%s]")
			stars = 3
		}
	}

	var d StageBeginData
	d.stage_id = int32(stage_id)
	p.CheckBeginStage(&d)

	return p.stage_pass(int32(result), int32(stage_id), int32(99999), int32(stars), make([]*msg_client_message.ItemInfo, 0), true)

	return 1
}

func compose_foster_cmd(p *Player, args []string) int32 {
	if len(args) < COMPOSE_FOSTER_CARD_NUM {
		log.Error("参数[%v]不够", len(args))
		return -1
	}

	var err error
	var id int
	card_ids := make([]int32, COMPOSE_FOSTER_CARD_NUM)
	for i := 0; i < COMPOSE_FOSTER_CARD_NUM; i++ {
		id, err = strconv.Atoi(args[i])
		if err != nil {
			log.Error("转换寄养卡ID[%v]失败[%v]", args[i], err.Error())
			return -1
		}
		card_ids[i] = int32(id)
	}

	return p.compose_foster_card(card_ids)
}

func set_foster_id_cmd(p *Player, args []string) int32 {
	if len(args) < 1 {
		log.Error("参数[%v]不够", len(args))
		return -1
	}

	var err error
	var id int
	id, err = strconv.Atoi(args[0])
	if err != nil {
		log.Error("转换寄养所建筑ID[%v]错误[%v]", args[0], err.Error())
		return -1
	}

	const max_id int = 10000
	if id < max_id {
		log.Error("该ID不能小于%v", max_id)
		return -1
	}

	p.db.Foster.SetBuildingId(int32(id))
	d := &dbPlayerBuildingData{
		Id:    int32(id),
		CfgId: 1005,
	}
	p.db.Buildings.Add(d)
	return 1
}

func pull_foster_cmd(p *Player, args []string) int32 {
	if len(args) < 1 {
		log.Error("参数[%v]不够", len(args))
		return -1
	}

	is_settle, err := strconv.Atoi(args[0])
	if err != nil {
		log.Error("是否结算[%v]转换错误[%v]", args[0], err.Error())
		return -1
	}

	if is_settle == 0 {
		return p.foster_data_pull(false)
	} else {
		return p.foster_data_pull(true)
	}
}

func pull_foster_with_friends_cmd(p *Player, args []string) int32 {
	return p.foster_data_pull_with_friend()
}

func get_player_foster_cmd(p *Player, args []string) int32 {
	if len(args) < 1 {
		log.Error("参数[%v]不够", len(args))
		return -1
	}

	var err error
	var player_id int
	player_id, err = strconv.Atoi(args[0])
	if err != nil {
		log.Error("玩家ID转换错误[%v]", args[0], err.Error())
		return -1
	}

	return p.get_player_foster_cats(int32(player_id))
}

func foster_equip_card_cmd(p *Player, args []string) int32 {
	if len(args) < 2 {
		log.Error("参数[%v]不够", len(args))
		return -1
	}

	var err error
	var building_id, card_id int
	building_id, err = strconv.Atoi(args[0])
	if err != nil {
		log.Error("转换寄养所建筑ID[%v]错误[%v]", args[0], err.Error())
		return -1
	}

	card_id, err = strconv.Atoi(args[1])
	if err != nil {
		log.Error("寄样卡ID[%v]转换错误[%v]", args[1], err.Error())
		return -1
	}

	return p.foster_equip_card(int32(building_id), int32(card_id))
}

func foster_unequip_card_cmd(p *Player, args []string) int32 {
	if len(args) < 1 {
		log.Error("参数[%v]不够", len(args))
		return -1
	}

	var err error
	var building_id int
	building_id, err = strconv.Atoi(args[0])
	if err != nil {
		log.Error("转换寄养所建筑ID[%v]错误[%v]", args[0], err.Error())
		return -1
	}

	return p.foster_unequip_card(int32(building_id))
}

func foster_set_cat_cmd(p *Player, args []string) int32 {
	if len(args) < 2 {
		log.Error("参数[%v]不够", len(args))
		return -1
	}

	var err error
	var building_id, cat_id int
	building_id, err = strconv.Atoi(args[0])
	if err != nil {
		log.Error("转换寄养所建筑ID[%v]错误[%v]", args[0], err.Error())
		return -1
	}
	cat_id, err = strconv.Atoi(args[1])
	if err != nil {
		log.Error("转换猫ID[%v]错误[%v]", args[1], err.Error())
		return -1
	}
	return p.foster_set_cat(int32(building_id), int32(cat_id))
}

func foster_out_cat_cmd(p *Player, args []string) int32 {
	if len(args) < 2 {
		log.Error("参数[%v]不够", len(args))
		return -1
	}

	var err error
	var building_id, cat_id int
	building_id, err = strconv.Atoi(args[0])
	if err != nil {
		log.Error("转换寄养所建筑ID[%v]错误[%v]", args[0], err.Error())
		return -1
	}
	cat_id, err = strconv.Atoi(args[1])
	if err != nil {
		log.Error("转换猫ID[%v]错误[%v]", args[1], err.Error())
		return -1
	}

	return p.foster_out_cat(int32(building_id), int32(cat_id))
}

func foster_set_cat_friend_cmd(p *Player, args []string) int32 {
	if len(args) < 2 {
		log.Error("参数[%v]不够", len(args))
		return -1
	}

	var err error
	var friend_id, cat_id int
	friend_id, err = strconv.Atoi(args[0])
	if err != nil {
		log.Error("转换好友ID[%v]错误[%v]", args[0], err.Error())
		return -1
	}
	cat_id, err = strconv.Atoi(args[1])
	if err != nil {
		log.Error("转换猫ID[%v]错误[%v]", args[1], err.Error())
		return -1
	}

	return p.foster_set_cat_friend(int32(friend_id), int32(cat_id))
}

func foster_get_player_foster_cmd(p *Player, args []string) int32 {
	if len(args) < 1 {
		log.Error("参数[%v]不够", len(args))
		return -1
	}

	var err error
	var player_id int
	player_id, err = strconv.Atoi(args[0])
	if err != nil {
		log.Error("转换玩家ID[%v]错误[%v]", args[0], err.Error())
		return -1
	}

	return p.get_player_foster_cats(int32(player_id))
}

func foster_empty_slot_friends_cmd(p *Player, args []string) int32 {
	return p.foster_get_empty_slot_friends()
}

func rank_test_cmd(p *Player, args []string) int32 {
	if len(args) < 1 {
		log.Error("参数[%v]不够", len(args))
		return -1
	}

	var err error
	var count int
	count, err = strconv.Atoi(args[0])
	if err != nil {
		log.Error("转换节点数量[%v]错误[%v]", args[0], err.Error())
		return -1
	}
	utils.SkiplistTest(int32(count))
	return 1
}

func rank_test2_cmd(p *Player, args []string) int32 {
	if len(args) < 1 {
		log.Error("参数[%v]不够", len(args))
		return -1
	}

	var err error
	var count int
	count, err = strconv.Atoi(args[0])
	if err != nil {
		log.Error("转换节点数量[%v]错误[%v]", args[0], err.Error())
		return -1
	}
	utils.SkiplistTest2(int32(count))
	return 1
}

func ranklist_cmd(p *Player, args []string) int32 {
	if len(args) < 1 {
		log.Error("参数[%v]不够", len(args))
		return -1
	}

	var err error
	var rank_type int
	rank_type, err = strconv.Atoi(args[0])
	if err != nil {
		log.Error("转换排行榜类型[%v]错误[%v]", args[0], err.Error())
		return -1
	}

	var param, rank_start, rank_num int
	if len(args) > 1 {
		rank_start, err = strconv.Atoi(args[1])
		if err != nil {
			log.Error("转换开始排名[%v]错误[%v]", args[1], err.Error())
			return -1
		}
	}

	if len(args) > 2 {
		rank_num, err = strconv.Atoi(args[2])
		if err != nil {
			log.Error("转换排名数[%v]错误[%v]", args[2], err.Error())
			return -1
		}
	}

	if len(args) > 3 {
		param, err = strconv.Atoi(args[3])
		if err != nil {
			log.Error("转换排名参数[%v]错误[%v]", args[3], err.Error())
			return -1
		}
	}

	if rank_type != common.RANK_LIST_TYPE_CAT_OUQI {
		if rank_type == common.RANK_LIST_TYPE_STAGE_TOTAL_SCORE || rank_type == common.RANK_LIST_TYPE_CHARM || rank_type == common.RANK_LIST_TYPE_BE_ZANED {
			return p.rank_list_get_data(int32(rank_type), int32(rank_start), int32(rank_num), nil)
		} else {
			log.Error("rank_type[%v] is invalid")
			return -1
		}
	} else {
		return p.rank_list_get_data(int32(rank_type), int32(rank_start), int32(rank_num), []int32{int32(param)})
	}
}

func visit_player_cmd(p *Player, args []string) int32 {
	if len(args) < 1 {
		log.Error("参数[%v]不够", len(args))
		return -1
	}

	var err error
	var player_id int
	player_id, err = strconv.Atoi(args[0])
	if err != nil {
		log.Error("转换玩家ID[%v]错误[%v]", args[0], err.Error())
		return -1
	}

	return p.VisitPlayerBuildings(int32(player_id))
}

func chat_cmd(p *Player, args []string) int32 {
	if len(args) < 1 {
		log.Error("参数[%v]不够", len(args))
		return -1
	}

	var chat_type int
	var err error
	chat_type, err = strconv.Atoi(args[0])
	if err != nil {
		return -1
	}

	return p.chat(int32(chat_type), []byte(args[1]), 0)
}

func chat_pull_cmd(p *Player, args []string) int32 {
	return 1
}

func push_sysmsg_cmd(p *Player, args []string) int32 {
	if len(args) < 2 {
		log.Error("参数[%v]不够", len(args))
		return -1
	}

	var err error
	var msg_type, param int
	msg_type, err = strconv.Atoi(args[0])
	if err != nil {
		log.Error("转换公告类型[%v]错误[%v]", args[0], err.Error())
		return -1
	}

	param, err = strconv.Atoi(args[1])
	if err != nil {
		log.Error("转换参数[%v]错误[%v]", args[1], err.Error())
		return -1
	}

	if !anouncement_mgr.PushNew(int32(msg_type), true, p.Id, p.db.GetName(), int32(p.db.GetLevel()), int32(param), 0, 0, "") {
		return -1
	}

	return 1
}

func push_sysmsg_text_cmd(p *Player, args []string) int32 {
	if len(args) < 1 {
		log.Error("参数[%v]不够", len(args))
		return -1
	}

	if !anouncement_mgr.PushNew(ANOUNCEMENT_TYPE_TEXT, true, p.Id, p.db.GetName(), int32(p.db.GetLevel()), 0, 0, 0, args[0]) {
		return -1
	}

	return 1
}

/*func get_personal_space_cmd(p *Player, args []string) int32 {
	var err error
	var player_id int
	if len(args) > 0 {
		player_id, err = strconv.Atoi(args[0])
		if err != nil {
			log.Error("转换参数[%v]错误[%v]", args[0], err.Error())
			return -1
		}
	}

	return p.get_personal_space(int32(player_id))
}

func change_signature_cmd(p *Player, args []string) int32 {
	if len(args) < 1 {
		log.Error("参数[%v]不够", len(args))
		return -1
	}

	return p.personal_space_modify_signature(args[0])
}

func send_personal_space_leave_msg_cmd(p *Player, args []string) int32 {
	if len(args) < 2 {
		log.Error("参数[%v]不够", len(args))
		return -1
	}

	var err error
	var player_id, pic_id int
	var msg string
	player_id, err = strconv.Atoi(args[0])
	if err != nil {
		log.Error("转换参数[%v]错误[%v]", args[0], err.Error())
		return -1
	}

	if len(args) > 2 {
		pic_id, err = strconv.Atoi(args[1])
		if err != nil {
			log.Error("转换参数[%v]错误[%v]", args[1], err.Error())
			return -1
		}
		msg = args[2]
	} else {
		msg = args[1]
	}
	return p.personal_space_send_leave_msg(int32(player_id), int32(pic_id), []byte(msg))
}

func delete_personal_space_leave_msg_cmd(p *Player, args []string) int32 {
	if len(args) < 2 {
		log.Error("参数[%v]不够", len(args))
		return -1
	}

	var err error
	var player_id, pic_id, msg_id int
	player_id, err = strconv.Atoi(args[0])
	if err != nil {
		log.Error("转换参数[%v]错误[%v]", args[0], err.Error())
		return -1
	}

	if len(args) > 2 {
		pic_id, err = strconv.Atoi(args[1])
		if err != nil {
			log.Error("转换参数[%v]错误[%v]", args[1], err.Error())
			return -1
		}
		msg_id, err = strconv.Atoi(args[2])
		if err != nil {
			log.Error("转换参数[%v]错误[%v]", args[2], err.Error())
			return -1
		}
	} else {
		msg_id, err = strconv.Atoi(args[1])
		if err != nil {
			log.Error("转换参数[%v]错误[%v]", args[1], err.Error())
			return -1
		}
	}

	return p.personal_space_delete_leave_msg(int32(player_id), int32(pic_id), int32(msg_id))
}

func pull_personal_space_leave_msg_cmd(p *Player, args []string) int32 {
	if len(args) < 3 {
		log.Error("参数[%v]不够", len(args))
		return -1
	}

	var err error
	var player_id, pic_id, start_index, msg_num int
	player_id, err = strconv.Atoi(args[0])
	if err != nil {
		log.Error("转换参数[%v]错误[%v]", args[0], err.Error())
		return -1
	}

	if len(args) > 3 {
		pic_id, err = strconv.Atoi(args[1])
		if err != nil {
			log.Error("转换参数[%v]错误[%v]", args[1], err.Error())
			return -1
		}
		start_index, err = strconv.Atoi(args[2])
		if err != nil {
			log.Error("转换参数[%v]错误[%v]", args[2], err.Error())
			return -1
		}
		msg_num, err = strconv.Atoi(args[3])
		if err != nil {
			log.Error("转换参数[%v]错误[%v]", args[3], err.Error())
			return -1
		}
	} else {
		start_index, err = strconv.Atoi(args[1])
		if err != nil {
			log.Error("转换参数[%v]错误[%v]", args[1], err.Error())
			return -1
		}
		msg_num, err = strconv.Atoi(args[2])
		if err != nil {
			log.Error("转换参数[%v]错误[%v]", args[2], err.Error())
			return -1
		}
	}

	return p.personal_space_pull_leave_msg(int32(player_id), int32(pic_id), int32(start_index), int32(msg_num))
}

func send_personal_space_leave_msg_comment_cmd(p *Player, args []string) int32 {
	if len(args) < 3 {
		log.Error("参数[%v]不够", len(args))
		return -1
	}

	var err error
	var player_id, pic_id, msg_id int
	player_id, err = strconv.Atoi(args[0])
	if err != nil {
		log.Error("转换参数[%v]错误[%v]", args[0], err.Error())
		return -1
	}

	var comment string
	if len(args) == 3 {
		msg_id, err = strconv.Atoi(args[1])
		if err != nil {
			log.Error("转换参数[%v]错误[%v]", args[1], err.Error())
			return -1
		}
		comment = args[2]
	} else {
		pic_id, err = strconv.Atoi(args[1])
		if err != nil {
			log.Error("转换参数[%v]错误[%v]", args[1], err.Error())
			return -1
		}
		msg_id, err = strconv.Atoi(args[2])
		if err != nil {
			log.Error("转换参数[%v]错误[%v]", args[2], err.Error())
			return -1
		}
		comment = args[3]
	}

	return p.personal_space_send_leave_msg_comment(int32(player_id), int32(pic_id), int32(msg_id), []byte(comment))
}

func delete_personal_space_leave_msg_comment_cmd(p *Player, args []string) int32 {
	if len(args) < 3 {
		log.Error("参数[%v]不够", len(args))
		return -1
	}

	var err error
	var player_id, pic_id, msg_id, comment_id int
	player_id, err = strconv.Atoi(args[0])
	if err != nil {
		log.Error("转换参数[%v]错误[%v]", args[0], err.Error())
		return -1
	}

	if len(args) < 4 {
		msg_id, err = strconv.Atoi(args[1])
		if err != nil {
			log.Error("转换参数[%v]错误[%v]", args[1], err.Error())
			return -1
		}
		comment_id, err = strconv.Atoi(args[2])
		if err != nil {
			log.Error("转换参数[%v]错误[%v]", args[2], err.Error())
			return -1
		}
	} else {
		pic_id, err = strconv.Atoi(args[1])
		if err != nil {
			log.Error("转换参数[%v]错误[%v]", args[1], err.Error())
			return -1
		}
		msg_id, err = strconv.Atoi(args[2])
		if err != nil {
			log.Error("转换参数[%v]错误[%v]", args[2], err.Error())
			return -1
		}
		comment_id, err = strconv.Atoi(args[3])
		if err != nil {
			log.Error("转换参数[%v]错误[%v]", args[3], err.Error())
			return -1
		}
	}

	return p.personal_space_delete_leave_msg_comment(int32(player_id), int32(pic_id), int32(msg_id), int32(comment_id))
}

func pull_personal_space_leave_msg_comment_cmd(p *Player, args []string) int32 {
	if len(args) < 4 {
		log.Error("参数[%v]不够", len(args))
		return -1
	}

	var err error
	var player_id, pic_id, msg_id, start_index, comment_num int
	player_id, err = strconv.Atoi(args[0])
	if err != nil {
		log.Error("转换参数[%v]错误[%v]", args[0], err.Error())
		return -1
	}

	if len(args) < 5 {
		msg_id, err = strconv.Atoi(args[1])
		if err != nil {
			log.Error("转换参数[%v]错误[%v]", args[1], err.Error())
			return -1
		}
		start_index, err = strconv.Atoi(args[2])
		if err != nil {
			log.Error("转换参数[%v]错误[%v]", args[2], err.Error())
			return -1
		}
		comment_num, err = strconv.Atoi(args[3])
		if err != nil {
			log.Error("转换参数[%v]错误[%v]", args[3], err.Error())
			return -1
		}
	} else {
		pic_id, err = strconv.Atoi(args[1])
		if err != nil {
			log.Error("转换参数[%v]错误[%v]", args[1], err.Error())
			return -1
		}
		msg_id, err = strconv.Atoi(args[2])
		if err != nil {
			log.Error("转换参数[%v]错误[%v]", args[2], err.Error())
			return -1
		}
		start_index, err = strconv.Atoi(args[3])
		if err != nil {
			log.Error("转换参数[%v]错误[%v]", args[3], err.Error())
			return -1
		}
		comment_num, err = strconv.Atoi(args[4])
		if err != nil {
			log.Error("转换参数[%v]错误[%v]", args[4], err.Error())
			return -1
		}
	}

	return p.personal_space_pull_leave_msg_comment(int32(player_id), int32(pic_id), int32(msg_id), int32(start_index), int32(comment_num))
}*/

type test_cmd_func func(*Player, []string) int32

var test_cmd2funcs = map[string]test_cmd_func{
	"player_info":         player_info_cmd,
	"add_exp":             add_exp_cmd,
	"set_level":           set_level_cmd,
	"add_item":            add_item_cmd,
	"add_all_item":        add_all_item_cmd,
	"use_item":            use_item_cmd,
	"list_item":           list_item_cmd,
	"add_coin":            add_coin_cmd,
	"set_coin":            set_coin_cmd,
	"add_diamond":         add_diamond_cmd,
	"set_diamond":         set_diamond_cmd,
	"add_cat":             add_cat_cmd,
	"add_cat2":            add_cat_with_level_star_skill_cmd,
	"add_catfood":         add_cat_food_cmd,
	"add_friendpoints":    add_friend_points_cmd,
	"add_soulstone":       add_soul_stone_cmd,
	"add_charm":           add_charm_cmd,
	"add_charmmedal":      add_charm_medal_cmd,
	"add_zan":             add_zan_cmd,
	"add_star":            add_star_cmd,
	"draw_card":           draw_card_cmd,
	"drop_items":          drop_items_cmd,
	"compose_cat":         compose_cat_cmd,
	"shop_items":          get_shop_items_cmd,
	"refresh_shop":        refresh_shop_cmd,
	"buy_item":            buy_shop_item_cmd,
	"sell_item":           sell_item_cmd,
	"feed_cat":            cat_feed_cmd,
	"upstar":              cat_upstar_cmd,
	"upskill":             cat_upskill_cmd,
	"list_cat":            list_cat_cmd,
	"see_cat":             see_cat_cmd,
	"making_buildings":    get_making_buildings_cmd,
	"get_formulas":        get_formulas_cmd,
	"exchange_formula":    exchange_formulas_cmd,
	"make_formula":        making_formula_building_cmd,
	"buy_slot":            buy_formula_slot_cmd,
	"speedup_slot":        speedup_making_formula_building_cmd,
	"get_completed":       get_completed_formula_building_cmd,
	"cancel_making":       cancel_making_formula_building_cmd,
	"get_crops":           get_crops_cmd,
	"plant_crop":          plant_crop_cmd,
	"speedup_crop":        speedup_crop_cmd,
	"harvest_crop":        harvest_crop_cmd,
	"add_depot_building":  add_depot_building_cmd, // 添加仓库建筑
	"all_depot_building":  all_depot_building_cmd, // 增加所有仓库建筑
	"list_depot_building": list_depot_building_cmd,
	"set_building":        set_building_cmd, // 放置建筑，仅用作测试
	"list_building":       list_building_cmd,
	"set_cat":             add_cathouse_cat_cmd,
	"out_cat":             remove_cathouse_cat_cmd,
	"list_cathouse":       list_cathouses_cmd,
	"cathouse_levelup":    cathouse_levelup_cmd,
	"cathouse_speed":      cathouse_speedup_cmd,
	"cathouse_sell":       cathouse_sell_cmd,
	"produce_gold":        cathouse_produce_gold_cmd,
	"cllect_gold":         cathouse_collect_gold_cmd,
	"get_dailys":          get_dailys_cmd,
	"get_achieves":        get_achieves_cmd,
	//"complete_task":             complete_task_cmd,
	//"daily_reward":              get_daily_reward_cmd,
	//"achieve_reward":            get_achieve_reward_cmd,
	"search_friend":             search_friend_id_cmd,
	"search_friend_name":        search_friend_name_cmd,
	"add_friend":                add_friend_cmd,
	"agree_friend":              agree_friend_cmd,
	"refuse_friend":             refuse_friend_cmd,
	"remove_friend":             remove_friend_cmd,
	"get_friends":               get_friends_cmd,
	"get_friend_info":           get_friend_info_cmd,
	"give_friend_points":        give_friend_points_cmd,
	"get_friend_points":         get_friend_points_cmd,
	"friend_chat":               friend_chat_cmd,
	"friend_unread_num":         friend_get_unread_message_num_cmd,
	"friend_unread":             friend_pull_unread_cmd,
	"friend_confirm_unread":     friend_confirm_unread_cmd,
	"finish_stage":              finish_stage_cmd,
	"compose_foster":            compose_foster_cmd,
	"set_foster_id":             set_foster_id_cmd,
	"pull_foster":               pull_foster_cmd,
	"pull_foster2":              pull_foster_with_friends_cmd,
	"get_player_foster":         get_player_foster_cmd,
	"foster_equip_card":         foster_equip_card_cmd,
	"foster_unequip_card":       foster_unequip_card_cmd,
	"foster_set_cat":            foster_set_cat_cmd,
	"foster_out_cat":            foster_out_cat_cmd,
	"foster_set_cat_friend":     foster_set_cat_friend_cmd,
	"foster_empty_slot_friends": foster_empty_slot_friends_cmd,
	"rank_test":                 rank_test_cmd,
	"rank_test2":                rank_test2_cmd,
	"ranklist":                  ranklist_cmd,
	"visit_player":              visit_player_cmd,
	//"world_chat":                world_chat_cmd,
	//"pull_world_chat":           pull_world_chat_cmd,
	"push_sysmsg":      push_sysmsg_cmd,
	"push_sysmsg_text": push_sysmsg_text_cmd,
	/*"get_ps":                get_personal_space_cmd,
	"change_signature":      change_signature_cmd,
	"send_ps_msg":           send_personal_space_leave_msg_cmd,
	"del_ps_msg":            delete_personal_space_leave_msg_cmd,
	"pull_ps_msg":           pull_personal_space_leave_msg_cmd,
	"send_ps_msg_comment":   send_personal_space_leave_msg_comment_cmd,
	"del_ps_msg_comment":    delete_personal_space_leave_msg_comment_cmd,
	"pull_ps_msg_comment":   pull_personal_space_leave_msg_comment_cmd,*/
}

func C2STestCommandHandler(p *Player, msg_data []byte) int32 {
	var req msg_client_message.C2S_TEST_COMMAND
	err := proto.Unmarshal(msg_data, &req)
	if err != nil {
		log.Error("unmarshal msg failed %v", err.Error())
		return -1
	}

	cmd := req.GetCmd()
	args := req.GetArgs()
	res := int32(0)

	fun := test_cmd2funcs[cmd]
	if fun != nil {
		res = fun(p, args)
	} else {
		log.Warn("不支持的测试命令[%v]", cmd)
	}

	return res
}
