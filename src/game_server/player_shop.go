package main

import (
	"mm_server/libs/log"
	"mm_server/libs/utils"
	"mm_server/proto/gen_go/client_message"
	"mm_server/src/tables"
	"strings"
	"time"
	//"github.com/golang/protobuf/proto"
)

func (this *Player) fetch_shop_limit_items(shop_id int32, send_msg bool) int32 {
	shop := shop_table_mgr.GetShop(shop_id)
	if shop == nil {
		log.Error("没有ID为[%v]的商店", shop_id)
		return int32(msg_client_message.E_ERR_SHOP_NOT_FOUND)
	}

	this.check_all_shop_items_refresh(false)

	i := int32(0)
	response := &msg_client_message.S2CShopItemsResult{}
	response.ShopId = shop_id
	response.Items = make([]*msg_client_message.S2CShopItem, shop.GetLimitNum())
	for _, v := range shop.GetItems() {
		if int32(i) >= shop.GetLimitNum() {
			break
		}

		var left_num, save_time int32
		if v.LimitedType == 1 {
			// 全服限定
			/*result := this.rpc_call_get_shop_limited_item(v.Id)
			if result == nil {
				continue
			}
			if result.ErrCode == 1 {
				log.Warn("没有配置ID为[%v]的限时商品", v.Id)
				continue
			} else if result.ErrCode == 2 {
				log.Warn("配置ID为[%v]的限定商品数[%v]量不足", v.Id, result.Num)
				continue
			} else if result.ErrCode == 3 {
				log.Warn("限定[%v]天商品[%v]没有保存时间", v.LimitedTime, v.Id)
				continue
			} else if result.ErrCode != 0 {
				log.Warn("商店不支持该错误码[%v]", result.ErrCode)
				continue
			}
			left_num = result.Num
			save_time = result.SaveTime*/
			save_time = int32(time.Now().Unix())
		} else if v.LimitedType == 2 {
			// 个人限定
			item := this.db.ShopItems.Get(v.Id)
			if item == nil {
				var d dbPlayerShopItemData
				d.Id = v.Id
				d.LeftNum = v.LimitedNumber
				this.db.ShopItems.Add(&d)
				item = this.db.ShopItems.Get(v.Id)
			}
			left_num = item.LeftNum
			o := false
			save_time, o = this.db.ShopLimitedInfos.GetLastSaveTime(v.LimitedTime)
			if !o {
				var ld dbPlayerShopLimitedInfoData
				ld.LastSaveTime = int32(time.Now().Unix())
				ld.LimitedDays = v.LimitedTime
				this.db.ShopLimitedInfos.Add(&ld)
				save_time = ld.LastSaveTime
			}
		} else if v.LimitedType != 0 {
			left_num = -1
			log.Warn("不支持的商店限定类型[%v]", v.LimitedType)
		}

		//utils.GetRemainSeconds2NextDayTime(save_time, v.LimitedTime)
		utils.GetRemainSeconds2NextDayTime(save_time, "")
		response.Items[i] = &msg_client_message.S2CShopItem{}
		response.Items[i].ItemId = v.Id
		response.Items[i].LeftNum = left_num
		//response.Items[i].RemainSeconds = remain_seconds
		i += 1
	}

	if send_msg {
		this.Send(uint16(msg_client_message.S2CShopItemsResult_ProtoID), response)
	}

	return 1
}

