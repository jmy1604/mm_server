package main

import (
	"math/rand"
	"mm_server/libs/log"
	"mm_server/proto/gen_go/client_message"
	"mm_server/src/tables"
	"time"

	"github.com/golang/protobuf/proto"
)

const (
	COMPOSE_FOSTER_CARD_NUM = 3 // 合成寄养卡需要MAX_的寄养卡数量
	FOSTER_CAT_MAX_NUM      = 6 // 最大寄养猫数
)

func foster_build_self_cats_msg(cat_ids []int32, cat_exps []int32, cat_items []map[int32]int32) (self_cats []*msg_client_message.FosterCatInfo) {
	self_cats = make([]*msg_client_message.FosterCatInfo, len(cat_ids))
	for i := 0; i < len(cat_ids); i++ {
		var items []*msg_client_message.ItemInfo
		if cat_items != nil {
			items = make([]*msg_client_message.ItemInfo, len(cat_items[i]))
			n := 0
			for k, v := range cat_items[i] {
				items[n] = &msg_client_message.ItemInfo{
					ItemCfgId: k,
					ItemNum:   v,
				}
				n += 1
			}
		} else {
			items = make([]*msg_client_message.ItemInfo, 0)
		}
		exp := int32(0)
		if cat_exps != nil {
			exp = cat_exps[i]
		}
		self_cats[i] = &msg_client_message.FosterCatInfo{
			CatId:  cat_ids[i],
			CatExp: exp,
			Items:  items,
		}
	}
	return
}

func reg_player_foster_msg() {
	msg_handler_mgr.SetPlayerMsgHandler(uint16(msg_client_message.C2SPullFosterData_ProtoID), C2SPullFosterDataHandler)
	msg_handler_mgr.SetPlayerMsgHandler(uint16(msg_client_message.C2SFosterEquipCard_ProtoID), C2SFosterEquipCardHandler)
	msg_handler_mgr.SetPlayerMsgHandler(uint16(msg_client_message.C2SFosterUnequipCard_ProtoID), C2SFosterUnequipCardHandler)
	msg_handler_mgr.SetPlayerMsgHandler(uint16(msg_client_message.C2SFosterSetCat_ProtoID), C2SFosterSetCatHandler)
	msg_handler_mgr.SetPlayerMsgHandler(uint16(msg_client_message.C2SFosterOutCat_ProtoID), C2SFosterOutCatHandler)
	msg_handler_mgr.SetPlayerMsgHandler(uint16(msg_client_message.C2SFosterSetCat2Friend_ProtoID), C2SFosterSetCat2FriendHandler)
	msg_handler_mgr.SetPlayerMsgHandler(uint16(msg_client_message.C2SGetPlayerFosterCats_ProtoID), C2SGetPlayerFosterCatsHandler)
	msg_handler_mgr.SetPlayerMsgHandler(uint16(msg_client_message.C2SPullFosterCatsWithFriend_ProtoID), C2SPullFosterDataWithFriendHandler)
	msg_handler_mgr.SetPlayerMsgHandler(uint16(msg_client_message.C2SFosterCardCompose_ProtoID), C2SFosterCardComposeHandler)
	msg_handler_mgr.SetPlayerMsgHandler(uint16(msg_client_message.C2SFosterGetEmptySlotFriends_ProtoID), C2SFosterGetEmptySlotFriendsHandler)
}

func get_foster_card_remain_seconds(card_id int32, card_start_time int32) int32 {
	foster_card := foster_table_mgr.Get(card_id)
	if foster_card == nil {
		return 0
	}
	return GetRemainSeconds(card_start_time, foster_card.FosterTime)
}

// 寄养卡剩余时间
func (this *dbPlayerFosterColumn) GetCardRemainSeconds() (card_remain_seconds int32) {
	this.m_row.m_lock.UnSafeRLock("dbPlayerFosterColumn.GetCardRemainSeconds")
	defer this.m_row.m_lock.UnSafeRUnlock()
	return get_foster_card_remain_seconds(this.m_data.EquippedCardId, this.m_data.StartTime)
}

// 检测是否可放入猫
func (this *dbPlayerFosterCatColumn) CheckAndAddCat(cat_id int32) int32 {
	this.m_row.m_lock.UnSafeLock("dbPlayerFosterCatColumn.CheckAndSetCat")
	defer this.m_row.m_lock.UnSafeUnlock()

	if len(this.m_data) >= FOSTER_CAT_MAX_NUM {
		log.Error("Player[%v] foster not enough space to set cat[%v]", this.m_row.m_PlayerId, cat_id)
		return int32(msg_client_message.E_ERR_FOSTER_NOT_ENOUGH_SPACE_TO_SET_CAT)
	}

	new_cat := &dbPlayerFosterCatData{
		CatId:     cat_id,
		StartTime: int32(time.Now().Unix()),
	}

	this.m_data[cat_id] = new_cat
	this.m_changed = true
	return 1
}

