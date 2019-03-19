package main

import (
	"math/rand"
	"mm_server/libs/log"
	"mm_server/proto/gen_go/client_message"
	"mm_server/src/tables"
	"time"

	"github.com/golang/protobuf/proto"
)

func reg_player_draw_msg() {
	msg_handler_mgr.SetPlayerMsgHandler(uint16(msg_client_message.C2SDraw_ProtoID), C2SDrawHandler)
}

func (this *Player) drop_item_by_id(id int32, check_same bool) (bool, *msg_client_message.ItemInfo, *msg_client_message.CatInfo, *msg_client_message.DepotBuildingInfo) {
	drop_lib := drop_card_table_mgr.Map[id]
	if nil == drop_lib {
		return false, nil, nil, nil
	}
	item, cat, building := this.drop_item(drop_lib, check_same, true)
	return true, item, cat, building
}

func (this *Player) drop_item(drop_lib *tables.DropCardTypeLib, check_same, badd bool) (item *msg_client_message.ItemInfo, cat *msg_client_message.CatInfo, building *msg_client_message.DepotBuildingInfo) {
	log.Info("当前抽取库 总数目%d 总权重%d 详细%v", drop_lib.TotalCount, drop_lib.TotalWeight, drop_lib)

	if check_same {
		if this.used_drop_ids == nil || len(this.used_drop_ids) == int(drop_lib.TotalCount) {
			this.used_drop_ids = make(map[int32]int32)
		}
	}

	get_same := false
	rand_val := rand.Int31n(drop_lib.TotalWeight)
	var tmp_item *tables.XmlDropCardItem
	for i := int32(0); i < drop_lib.TotalCount; i++ {
		tmp_item = drop_lib.DropItems[i]
		if nil == tmp_item {
			continue
		}

		if tmp_item.Weight > rand_val || get_same {
			if check_same {
				if _, o := this.used_drop_ids[tmp_item.DropType]; o {
					get_same = true
					if i == drop_lib.TotalCount-1 {
						i = 0
					}
					if len(this.used_drop_ids) == int(drop_lib.TotalCount) {
						this.used_drop_ids = make(map[int32]int32)
					}
					log.Debug("!!!!!!!!!!! !!!!!!!! total_count[%v]  used_drop_ids len[%v]  i[%v]", drop_lib.TotalCount, len(this.used_drop_ids), i)
					continue
				}
			}
			_, num := rand31n_from_range(tmp_item.Min, tmp_item.Max)
			if nil != item_table_mgr.Map[tmp_item.DropType] {
				if badd {
					this.AddItem(tmp_item.DropType, num, "draw", "player", true)
				}
				item = &msg_client_message.ItemInfo{}
				item.ItemCfgId = tmp_item.DropType
				item.ItemNum = num

			} else if nil != cat_table_mgr.Map[tmp_item.DropType] {
				if badd {
					cat = this.AddCat(tmp_item.DropType, "draw", "player", true)
				} else {
					cat = &msg_client_message.CatInfo{}
					cat.CatCfgId = tmp_item.DropType
				}
			} else if nil != building_table_mgr.Map[tmp_item.DropType] {
				if badd {
					this.AddDepotBuilding(tmp_item.DropType, num, "draw", "draw_building", true)
				}
				building = &msg_client_message.DepotBuildingInfo{}
				building.CfgId = tmp_item.DropType
				building.Num = num

			} else {
				if badd {
					if this.AddItemResource(tmp_item.DropType, num, "draw", "draw_item_resource") < 0 {
						log.Error("C2SDrawHandler rand dropid[%d] not item or cat or building or item resource", tmp_item.DropType)
					} else {
						item = &msg_client_message.ItemInfo{ItemCfgId: tmp_item.DropType, ItemNum: num, RemainSeconds: 0}
					}
				} else {
					item = &msg_client_message.ItemInfo{ItemCfgId: tmp_item.DropType, ItemNum: num, RemainSeconds: 0}
				}
			}
			if check_same {
				this.used_drop_ids[tmp_item.DropType] = tmp_item.Weight
			}
			break
		} else {
			rand_val -= tmp_item.Weight
		}
	}

	return
}