func (this *Player) buy_item(item_id int32, num int32, send_msg bool) int32 {
	if num <= 0 {
		return -1
	}

	item := shop_table_mgr.GetItem(item_id)
	if item == nil {
		log.Error("没有商店[%v]商品[%v]", item_id)
		return int32(msg_client_message.E_ERR_SHOP_NOT_FOUND)
	}

	if item.BundleId != "" {
		// 月卡
		if strings.HasSuffix(item.BundleId, "mcard") {
			now_time := int32(time.Now().Unix())
			end_time := this.db.Info.GetVipCardEndDay()
			if now_time > end_time {
				log.Error("Player[%v] month card is using, cant buy another", this.Id)
				return -1
			}
			this.db.Info.SetVipCardEndDay(now_time + 30*24*3600)
			return 2
		}
	} else {
		if item.CostResourceId == ITEM_RESOURCE_ID_DIAMOND {
			if this.GetDiamond() < num*item.CostNum {
				log.Warn("商品[%v]价格高于所持钻石[%v]", item_id, item.CostNum, this.GetDiamond())
				return int32(msg_client_message.E_ERR_SHOP_DIAMOND_NOT_ENOUGH)
			}
		} else if item.CostResourceId == ITEM_RESOURCE_ID_GOLD {
			if this.GetGold() < num*item.CostNum {
				log.Warn("商品[%v]价格高于所持金币[%v]", item_id, item.CostNum, this.GetGold())
				return int32(msg_client_message.E_ERR_SHOP_COIN_NOT_ENOUGH)
			}
		} else if item.CostResourceId == ITEM_RESOURCE_ID_RMB {
			// 人民币需要走支付第三方SDK
			return -1
		} else if item.CostResourceId == ITEM_RESOURCE_ID_CHARM_VALUE {
			if this.db.Info.GetCharmVal() < num*item.CostNum {
				log.Warn("商品[%v]价格[%v]高于所持魅力值[%v]", item_id, item.CostNum, this.db.Info.GetCharmVal())
				return int32(msg_client_message.E_ERR_SHOP_CHARM_NOT_ENOUGH)
			}
		} else if item.CostResourceId == ITEM_RESOURCE_ID_FRIEND_POINT {
			if this.db.Info.GetFriendPoints() < num*item.CostNum {
				log.Warn("商品[%v]价格[%v]高于所持友情点[%v]", item_id, item.CostNum, this.db.Info.GetFriendPoints())
				return int32(msg_client_message.E_ERR_SHOP_FRIEND_POINT_NOT_ENOUGH)
			}
		} else if item.CostResourceId == ITEM_RESOURCE_ID_SOUL_STONE {
			if this.db.Info.GetSoulStone() < num*item.CostNum {
				log.Warn("商品[%v]价格[%v]高于所持魂石数[%v]", item_id, item.CostNum, this.db.Info.GetSoulStone())
				return int32(msg_client_message.E_ERR_SHOP_SOUL_STONE_NOT_ENOUGH)
			}
		} else if item.CostResourceId == ITEM_RESOURCE_ID_CHARM_MEDAL {
			if this.db.Info.GetCharmMedal() < num*item.CostNum {
				log.Warn("商品[%v]价格[%v]高于所持魅力勋章数[%v]", item_id, item.CostNum, this.db.Info.GetCharmMedal())
				return int32(msg_client_message.E_ERR_SHOP_CHARM_MEDAL_NOT_ENOUGH)
			}
		} else {
			log.Warn("商品[%v]的支付类型[%v]不支持", item_id, item.CostResourceId)
			return int32(msg_client_message.E_ERR_SHOP_PURCHASE_TYPE_INVALID)
		}
	}

	//left_num := int32(0)
	if item.LimitedType == 0 {
		// 不限量
	} else if item.LimitedType == 1 {
		// 全服限定
		/*result := this.rpc_call_buy_shop_limited_item(item_id, num)
		if result == nil {
			return int32(msg_client_message.E_ERR_SHOP_PURCHASED_FAILED)
		}
		if result.ErrCode == 1 {
			return int32(msg_client_message.E_ERR_SHOP_ITEM_NOT_FOUND)
		} else if result.ErrCode == 2 || result.ErrCode == 3 {
			return int32(msg_client_message.E_ERR_SHOP_ITEM_NOT_ENOUGH)
		} else if result.ErrCode > 0 {
			log.Error("!!!!! 商店不存在的错误码[%v]", result.ErrCode)
			return -4
		}
		left_num = result.LeftNum*/
	} else if item.LimitedType == 2 {
		// 个人限定
		item_data := this.db.ShopItems.Get(item_id)
		if item_data == nil {
			log.Error("商品[%v]不存在", item_id)
			return int32(msg_client_message.E_ERR_SHOP_ITEM_NOT_FOUND)
		}
		if item_data.LeftNum < num*item.Number {
			log.Error("商品[%v]数量[%v]不足", item_id, item_data.LeftNum)
			return int32(msg_client_message.E_ERR_SHOP_ITEM_NOT_ENOUGH)
		}
		//left_num = this.db.ShopItems.IncbyLeftNum(item_id, -num*item.Number)
	} else {
		return int32(msg_client_message.E_ERR_SHOP_LIMITED_TYPE_INVALID)
	}

	for n := 0; n < len(item.BagItems)/2; n++ {
		this.AddItemResource(item.BagItems[2*n], num*item.BagItems[2*n+1], "shop", "buy_shop_item")
	}
	this.SendItemsUpdate()
	this.SendCatsUpdate()
	this.SendDepotBuildingUpdate()

	// 花费资源
	if item.CostResourceId == ITEM_RESOURCE_ID_GOLD {
		this.SubGold(num*item.CostNum, "buy_shop_item", "shop")
	} else if item.CostResourceId == ITEM_RESOURCE_ID_DIAMOND {
		this.SubDiamond(num*item.CostNum, "buy_shop_item", "shop")
	} else if item.CostResourceId == ITEM_RESOURCE_ID_RMB {
		// 人民币另外处理
	} else if item.CostResourceId == ITEM_RESOURCE_ID_CHARM_VALUE {
		this.SubCharmVal(num*item.CostNum, "buy_shop_item", "shop")
	} else if item.CostResourceId == ITEM_RESOURCE_ID_FRIEND_POINT {
		this.SubFriendPoints(num*item.CostNum, "buy_shop_item", "shop")
	} else if item.CostResourceId == ITEM_RESOURCE_ID_CHARM_MEDAL {
		this.SubCharmMedal(num*item.CostNum, "buy_shop_item", "shop")
	} else if item.CostResourceId == ITEM_RESOURCE_ID_SOUL_STONE {
		this.SubSoulStone(num*item.CostNum, "buy_shop_item", "shop")
	}

	if send_msg {
		response := &msg_client_message.S2CBuyShopItemResult{}
		response.ShopId = item.Type
		response.ItemId = item_id
		response.ItemNum = num
		response.AddItem = &msg_client_message.ItemInfo{}
		response.AddItem.ItemCfgId = item_id
		response.AddItem.ItemNum = item.Number
		response.CostRes = &msg_client_message.ResourceInfo{}
		response.CostRes.ResourceType = item.CostResourceId
		response.CostRes.ResourceValue = item.CostNum
		this.Send(uint16(msg_client_message.S2CBuyShopItemResult_ProtoID), response)
	}

	return 1
}