// 结算
func (this *dbPlayerFosterCatColumn) Settlement(is_settle bool, card_start_time int32, card *tables.XmlFosterItem) (cat_ids []int32, cat_exps []int32, cat_items []map[int32]int32) {
	this.m_row.m_lock.UnSafeLock("dbPlayerFosterCatColumn.Settlement")
	defer this.m_row.m_lock.UnSafeUnlock()

	now_time := int32(time.Now().Unix())

	i := int32(0)
	cat_ids = make([]int32, len(this.m_data))
	for k, v := range this.m_data {
		cat_ids[i] = k
		if card != nil {
			use_time := now_time - v.StartTime
			if card_start_time > v.StartTime {
				use_time = now_time - card_start_time
			}
			if use_time < 0 {
				log.Warn("!!!!!! player[%v] foster cat[%v] used time[%v] error[cat_start_time:%v, card_start_time:%v, card_time:%v]",
					this.m_row.m_PlayerId, cat_ids[i], use_time, v.StartTime, card_start_time, card.FosterTime)
				continue
			}
			if use_time > card.FosterTime {
				use_time = card.FosterTime
			}

			for n := 0; n < len(card.Rewards)/2; n++ {
				id := card.Rewards[2*n]
				num := card.Rewards[2*n+1] * use_time / card.FosterTime
				if id == ITEM_RESOURCE_ID_CAT_EXP {
					if cat_exps == nil {
						cat_exps = make([]int32, len(cat_ids))
					}
					cat_exps[i] = num
				} else {
					if cat_items == nil {
						cat_items = make([]map[int32]int32, len(cat_ids))
					}
					if cat_items[i] == nil {
						cat_items[i] = make(map[int32]int32)
					}
					cat_items[i][id] += num
				}

			}
			if is_settle {
				v.StartTime = now_time
			}
		}
		i += 1
	}
	if is_settle {
		this.m_changed = true
	}
	return
}

func (this *dbPlayerFosterFriendCatColumn) Settlement(friend_id int32) (remain_seconds, cat_exp int32, items map[int32]int32) {
	this.m_row.m_lock.UnSafeLock("dbPlayerFosterFriendCatColumn.Settlement")
	defer this.m_row.m_lock.UnSafeUnlock()

	d := this.m_data[friend_id]
	if d == nil {
		log.Error("Player[%v] NO friend[%v] foster cat in db data", this.m_row.m_PlayerId, friend_id)
		return -1, 0, nil
	}

	card := foster_table_mgr.Get(d.StartCardId)
	if card == nil {
		log.Error("foster table NO card[%v]", d.StartCardId)
		return -1, 0, nil
	}

	now_time := int32(time.Now().Unix())
	used_time := now_time - d.StartTime
	if used_time >= global_config.FriendFosterHours*3600 {
		for i := 0; i < len(card.Rewards); i++ {
			id := card.Rewards[2*i]
			num := card.Rewards[2*i+1]
			if id == ITEM_RESOURCE_ID_CAT_EXP {
				cat_exp += num
			} else {
				if items == nil {
					items = make(map[int32]int32)
				}
				items[id] = num
			}
		}
	} else {
		remain_seconds = global_config.FriendFosterHours*3600 - used_time
	}

	return
}

func (this *dbPlayerFosterFriendCatColumn) CheckAndAddFriendCat(friend_id, friend_level int32, friend_name string, friend_head, cat_id, cat_table_id, cat_level, cat_star int32, foster_friend_num int32, card *tables.XmlFosterItem) int32 {
	this.m_row.m_lock.UnSafeLock("dbPlayerFosterFriendCatColumn.CheckAndAddFriendCat")
	defer this.m_row.m_lock.UnSafeUnlock()

	l := len(this.m_data)
	if int32(l) >= foster_friend_num {
		log.Error("Player[%v] foster friend[%v] cat[%v] no enough space", this.m_row.m_PlayerId, friend_id, cat_id)
		return int32(msg_client_message.E_ERR_FOSTER_FRIEND_NO_SPACE_TO_FOSTER)
	}

	if this.m_data[friend_id] != nil {
		log.Error("Player[%v] only set one cat in one friend foster", this.m_row.m_PlayerId)
		return int32(msg_client_message.E_ERR_FOSTER_ALREADY_CAT_IN_THE_FRIEND)
	}

	card_id := int32(0)
	if card != nil {
		card_id = card.Id
	}
	this.m_data[friend_id] = &dbPlayerFosterFriendCatData{
		PlayerId:    friend_id,
		PlayerLevel: friend_level,
		PlayerName:  friend_name,
		PlayerHead:  friend_head,
		CatId:       cat_id,
		CatTableId:  cat_table_id,
		CatLevel:    cat_level,
		CatStar:     cat_star,
		StartCardId: card_id,
		StartTime:   int32(time.Now().Unix()),
	}

	this.m_changed = true

	log.Debug("Player[%v] was fostered friend[%v] cat[%v]", this.m_row.m_PlayerId, friend_id, cat_id)

	return 1
}

func (this *Player) send_foster_self_data(self_cats []*msg_client_message.FosterCatInfo) {
	response := &msg_client_message.S2CPullFosterDataResult{}
	response.BuildingId = this.db.Foster.GetBuildingId()
	response.CardId = this.db.Foster.GetEquippedCardId()
	response.CardRemainSeconds = this.db.Foster.GetCardRemainSeconds()
	response.SelfCats = self_cats
	this.Send(uint16(msg_client_message.S2CPullFosterDataResult_ProtoID), response)
}

// 拉取自己寄养猫
func (this *Player) foster_data_pull(is_settle bool) int32 {
	self_cats, _ := this.foster_settlement_self_cats(is_settle, true, false)
	this.send_foster_self_data(self_cats)
	log.Debug("Player[%v] pull foster data", this.Id)
	return 1
}