func (this *Player) DropItems(items_info []*tables.ItemInfo, draw_count int32, badd bool) (bool, []*msg_client_message.ItemInfo, []*msg_client_message.CatInfo, []*msg_client_message.DepotBuildingInfo) {
	total_drop_count := int32(0)
	for n := 0; n < len(items_info); n++ {
		total_drop_count += items_info[n].Num
	}

	cats := make([]*msg_client_message.CatInfo, 0, draw_count*total_drop_count)
	items := make([]*msg_client_message.ItemInfo, 0, draw_count*total_drop_count)
	buildings := make([]*msg_client_message.DepotBuildingInfo, 0, draw_count*total_drop_count)

	this.used_drop_ids = make(map[int32]int32)

	seed := time.Now().Unix() + time.Now().UnixNano()
	rand.Seed(seed + int64(rand.Int31n(100)))
	for count := int32(0); count < draw_count; count++ {
		for i := 0; i < len(items_info); i++ {
			for j := 0; j < int(items_info[i].Num); j++ {
				draw_lib := drop_card_table_mgr.Map[items_info[i].Id]
				if nil == draw_lib {
					log.Error("Player[%v] draw card not found draw lib[%v]", this.Id, items_info[i].Id)
					return false, nil, nil, nil
				}

				tmp_item, tmp_cat, tmp_building := this.drop_item(draw_lib, true, badd)
				if tmp_item != nil {
					items = append(items, tmp_item)
				}
				if tmp_cat != nil {
					cats = append(cats, tmp_cat)
				}
				if tmp_building != nil {
					buildings = append(buildings, tmp_building)
				}
			}
		}
	}

	return true, items, cats, buildings
}

func (this *Player) DropItems2(items_info []int32, badd bool) (bool, []*msg_client_message.ItemInfo, []*msg_client_message.CatInfo, []*msg_client_message.DepotBuildingInfo) {
	total_drop_count := int32(0)
	for n := 0; n < len(items_info)/2; n++ {
		total_drop_count += items_info[2*n+1]
	}

	cats := make([]*msg_client_message.CatInfo, 0, total_drop_count)
	items := make([]*msg_client_message.ItemInfo, 0, total_drop_count)
	buildings := make([]*msg_client_message.DepotBuildingInfo, 0, total_drop_count)

	rand.Seed(time.Now().Unix() + time.Now().UnixNano())
	for i := 0; i < len(items_info)/2; i++ {
		drop_lib := drop_card_table_mgr.Map[items_info[2*i]]
		if nil == drop_lib {
			return false, nil, nil, nil
		}

		for j := 0; j < int(items_info[2*i+1]); j++ {
			tmp_item, tmp_cat, tmp_building := this.drop_item(drop_lib, false, badd)
			if tmp_item != nil {
				items = append(items, tmp_item)
			}
			if tmp_cat != nil {
				cats = append(cats, tmp_cat)
			}
			if tmp_building != nil {
				buildings = append(buildings, tmp_building)
			}
		}
	}
	return true, items, cats, buildings
}