func (this *Player) refresh_shop() int32 {
	all_ids := this.db.ShopItems.GetAllIndex()
	if all_ids == nil || len(all_ids) == 0 {
		return 0
	}
	for _, id := range all_ids {
		item := this.db.ShopItems.Get(id)
		if item == nil {
			log.Warn("刷新个人限定商店物品[%v]失败，该物品不存在", id)
			continue
		}
		citem := shop_table_mgr.GetItem(id)
		if citem == nil {
			log.Warn("商店物品[%v]配置不存在", id)
			continue
		}
		item.LeftNum = citem.LimitedNumber
	}
	return 1
}

func (this *Player) check_limited_days_shop_items(days int32, limited_items *tables.ShopLimitedItems) bool {
	// 刷新全服商品
	if limited_items.GlobalItemsId != nil && len(limited_items.GlobalItemsId) > 0 {
		/*err := this.rpc_call_check_refresh_shop_limited_item(days)
		if err != nil {
			log.Error("Refresh Global shop limited items failed[%v]", err.Error())
		}*/
	}

	// 刷新个人商品
	/*last_save_time, o := this.db.ShopLimitedInfos.GetLastSaveTime(days)
	if !o {
		return false
	}
	if ! .IsArrival(last_save_time, days) {
		return false
	}*/
	for _, pv := range limited_items.PersonalItems {
		this.db.ShopItems.SetLeftNum(pv.Id, pv.LimitedNumber)
	}
	this.db.ShopLimitedInfos.SetLastSaveTime(days, int32(time.Now().Unix()))

	return true
}

// 检测更新所有的商店限时物品
func (this *Player) check_all_shop_items_refresh(send_msg bool) bool {
	all_limited_items := shop_table_mgr.GetAllLimitedItems4Days()
	if all_limited_items == nil {
		return false
	}

	is_refresh := false
	for days, item_arr := range all_limited_items {
		if this.check_limited_days_shop_items(days, item_arr) {
			is_refresh = true
		}
	}
	return is_refresh
}

func (this *Player) check_shop_limited_days_items_refresh_by_shop_itemid(shop_item_id int32, send_msg bool) bool {
	/*if !check_week_time_arrival(this.db.Info.GetLastRefreshShopTime(), global_config.ShopRefreshTime) {
		return false
	}
	// 刷新自己的商店
	if this.refresh_shop() < 0 {
		log.Error("刷新自己的商店失败")
		return false
	}
	// 刷新全服商店
	err := this.rpc_call_refresh_shop_limited_item()
	if err != nil {
		log.Error("rpc调用刷新商店失败")
		return false
	}

	this.db.Info.SetLastRefreshShopTime(int32(time.Now().Unix()))*/

	citem := shop_table_mgr.GetItem(shop_item_id)
	if citem == nil {
		return false
	}

	limited_items := shop_table_mgr.GetLimitedItems4Days(citem.LimitedTime)
	if limited_items == nil {
		return false
	}

	if !this.check_limited_days_shop_items(citem.LimitedTime, limited_items) {
		return false
	}

	if send_msg {
		notify := msg_client_message.S2CShopNeedRefreshNotify{}
		this.Send(uint16(msg_client_message.S2CShopNeedRefreshNotify_ProtoID), &notify)
	}
	return true
}