// 拉取寄养好友那的猫
func (this *Player) foster_pull_cats_from_friend() (cats_in_friend []*msg_client_message.FosterCatInFriendInfo) {
	cats_in_friend = make([]*msg_client_message.FosterCatInFriendInfo, 0)
	cat_ids := this.db.FosterCatOnFriends.GetAllIndex()
	for i := 0; i < len(cat_ids); i++ {
		friend_id, _ := this.db.FosterCatOnFriends.GetFriendId(cat_ids[i])
		var friend_level, friend_head, start_card, remain_seconds, cat_exp int32
		var friend_name string
		var items map[int32]int32
		friend := player_mgr.GetPlayerById(friend_id)
		if friend != nil {
			friend_level = friend.db.Info.GetLvl()
			friend_name = friend.db.GetName()
			friend_head = friend.db.Info.GetHead()
			start_card, _ = friend.db.FosterFriendCats.GetStartCardId(this.Id)
			remain_seconds, cat_exp, items = friend.db.FosterFriendCats.Settlement(this.Id)
			if remain_seconds <= 0 {
				friend.db.FosterFriendCats.Remove(this.Id)
			}
		} else {
			/*result := this.rpc_call_foster_get_cat_on_friend2(friend_id, cat_ids[i])
			if result == nil {
				log.Error("Player[%v] get cat[%v] from friend[%v] failed by rpc", this.Id, cat_ids[i], friend_id)
				return nil
			}
			friend_level = result.ToFriendLevel
			friend_name = result.ToFriendName
			friend_head = result.ToFriendHead
			start_card = result.StartCardId
			remain_seconds = result.RemainSeconds
			cat_exp = result.FromPlayerCatExp
			items = result.FromPlayerItems*/
		}

		var get_items []*msg_client_message.ItemInfo
		if items != nil {
			get_items = make([]*msg_client_message.ItemInfo, len(items))
			n := 0
			for k, v := range items {
				get_items[n] = &msg_client_message.ItemInfo{
					ItemCfgId: k,
					ItemNum:   v,
				}
			}
		} else {
			get_items = make([]*msg_client_message.ItemInfo, 0)
		}
		d := &msg_client_message.FosterCatInFriendInfo{
			CatId:         cat_ids[i],
			FriendId:      friend_id,
			FriendLevel:   friend_level,
			FriendHead:    friend_head,
			FriendName:    friend_name,
			StartCardId:   start_card,
			RemainSeconds: remain_seconds,
			CatExp:        cat_exp,
			Items:         get_items,
		}
		cats_in_friend = append(cats_in_friend, d)
	}
	return
}

// 拉取寄养在好友和好友寄养的猫
func (this *Player) foster_data_pull_with_friend() int32 {
	response := &msg_client_message.S2CPullFosterCatsWithFriendResult{}
	response.FriendCats = this.foster_settlement_friends_cat()
	response.CatsInFriend = this.foster_pull_cats_from_friend()
	response.FosterFriendSlotNum = this.foster_slot_num_to_friend()
	response.FriendFosteredSlotNum = this.fostered_slot_num_for_friend()
	this.Send(uint16(msg_client_message.S2CPullFosterCatsWithFriendResult_ProtoID), response)
	return 1
}

// 寄养卡合成
func (this *Player) compose_foster_card(card_ids []int32) int32 {
	if card_ids == nil || len(card_ids) < COMPOSE_FOSTER_CARD_NUM {
		log.Error("Player[%v] compose foster card from cards[%v] not enough", this.Id, card_ids)
		return int32(msg_client_message.E_ERR_FOSTER_COMPOSE_NOT_ENOUGH_CARD)
	}

	cards := make(map[int32]int32)
	for i := 0; i < len(card_ids); i++ {
		if item_table_mgr.Map[card_ids[i]] == nil {
			log.Error("not found card item[%v] in table", card_ids[i])
			return int32(msg_client_message.E_ERR_ITEM_TABLE_DATA_NOT_FOUND)
		}
		if cards[card_ids[i]] == 0 {
			cards[card_ids[i]] = 1
		} else {
			cards[card_ids[i]] += 1
		}
	}

	for k, v := range cards {
		log.Debug("@@@@@@ compose source card [%v,%v]", k, v)
		num, o := this.db.Items.GetItemNum(k)
		if !o || num < v {
			log.Error("Player[%v] compose foster card no enough item[%v,%v]", this.Id, k, v)
			return int32(msg_client_message.E_ERR_FOSTER_COMPOSE_NOT_ENOUGH_CARD)
		}
	}

	rand.Seed(time.Now().Unix() + int64(this.Id))

	type_ids := make([]int32, len(card_ids))
	weights := make([]int32, len(card_ids))
	scores := make([]int32, len(card_ids))
	for i := 0; i < len(card_ids); i++ {
		f := foster_table_mgr.Get(card_ids[i])
		if f == nil {
			log.Error("Player[%v] compose foster card from card[%v] invalid", this.Id, card_ids[i])
			return int32(msg_client_message.E_ERR_FOSTER_COMPOSE_CARD_INVALID)
		}

		// 合成类型 权重
		tw := int32(0)
		for n := 0; n < len(f.FusionTypeWeights)/2; n++ {
			tw += f.FusionTypeWeights[2*n+1]
		}
		rw := rand.Int31n(tw)
		for n := 0; n < len(f.FusionTypeWeights)/2; n++ {
			w := f.FusionTypeWeights[2*n+1]
			if w > rw {
				type_ids[i] = f.FusionTypeWeights[2*n]
				weights[i] = w
				break
			}
			rw -= w
		}

		// 合成积分
		tg := int32(0)
		for n := 0; n < len(f.FusionScores)/2; n++ {
			tg += f.FusionScores[2*n+1]
		}
		rg := rand.Int31n(tg)
		for n := 0; n < len(f.FusionScores)/2; n++ {
			g := f.FusionScores[2*n+1]
			if g > rg {
				scores[i] = f.FusionScores[2*n]
				break
			}
			rg -= g
		}
	}

	log.Debug("@@@@@@@@@@@@@@ type_ids: %v", type_ids)

	// 获得寄养卡类型 积分
	tw := int32(0)
	for n := 0; n < len(card_ids); n++ {
		tw += weights[n]
	}
	the_type := int32(0)
	rw := rand.Int31n(tw)
	for i := 0; i < len(card_ids); i++ {
		if rw < weights[i] {
			the_type = type_ids[i]
			break
		}
		rw -= weights[i]
	}
	the_score := int32(0)
	for i := 0; i < len(card_ids); i++ {
		the_score += scores[i]
	}

	titems := foster_table_mgr.GetInnerItems(the_type)
	if titems == nil {
		log.Error("not found foster type[%v]", the_type)
		return int32(msg_client_message.E_ERR_FOSTER_COMPOSE_TYPE_INVALID)
	}

	dest_card := int32(0)
	for i := 0; i < len(titems.Items); i++ {
		if the_score >= titems.Items[i].BeHitScores[0] && the_score <= titems.Items[i].BeHitScores[1] {
			dest_card = titems.Items[i].Id
			break
		}
	}

	if dest_card == 0 {
		log.Error("Player[%v] compose foster card from cards[%v] to result failed", this.Id, card_ids)
		return -1
	}

	for i := 0; i < len(card_ids); i++ {
		this.RemoveItem(card_ids[i], 1, true)
	}
	this.AddItem(dest_card, 1, "foster_compose", "foster", true)
	this.SendItemsUpdate()

	response := &msg_client_message.S2CFosterCardComposeResult{}
	response.ItemIds = card_ids
	response.DestItemTableId = dest_card
	this.Send(uint16(msg_client_message.S2CFosterCardComposeResult_ProtoID), response)

	log.Debug("Player[%v] composed foster card[%v] from source cards[%v]", this.Id, dest_card, card_ids)

	return 1
}