func (this *Player) DropItems3(items_info []int32, items map[int32]*msg_client_message.ItemInfo, cats map[int32]*msg_client_message.CatInfo, buildings map[int32]*msg_client_message.DepotBuildingInfo) bool {
	total_drop_count := int32(0)
	for n := 0; n < len(items_info)/2; n++ {
		total_drop_count += items_info[2*n+1]
	}

	for i := 0; i < len(items_info)/2; i++ {
		draw_lib := drop_card_table_mgr.Map[items_info[2*i]]
		if nil == draw_lib {
			return false
		}

		log.Info("当前抽取库 总数目%d 总权重%d 详细%v", draw_lib.TotalCount, draw_lib.TotalWeight, draw_lib)

		for j := 0; j < int(items_info[2*i+1]); j++ {
			rand.Seed(time.Now().UnixNano() + int64(i+j))
			rand_val := rand.Int31n(draw_lib.TotalWeight)
			var tmp_item *tables.XmlDropCardItem
			for i := int32(0); i < draw_lib.TotalCount; i++ {
				tmp_item = draw_lib.DropItems[i]
				if nil == tmp_item {
					continue
				}

				if tmp_item.Weight > rand_val {
					if nil != item_table_mgr.Map[tmp_item.DropType] {
						_, num := rand31n_from_range(tmp_item.Min, tmp_item.Max)
						items[tmp_item.DropType] = this.AddItem(tmp_item.DropType, num, "draw", "player", true)
					} else if nil != cat_table_mgr.Map[tmp_item.DropType] {
						cats[tmp_item.DropType] = this.AddCat(tmp_item.DropType, "draw", "player", true)
					} else if nil != building_table_mgr.Map[tmp_item.DropType] {
						_, num := rand31n_from_range(tmp_item.Min, tmp_item.Max)
						if this.AddDepotBuilding(tmp_item.DropType, num, "draw", "draw_building", true) {
							buildings[tmp_item.DropType] = &msg_client_message.DepotBuildingInfo{
								CfgId: tmp_item.DropType, Num: num,
							}
						}
					} else {
						log.Error("C2SDrawHandler rand dropid[%d] not item or cat", tmp_item.DropType)
					}
					break
				} else {
					rand_val -= tmp_item.Weight
				}
			}
		}
	}
	return true
}

func (this *Player) DrawCard(draw_type, draw_count int32) int32 {
	extract := extract_table_mgr.Get(draw_type)
	if extract == nil || extract.DropItems == nil {
		log.Error("Player[%v] draw id[%v] not found", this.Id, draw_type)
		return -1
	}

	num := this.GetItemResourceValue(extract.CostId)
	if num < extract.CostNum*draw_count {
		log.Error("Player[%v] draw card need item[%v] num[%v] not enough", this.Id, extract.CostId, extract.CostNum*draw_count)
		return int32(msg_client_message.E_ERR_ITEM_NUM_NOT_ENOUGH)
	}

	var b bool
	res2cli := &msg_client_message.S2CDrawResult{}

	// 首抽
	if !this.db.FirstDrawCards.HasIndex(draw_type) && (extract.FirstDropIds != nil && len(extract.FirstDropIds) > 0) {
		b, res2cli.Items, res2cli.Cats, res2cli.Buildings = this.DropItems2(extract.FirstDropIds, true)
		if !b {
			log.Error("C2SDrawHandler failed to find draw_lib [%d]", draw_type)
			return int32(msg_client_message.E_ERR_DRAW_WRONG_DRAW_TYPE)
		}
		var d dbPlayerFirstDrawCardData
		d.Id = draw_type
		d.Drawed = 1
		this.db.FirstDrawCards.Add(&d)
		res2cli.IsFirst = true
	} else {
		b, res2cli.Items, res2cli.Cats, res2cli.Buildings = this.DropItems(extract.DropItems, draw_count, true)
		if !b {
			log.Error("C2SDrawHandler failed to find draw_lib [%d]", draw_type)
			return int32(msg_client_message.E_ERR_DRAW_WRONG_DRAW_TYPE)
		}
		res2cli.IsFirst = false
	}

	this.RemoveItemResource(extract.CostId, extract.CostNum*draw_count, "draw_card", "draw")

	this.SendCatsUpdate()
	this.SendItemsUpdate()
	this.SendDepotBuildingUpdate()

	this.Send(uint16(msg_client_message.S2CDrawResult_ProtoID), res2cli)

	return 1
}

func C2SDrawHandler(p *Player, msg_data []byte) int32 {
	var req msg_client_message.C2SDraw
	err := proto.Unmarshal(msg_data, &req)
	if err != nil {
		log.Error("unmarshal msg failed %v", err.Error())
		return -1
	}

	draw_type := req.GetDrawType()
	draw_num := req.GetDrawCount()

	return p.DrawCard(draw_type, draw_num)
}
