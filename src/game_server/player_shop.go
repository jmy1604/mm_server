package main

import (
	"mm_server/libs/log"
	"mm_server/libs/utils"
	"mm_server/proto/gen_go/client_message"

	"mm_server/src/tables"
	"strings"
	"time"
)

func (this *Player) shop_refresh(shop_data *tables.ShopData) {
	items := shop_data.GetItems()
	if items == nil {
		return
	}

	for _, item := range items {
		if !this.db.ShopItems.HasIndex(item.Id) {
			this.db.ShopItems.Add(&dbPlayerShopItemData{
				Id:      item.Id,
				LeftNum: item.LimitedNumber,
			})
		} else {
			this.db.ShopItems.SetLeftNum(item.Id, item.LimitedNumber)
		}
	}
}

func (this *Player) check_shop_refresh(shop_id int32, shop_data *tables.ShopData) (refreshed bool, remain_seconds int64) {
	shop_type := shoptype_table_mgr.GetShopType(shop_id)
	if shop_type == nil {
		log.Error("没有ID为[%v]的商店", shop_id)
		return false, -1
	}
	if shop_type.AutoRefreshTime == "" || shop_type.RefreshDays == 0 {
		return false, 0
	}

	var err error
	now_time := time.Now()
	if !this.db.Shops.HasIndex(shop_id) {
		remain_seconds, err = utils.GetRemainSeconds2NextDaysPoint(0, 0, shop_type.AutoRefreshTime, shop_type.RefreshDays)
		if err != nil {
			log.Error("get remain seconds err %v", err.Error())
			return false, -1
		}
		this.db.Shops.Add(&dbPlayerShopData{
			Id:                    shop_id,
			FirstRefreshTimePoint: int32(now_time.Unix()),
		})
		this.shop_refresh(shop_data)
		refreshed = true
	} else {
		first_refresh, _ := this.db.Shops.GetFirstRefreshTimePoint(shop_id)
		lastest_refresh, _ := this.db.Shops.GetLastestRefreshTimePoint(shop_id)
		remain_seconds, err = utils.GetRemainSeconds2NextDaysPoint(int64(first_refresh), int64(lastest_refresh), shop_type.AutoRefreshTime, shop_type.RefreshDays)
		if err != nil {
			log.Error("get remain seconds err %v", err.Error())
			return false, -1
		}
		if remain_seconds <= 0 {
			this.shop_refresh(shop_data)
			this.db.Shops.SetLastestRefreshTimePoint(shop_id, int32(now_time.Unix()))
			refreshed = true
		}
	}
	return
}

func (this *Player) shop_send_items(shop_id int32, shop *tables.ShopData, remain_seconds int32) {
	i := int32(0)
	response := &msg_client_message.S2CShopItemsResult{}
	response.ShopId = shop_id
	response.RemainSeconds = remain_seconds
	response.Items = make([]*msg_client_message.S2CShopItem, shop.GetLimitNum())
	for _, v := range shop.GetItems() {
		if int32(i) >= shop.GetLimitNum() {
			break
		}

		var left_num int32
		if v.LimitedType == 1 {

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
		} else if v.LimitedType != 0 {
			left_num = -1
			log.Warn("不支持的商店限定类型[%v]", v.LimitedType)
		}
		response.Items[i] = &msg_client_message.S2CShopItem{}
		response.Items[i].ItemId = v.Id
		response.Items[i].LeftNum = left_num
		i += 1
	}

	this.Send(uint16(msg_client_message.S2CShopItemsResult_ProtoID), response)
	log.Trace("Player %v shop %v", this.Id, response)
}

func (this *Player) fetch_shop_limit_items(shop_id int32, send_msg bool) int32 {
	shop := shop_table_mgr.GetShop(shop_id)
	if shop == nil {
		log.Error("没有ID为[%v]的商店", shop_id)
		return int32(msg_client_message.E_ERR_SHOP_NOT_FOUND)
	}

	_, remain_seconds := this.check_shop_refresh(shop_id, shop)
	if remain_seconds < 0 {
		log.Error("没有ID为[%v]的商店类型", shop_id)
		return int32(msg_client_message.E_ERR_SHOP_NOT_FOUND)
	}

	if send_msg {
		this.shop_send_items(shop_id, shop, int32(remain_seconds))
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

	shop_data := shop_table_mgr.GetShop(item.Type)
	if shop_data == nil {
		log.Error("没有商店[%v]商品[%v]", item_id)
		return int32(msg_client_message.E_ERR_SHOP_NOT_FOUND)
	}

	refreshed, remain_seconds := this.check_shop_refresh(item.Type, shop_data)
	if refreshed {
		this.shop_send_items(item.Type, shop_data, int32(remain_seconds))
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

	if item.LimitedType == 0 {
		// 不限量
	} else if item.LimitedType == 1 {

	} else if item.LimitedType == 2 {
		// 个人限定
		item_data := this.db.ShopItems.Get(item_id)
		if item_data == nil {
			log.Error("商品[%v]不存在", item_id)
			return int32(msg_client_message.E_ERR_SHOP_ITEM_NOT_FOUND)
		}
		if item_data.LeftNum < 1 {
			log.Error("商品[%v]数量[%v]不足", item_id, item_data.LeftNum)
			return int32(msg_client_message.E_ERR_SHOP_ITEM_NOT_ENOUGH)
		}
		this.db.ShopItems.IncbyLeftNum(item_id, -1)
	} else {
		return int32(msg_client_message.E_ERR_SHOP_LIMITED_TYPE_INVALID)
	}

	for n := 0; n < len(item.BagItems)/2; n++ {
		this.AddItemResource(item.BagItems[2*n], num*item.Number*item.BagItems[2*n+1], "shop", "buy_shop_item")
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
		response.AddItem.ItemNum = num
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