// 结算自己的猫
func (this *Player) foster_settlement_self_cats(is_settle bool, get_cats_msg, get_rewards_msg bool) (self_cats []*msg_client_message.FosterCatInfo, rewards []*msg_client_message.ItemInfo) {
	if this.db.Foster.GetCardRemainSeconds() <= 0 {
		this.db.Foster.SetEquippedCardId(0)
		this.db.Foster.SetStartTime(0)
	}

	equipped_card := this.db.Foster.GetEquippedCardId()
	card := foster_table_mgr.Get(equipped_card)
	card_start_time := this.db.Foster.GetStartTime()
	cat_ids, cat_exps, cat_items := this.db.FosterCats.Settlement(is_settle, card_start_time, card)
	log.Debug("######## cat_ids[%v] cat_exps[%v] cat_items[%v]", cat_ids, cat_exps, cat_items)

	if cat_ids != nil && len(cat_ids) > 0 {
		if is_settle && cat_exps != nil {
			for i := 0; i < len(cat_ids); i++ {
				this.feed_cat(cat_ids[i], 0, cat_exps[i], false)
			}
			this.TaskUpdate(tables.TASK_COMPLETE_TYPE_GET_EXP_BY_FOSTER, false, 0, 1)
		}

		if cat_items != nil {
			if get_rewards_msg {
				rewards = make([]*msg_client_message.ItemInfo, 0)
			}
			for i := 0; i < len(cat_items); i++ {
				items := cat_items[i]
				for k, v := range items {
					if is_settle {
						this.AddItemResource(k, v, "foster_settlement", "foster")
					}
					if get_rewards_msg {
						rewards = append(rewards, &msg_client_message.ItemInfo{
							ItemCfgId: k,
							ItemNum:   v,
						})
					}
				}
			}
			if is_settle {
				this.SendItemsUpdate()
			}
		}
		if get_cats_msg {
			self_cats = foster_build_self_cats_msg(cat_ids, cat_exps, cat_items)
			for i := 0; i < len(self_cats); i++ {
				t, _ := this.db.FosterCats.GetStartTime(cat_ids[i])
				self_cats[i].StartTime = t
			}
		}
	}
	return
}

func (this *Player) foster_build_friends_cats_msg(friend_id int32, remain_seconds int32) *msg_client_message.FosteredFriendCatInfo {
	cat_table_id, _ := this.db.FosterFriendCats.GetCatTableId(friend_id)
	cat_nick, _ := this.db.FosterFriendCats.GetCatNick(friend_id)
	cat_level, _ := this.db.FosterFriendCats.GetCatLevel(friend_id)
	cat_star, _ := this.db.FosterFriendCats.GetCatStar(friend_id)
	friend_name, _ := this.db.FosterFriendCats.GetPlayerName(friend_id)
	friend_level, _ := this.db.FosterFriendCats.GetPlayerLevel(friend_id)
	friend_head, _ := this.db.FosterFriendCats.GetPlayerHead(friend_id)
	card_id, _ := this.db.FosterFriendCats.GetStartCardId(friend_id)
	return &msg_client_message.FosteredFriendCatInfo{
		CatTableId:    cat_table_id,
		CatLevel:      cat_level,
		CatStar:       cat_star,
		CatNick:       cat_nick,
		FriendId:      friend_id,
		FriendName:    friend_name,
		FriendLevel:   friend_level,
		FriendHead:    friend_head,
		StartCardId:   card_id,
		RemainSeconds: remain_seconds,
	}
}

