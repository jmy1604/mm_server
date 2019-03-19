package main

//"math/rand"
//"mm_server/libs/log"
//"mm_server/proto/gen_go/client_message"
//"time"

//"github.com/golang/protobuf/proto"

/*func (this *Player) OpenChest(chest_id int32, reason, mod string, cost_dimond, pos int32, bslience bool) *msg_client_message.S2COpenChest {
	chest_cfg := map_chest_mgr.Map[chest_id]
	if nil == chest_cfg {
		log.Error("Player OpenChest failed to find[%d]", chest_id)
		return nil
	}

	rand.Seed(time.Now().Unix())

	res_2cli := &msg_client_message.S2COpenChest{}
	res_2cli.ChestId = chest_id
	coin_rand := chest_cfg.GoldMax - chest_cfg.GoldMin
	cur_coin := this.GetCoin()
	tmp_val := int32(0)
	for i := int32(0); i < chest_cfg.GoldExtractTimes; i++ {
		if coin_rand <= 0 {
			cur_coin = this.AddCoin(chest_cfg.GoldMin, reason, mod)
		} else {
			tmp_val = chest_cfg.GoldMin + rand.Int31n(coin_rand)
			cur_coin = this.AddCoin(tmp_val, reason, mod)
		}
	}

	diamond_rand := chest_cfg.GemMax - chest_cfg.GemMin + 1
	cur_diamond := this.GetDiamond()
	for i := int32(0); i < chest_cfg.GemExtractTimes; i++ {
		if diamond_rand <= 0 {
			cur_diamond = this.AddDiamond(chest_cfg.GemMin, reason, mod)
		} else {
			tmp_val = chest_cfg.GemMin + rand.Int31n(diamond_rand)
			cur_diamond = this.AddDiamond(chest_cfg.GemMin+rand.Int31n(diamond_rand), reason, mod)
		}
	}

	res_2cli.CurCoins = proto.Int32(cur_coin)
	res_2cli.CurDiamond = proto.Int32(cur_diamond)

	drop_items := make(map[int32]int32)
	if len(drop_items) > 0 {
		res_2cli.NewItems = make([]*msg_client_message.ItemInfo, 0, len(drop_items))
		var tmp_item_add *msg_client_message.ItemInfo
		for itemcfgid, count := range drop_items {
			tmp_item_add = &msg_client_message.ItemInfo{}
			tmp_item_add.ItemCfgId = proto.Int32(itemcfgid)
			tmp_item_add.ItemNum = proto.Int32(count)
			res_2cli.NewItems = append(res_2cli.NewItems, tmp_item_add)
		}
		if cost_dimond > 0 {
			res_2cli.CostDiamond = proto.Int32(cost_dimond)
		}
	}

	if !bslience {
		this.Send(res_2cli)
	}

	return res_2cli
}*/

// =====================================================================