// 结算所有好友寄养猫的收益
func (this *Player) foster_settlement_friends_cat() (friends_cat []*msg_client_message.FosteredFriendCatInfo) {
	friends_cat = make([]*msg_client_message.FosteredFriendCatInfo, 0)
	all_index := this.db.FosterFriendCats.GetAllIndex()
	for i := 0; i < len(all_index); i++ {
		fid := all_index[i]
		cat_id, _ := this.db.FosterFriendCats.GetCatId(fid)
		remain_seconds, cat_exp, items := this.db.FosterFriendCats.Settlement(fid)
		if remain_seconds < 0 {
			log.Error("Player[%v] settle friend[%v] cat[%v] failed", this.Id, fid)
			continue
		}

		// 剩余时间为0 就结算
		if remain_seconds <= 0 {
			friend := player_mgr.GetPlayerById(fid)
			if friend != nil {
				this.settlement_from_friend_foster(cat_id, cat_exp, items)
				this.SendItemsUpdate()
			} else {
				/*if this.rpc_call_foster_settlement_to_friend(all_index[i], cat_id, cat_exp, items) == nil {
					log.Warn("Player[%v] settlement friend[%v] cat[%v] error", this.Id, fid, cat_id)
				}*/
			}
			this.db.FosterFriendCats.Remove(fid)
			continue
		}
		d := this.foster_build_friends_cats_msg(fid, remain_seconds)
		friends_cat = append(friends_cat, d)
	}
	return
}

// 装备寄养卡
func (this *Player) foster_equip_card(building_id int32, card_id int32) int32 {
	/*if !this.db.Buildings.HasIndex(building_id) {
		log.Error("Player[%v] no building[%v]", this.Id, building_id)
		return int32(msg_client_message.E_ERR_BUILDING_NOT_EXIST)
	}

	bcid, _ := this.db.Buildings.GetCfgId(building_id)
	building := cfg_building_mgr.Map[bcid]
	if building == nil {
		log.Error("building[%v] table id[%v] not found", building_id, bcid)
		return int32(msg_client_message.E_ERR_BUILDING_NO_DEPOT_BUILDING)
	}

	if building.Type != PLAYER_BUILDING_TYPE_FOSTER {
		log.Error("Player[%v] building[%v] is not foster", this.Id, building_id)
		return int32(msg_client_message.E_ERR_BUILDING_AREA_TYPE_NOT_MATCH)
	}

	if building_id != this.db.Foster.GetBuildingId() {
		log.Error("Player[%v] foster building[%v] must only one", this.Id, building_id)
		return -1
	}*/

	if this.db.Foster.GetEquippedCardId() > 0 {
		log.Error("Player[%v] foster must unequip old before equip new", this.Id)
		return int32(msg_client_message.E_ERR_FOSTER_EQUIP_NEW_CARD_MUST_UNEQUIP_OLD)
	}

	item := item_table_mgr.Map[card_id]
	if item == nil {
		log.Error("item[%v] table data not found")
		return int32(msg_client_message.E_ERR_ITEM_TABLE_DATA_NOT_FOUND)
	}

	// 寄养卡数量
	if num, o := this.db.Items.GetItemNum(card_id); !o || num < 1 {
		log.Error("Player[%v] not enough item[%v] to equip foster", this.Id, card_id)
		return int32(msg_client_message.E_ERR_ITEM_NUM_NOT_ENOUGH)
	}

	now_time := int32(time.Now().Unix())
	this.db.Foster.SetEquippedCardId(card_id)
	this.db.Foster.SetStartTime(now_time)

	response := &msg_client_message.S2CFosterEquipCardResult{}
	response.BuildingId = building_id
	response.CardId = card_id
	response.CardRemainSeconds = get_foster_card_remain_seconds(card_id, now_time)
	this.Send(uint16(msg_client_message.S2CFosterEquipCardResult_ProtoID), response)

	log.Debug("Player[%v] foster building[%v] equip card[%v]", this.Id, building_id, card_id)

	return 1
}

// 卸载寄养卡
func (this *Player) foster_unequip_card(building_id int32) int32 {
	/*if this.db.Foster.GetBuildingId() != building_id {
		log.Error("Player[%v] foster building[%v] must only one", this.Id, building_id)
		return -1
	}*/

	card_id := this.db.Foster.GetEquippedCardId()
	if card_id <= 0 {
		log.Error("Player[%v] unequip must has equipped card", this.Id)
		return int32(msg_client_message.E_ERR_FOSTER_UNEQUIP_NO_EQUIP_CARD)
	}

	// 先结算
	self_cats, _ := this.foster_settlement_self_cats(true, true, false)
	this.db.Foster.SetEquippedCardId(0)
	this.db.Foster.SetStartTime(0)
	response := &msg_client_message.S2CFosterUnequipCardResult{
		BuildingId: building_id,
	}
	this.Send(uint16(msg_client_message.S2CFosterUnequipCardResult_ProtoID), response)
	this.send_foster_self_data(self_cats)

	log.Debug("Player[%v] unequip foster card[%v]", this.Id, card_id)

	return 1
}

// 猫放入寄养所
func (this *Player) foster_set_cat(building_id int32, cat_id int32) int32 {
	/*if building_id != this.db.Foster.GetBuildingId() {
		log.Error("Player[%v] building[%v] dismatch", this.Id, building_id)
		return int32(msg_client_message.E_ERR_FOSTER_BUILDING_IS_DISMATCH)
	}*/

	if this.IfCatBusy(cat_id) {
		log.Error("Player[%v] cat[%v] is busy, cant set to foster", this.Id, cat_id)
		return int32(msg_client_message.E_ERR_CAT_IS_BUSY)
	}

	res := this.db.FosterCats.CheckAndAddCat(cat_id)
	if res < 0 {
		return res
	}

	this.SendCatUpdate(cat_id)

	response := &msg_client_message.S2CFosterSetCatResult{}
	response.BuildingId = building_id
	response.CatId = cat_id
	this.Send(uint16(msg_client_message.S2CFosterSetCatResult_ProtoID), response)

	this.foster_data_pull(false)

	log.Debug("Player[%v] set cat[%v] in foster", this.Id, cat_id)

	return 1
}

// 猫从寄养所取出
func (this *Player) foster_out_cat(building_id int32, cat_id int32) int32 {
	/*if building_id != this.db.Foster.GetBuildingId() {
		log.Error("Player[%v] buiding[%v] dismatch", this.Id, building_id)
		return int32(msg_client_message.E_ERR_FOSTER_BUILDING_IS_DISMATCH)
	}*/

	if !this.db.FosterCats.HasIndex(cat_id) {
		log.Error("Player[%v] no cat[%v] in foster", this.Id, cat_id)
		return int32(msg_client_message.E_ERR_FOSTER_NO_SUCH_CAT_IN_FOSTER)
	}

	this.db.FosterCats.Remove(cat_id)

	this.SendCatUpdate(cat_id)

	response := &msg_client_message.S2CFosterOutCatResult{
		BuildingId: building_id,
		CatId:      cat_id,
	}
	this.Send(uint16(msg_client_message.S2CFosterOutCatResult_ProtoID), response)

	this.foster_data_pull(false)

	log.Debug("Player[%v] out cat[%v] from foster", this.Id, cat_id)

	return 1
}

func (this *Player) add_friend_cat_in_foster(friend_id, friend_level int32, friend_name string, friend_head, cat_id, cat_table_id, cat_level, cat_star int32) int32 {
	card := foster_table_mgr.Get(this.db.Foster.GetEquippedCardId())
	return this.db.FosterFriendCats.CheckAndAddFriendCat(friend_id, friend_level, friend_name, friend_head, cat_id, cat_table_id, cat_level, cat_star, this.fostered_slot_num_for_friend(), card)
}

func (this *Player) settlement_from_friend_foster(cat_id, cat_exp int32, items map[int32]int32) {
	if cat_exp > 0 {
		this.feed_cat(cat_id, 0, cat_exp, false)
	}
	if items != nil {
		for k, v := range items {
			this.AddItemResource(k, v, "settlement_friend_cat", "foster")
		}
	}
}

// 猫放入好友的寄养所
func (this *Player) foster_set_cat_friend(friend_id int32, cat_id int32) int32 {
	// 达到寄存数上限
	if this.foster_slot_num_to_friend() <= this.db.FosterCatOnFriends.NumAll() {
		log.Error("Player[%v] cant set cat to friend[%v], no empty slot", this.Id, friend_id)
		return int32(msg_client_message.E_ERR_FOSTER_MAX_FRIEND_NUM_TO_FOSTER)
	}

	// 猫不存在
	if !this.db.Cats.HasIndex(cat_id) {
		log.Error("Player[%v] not found cat[%v]", this.Id, cat_id)
		return int32(msg_client_message.E_ERR_CAT_NOT_FOUND)
	}

	// 猫是否忙
	if this.IfCatBusy(cat_id) {
		log.Error("Player[%v] cat[%v] is busy, cant set to friend[%v] foster", this.Id, cat_id, friend_id)
		return int32(msg_client_message.E_ERR_CAT_IS_BUSY)
	}

	// 是否已寄养
	if this.db.FosterCatOnFriends.HasIndex(friend_id) {
		log.Error("Player[%v] already set cat[%v] in friend[%v] foster, cant set other cat", this.Id, cat_id, friend_id)
		return int32(msg_client_message.E_ERR_FOSTER_ALREADY_CAT_IN_THE_FRIEND)
	}

	friend := player_mgr.GetPlayerById(friend_id)
	if friend != nil {
		// 先结算该好友寄养所中其他玩家的猫
		friend.foster_settlement_friends_cat()
		// 放猫
		cat_table_id, _ := this.db.Cats.GetCfgId(cat_id)
		cat_level, _ := this.db.Cats.GetLevel(cat_id)
		cat_star, _ := this.db.Cats.GetStar(cat_id)
		res := friend.add_friend_cat_in_foster(this.Id, this.db.Info.GetLvl(), this.db.GetName(), this.db.Info.GetHead(), cat_id, cat_table_id, cat_level, cat_star)
		if res < 0 {
			log.Error("Player[%v] set cat[%v] to local friend[%v] foster failed", this.Id, cat_id, friend_id)
			return res
		}
	} else {
		// 先结算再放猫
		/*result := this.rpc_call_foster_cat_to_friend(friend_id, cat_id)
		if result == nil {
			log.Error("Player[%v] set cat[%v] to remote friend[%v] foster failed", this.Id, cat_id, friend_id)
			return int32(msg_client_message.E_ERR_FOSTER_SET_CAT_TO_FRIEND_FAILED)
		}*/
	}

	// 加入
	d := &dbPlayerFosterCatOnFriendData{
		FriendId: friend_id,
		CatId:    cat_id,
	}
	this.db.FosterCatOnFriends.Add(d)

	// 同步猫的状态
	this.SendCatUpdate(cat_id)

	response := &msg_client_message.S2CFosterSetCat2FriendResult{
		FriendId: friend_id,
		CatId:    cat_id,
	}
	this.Send(uint16(msg_client_message.S2CFosterSetCat2FriendResult_ProtoID), response)

	log.Debug("Player[%v] set cat[%v] to friend[%v] foster", this.Id, cat_id, friend_id)

	return 1
}

// 玩家提供给好友的槽位数
func (this *Player) fostered_slot_num_for_friend() int32 {
	lvl := this.db.Info.GetLvl()
	player_level := player_level_table_mgr.Map[lvl]
	if player_level == nil {
		return 0
	}
	return player_level.FosteredSlot
}

// 寄养到好友的槽位数
func (this *Player) foster_slot_num_to_friend() int32 {
	lvl := this.db.Info.GetLvl()
	player_level := player_level_table_mgr.Map[lvl]
	if player_level == nil {
		return 0
	}
	return player_level.FosterSlot
}

// 是否有提供给好友的寄养空位
func (this *Player) foster_has_empty_slot_for_friend(friend_id int32) bool {
	// 是否有寄养所
	/*if this.db.Foster.GetBuildingId() <= 0 {
		return false
	}*/
	// 是否已寄养
	if this.db.FosterFriendCats.HasIndex(friend_id) {
		return false
	}
	// 槽位
	slot_num := this.fostered_slot_num_for_friend()
	if this.db.FosterFriendCats.NumAll() >= slot_num {
		return false
	}
	return true
}

// 获取有寄养空位的好友列表
func (this *Player) foster_get_empty_slot_friends() int32 {
	friends := make([]*msg_client_message.FriendInfo, 0)
	friend_ids := this.db.Friends.GetAllIndex()
	if friend_ids != nil || len(friend_ids) > 0 {
		for i := 0; i < len(friend_ids); i++ {
			friend := player_mgr.GetPlayerById(friend_ids[i])
			if friend != nil {
				if !friend.foster_has_empty_slot_for_friend(this.Id) {
					continue
				}
				d := this.db.Friends.GetFriendInfoMsg(friend_ids[i])
				if d != nil {
					friends = append(friends, d)
				}
				d.FosterCardId = friend.db.Foster.GetEquippedCardId()
			} else {
				/*result := this.rpc_call_foster_get_empty_slot_friend_info2(friend_ids[i])
				if result == nil {
					log.Error("Player[%v] get friend[%v] info error by rpc", this.Id, friend_ids[i])
					continue
				}
				d := &msg_client_message.FriendInfo{
					PlayerId:     friend_ids[i],
					Level:        result.ToPlayerLevel,
					Name:         result.ToPlayerName,
					Head:         result.ToPlayerHead,
					VipLevel:     result.ToPlayerVipLevel,
					LastLogin:    result.ToPlayerLastLogin,
					FosterCardId: result.FosterCardId,
				}
				friends = append(friends, d)*/
			}
		}
	}

	response := &msg_client_message.S2CFosterGetEmptySlotFriendsResult{
		Friends: friends,
	}
	this.Send(uint16(msg_client_message.S2CFosterGetEmptySlotFriendsResult_ProtoID), response)
	log.Debug("Player[%v] get empty slot friends[%v]", this.Id, response.Friends)
	return 1
}

func (this *Player) foster_read_cats() (cats []*msg_client_message.FosterPlayerCatInfo) {
	cats = make([]*msg_client_message.FosterPlayerCatInfo, 0)
	cat_ids := this.db.FosterCats.GetAllIndex()
	for i := 0; i < len(cat_ids); i++ {
		if !this.db.Cats.HasIndex(cat_ids[i]) {
			log.Warn("Player[%v] cat[%v] not found", this.Id, cat_ids[i])
			continue
		}
		cat_table_id, _ := this.db.Cats.GetCfgId(cat_ids[i])
		cat_level, _ := this.db.Cats.GetLevel(cat_ids[i])
		cat_star, _ := this.db.Cats.GetStar(cat_ids[i])
		d := &msg_client_message.FosterPlayerCatInfo{
			CatTableId: cat_table_id,
			CatLevel:   cat_level,
			CatStar:    cat_star,
		}
		cats = append(cats, d)
	}
	return
}

func (this *Player) foster_get_cat_id_from_friend(friend_id int32) (cat_id int32) {
	cat_ids := this.db.FosterCatOnFriends.GetAllIndex()
	for i := 0; i < len(cat_ids); i++ {
		fid, _ := this.db.FosterCatOnFriends.GetFriendId(cat_ids[i])
		if fid == friend_id {
			cat_id = cat_ids[i]
			break
		}
	}
	return
}

// 获取玩家的寄养所
func (this *Player) get_player_foster_cats(player_id int32) int32 {
	if player_id == this.Id {
		return -1
	}

	player := player_mgr.GetPlayerById(player_id)
	var card_id int32
	var remain_seconds int32
	var cats []*msg_client_message.FosterPlayerCatInfo
	var friend_cats []*msg_client_message.FosteredFriendCatInfo
	var fostered_num int32
	if player != nil {
		card_id = player.db.Foster.GetEquippedCardId()
		remain_seconds = player.db.Foster.GetCardRemainSeconds()
		cats = player.foster_read_cats()
		friend_cats = player.foster_settlement_friends_cat()
		fostered_num = player.fostered_slot_num_for_friend()
	} else {
		/*result := this.rpc_call_foster_get_player_foster2(player_id)
		if result == nil {
			log.Error("Player[%v] get player[%v] foster data failed", this.Id, player_id)
			return -1
		}
		card_id = result.FosterCardId
		remain_seconds = result.CardRemainSeconds
		for i := 0; i < len(result.PlayerCats); i++ {
			d := &msg_client_message.FosterPlayerCatInfo{
				CatTableId: result.PlayerCats[i].CatTableId,
				CatLevel:   result.PlayerCats[i].CatLevel,
				CatStar:    result.PlayerCats[i].CatStar,
			}
			cats = append(cats, d)
		}
		for i := 0; i < len(result.PlayerFriendCats); i++ {
			c := result.PlayerFriendCats[i]
			d := &msg_client_message.FosteredFriendCatInfo{
				StartCardId:   c.StartCardId,
				RemainSeconds: c.RemainSeconds,
				CatTableId:    c.CatTableId,
				CatNick:       c.CatNick,
				CatLevel:      c.CatLevel,
				CatStar:       c.CatStar,
				FriendId:      c.PlayerId,
				FriendName:    c.PlayerName,
				FriendLevel:   c.PlayerLevel,
				FriendHead:    c.PlayerHead,
			}
			friend_cats = append(friend_cats, d)
		}
		fostered_num = result.FosteredSlot*/
	}

	response := &msg_client_message.S2CGetPlayerFosterCatsResult{
		PlayerId:          player_id,
		FosterCardId:      card_id,
		CardRemainSeconds: remain_seconds,
		Cats:              cats,
		FriendCats:        friend_cats,
		FosteredSlotNum:   fostered_num,
	}
	this.Send(uint16(msg_client_message.S2CGetPlayerFosterCatsResult_ProtoID), response)

	log.Debug("Player[%v] get the player[%v] foster data", this.Id, player_id)

	return 1
}

func C2SPullFosterDataHandler(p *Player, msg_data []byte) int32 {
	var req msg_client_message.C2SPullFosterData
	err := proto.Unmarshal(msg_data, &req)
	if err != nil {
		log.Error("unmarshal msg failed %v", err.Error())
		return -1
	}
	return p.foster_data_pull(req.GetIsSettle())
}

func C2SPullFosterDataWithFriendHandler(p *Player, msg_data []byte) int32 {
	var req msg_client_message.C2SPullFosterCatsWithFriend
	err := proto.Unmarshal(msg_data, &req)
	if err != nil {
		log.Error("unmarshal msg failed %v", err.Error())
		return -1
	}
	return p.foster_data_pull_with_friend()
}

func C2SFosterEquipCardHandler(p *Player, msg_data []byte) int32 {
	var req msg_client_message.C2SFosterEquipCard
	err := proto.Unmarshal(msg_data, &req)
	if err != nil {
		log.Error("unmarshal msg failed %v", err.Error())
		return -1
	}
	return p.foster_equip_card(req.GetBuildingId(), req.GetCardId())
}

func C2SFosterUnequipCardHandler(p *Player, msg_data []byte) int32 {
	var req msg_client_message.C2SFosterUnequipCard
	err := proto.Unmarshal(msg_data, &req)
	if err != nil {
		log.Error("unmarshal msg failed %v", err.Error())
		return -1
	}
	return p.foster_unequip_card(req.GetBuildingId())
}

func C2SFosterSetCatHandler(p *Player, msg_data []byte) int32 {
	var req msg_client_message.C2SFosterSetCat
	err := proto.Unmarshal(msg_data, &req)
	if err != nil {
		log.Error("unmarshal msg failed %v", err.Error())
		return -1
	}
	return p.foster_set_cat(req.GetBuildingId(), req.GetCatId())
}

func C2SFosterOutCatHandler(p *Player, msg_data []byte) int32 {
	var req msg_client_message.C2SFosterOutCat
	err := proto.Unmarshal(msg_data, &req)
	if err != nil {
		log.Error("unmarshal msg failed %v", err.Error())
		return -1
	}
	return p.foster_out_cat(req.GetBuildingId(), req.GetCatId())
}

func C2SFosterSetCat2FriendHandler(p *Player, msg_data []byte) int32 {
	var req msg_client_message.C2SFosterSetCat2Friend
	err := proto.Unmarshal(msg_data, &req)
	if err != nil {
		log.Error("unmarshal msg failed %v", err.Error())
		return -1
	}
	return p.foster_set_cat_friend(req.GetFriendId(), req.GetCatId())
}

func C2SFosterGetEmptySlotFriendsHandler(p *Player, msg_data []byte) int32 {
	var req msg_client_message.C2SFosterGetEmptySlotFriends
	err := proto.Unmarshal(msg_data, &req)
	if err != nil {
		log.Error("unmarshal msg failed %v", err.Error())
		return -1
	}
	return p.foster_get_empty_slot_friends()
}

func C2SGetPlayerFosterCatsHandler(p *Player, msg_data []byte) int32 {
	var req msg_client_message.C2SGetPlayerFosterCats
	err := proto.Unmarshal(msg_data, &req)
	if err != nil {
		log.Error("unmarshal msg failed %v", err.Error())
		return -1
	}
	return p.get_player_foster_cats(req.GetPlayerId())
}

func C2SFosterCardComposeHandler(p *Player, msg_data []byte) int32 {
	var req msg_client_message.C2SFosterCardCompose
	err := proto.Unmarshal(msg_data, &req)
	if err != nil {
		log.Error("unmarshal msg failed %v", err.Error())
		return -1
	}
	return p.compose_foster_card(req.GetItemIds())
}
