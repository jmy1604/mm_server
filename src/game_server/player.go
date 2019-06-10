package main

import (
	"mm_server/libs/log"
	"mm_server/proto/gen_go/client_message"
	"mm_server/proto/gen_go/server_message"
	"mm_server/src/common"
	"mm_server/src/tables"
	"sync"
	"sync/atomic"
	"time"

	"github.com/golang/protobuf/proto"
)

const (
	INIT_PLAYER_MSG_NUM = 10

	MSG_ITEM_HEAD_LEN = 4

	BUILDING_ADD_MSG_INIT_LEN = 5
	BUILDING_ADD_MSG_ADD_STEP = 2
)

type PlayerMsgItem struct {
	data          []byte
	data_len      int32
	data_head_len int32
	msg_code      uint16
}

type Player struct {
	UniqueId      string
	Id            int32
	Account       string
	Token         string
	ol_array_idx  int32
	all_array_idx int32

	db *dbPlayerRow

	inited  bool
	is_lock int32

	is_login int32 // 是否在线

	pos                int32
	msg_items          []*PlayerMsgItem
	msg_items_lock     *sync.Mutex
	cur_msg_items_len  int32
	max_msg_items_len  int32
	total_msg_data_len int32

	b_cur_building_map_init      bool
	b_cur_building_map_init_lock *sync.Mutex
	cur_area_map_lock            *sync.RWMutex
	cur_building_map             map[int32]int32
	cur_open_pos_map             map[int32]int32
	cur_areablocknum_map         map[int32]int32

	b_base_prop_chg bool

	item_cat_building_change_info ItemCatBuildingChangeInfo // 物品猫建筑数量状态变化

	new_unlock_chapter_id int32

	used_drop_ids    map[int32]int32       // 抽卡掉落ID统计
	world_chat_data  PlayerChatData        // 世界聊天缓存数据
	system_chat_data PlayerChatData        // 系统聊天缓存数据
	anouncement_data PlayerAnouncementData // 公告缓存数据

	stage_id     int32
	stage_cat_id int32
	stage_state  int32

	is_paying int32 // 是否正在支付

	new_mail_list_locker sync.RWMutex // 新邮件列表锁
	new_mail_ids         []int32      // 新邮件ID列表
	receive_mail_locker  sync.RWMutex // 接受邮件锁

	surface_data        map[int32]map[int32]int32 // 地块
	surface_data_locker sync.RWMutex

	msg_acts_lock    sync.Mutex
	cur_msg_acts_len int32
	max_msg_acts_len int32
	msg_acts         []*msg_client_message.ActivityInfo
}

func new_player(id int32, uid, account, token string, db *dbPlayerRow) *Player {
	ret_p := &Player{}
	ret_p.UniqueId = uid
	ret_p.Id = id
	ret_p.Account = account
	ret_p.db = db
	ret_p._init()

	return ret_p
}

func new_player_with_db(id int32, db *dbPlayerRow) *Player {
	if id <= 0 || nil == db {
		log.Error("new_player_with_db param error !", id, nil == db)
		return nil
	}

	ret_p := &Player{}
	ret_p.Id = id
	ret_p.db = db

	ret_p.Account = db.GetAccount()
	ret_p.UniqueId = db.GetUniqueId()

	ret_p._init()

	return ret_p
}

func (this *Player) _init() {
	this.max_msg_items_len = INIT_PLAYER_MSG_NUM
	this.msg_items_lock = &sync.Mutex{}
	this.msg_items = make([]*PlayerMsgItem, this.max_msg_items_len)

	this.b_cur_building_map_init_lock = &sync.Mutex{}
	this.cur_building_map = make(map[int32]int32)
	this.cur_open_pos_map = make(map[int32]int32)
	this.cur_areablocknum_map = make(map[int32]int32)
	this.cur_area_map_lock = &sync.RWMutex{}

	this.item_cat_building_change_info.init()
}

func (this *Player) add_msg_data(msg_code uint16, data []byte) {
	if nil == data {
		log.Error("Player add_msg_data !")
		return
	}

	this.msg_items_lock.Lock()
	defer this.msg_items_lock.Unlock()

	if this.cur_msg_items_len >= this.max_msg_items_len {
		new_max := this.max_msg_items_len + 5
		new_msg_items := make([]*PlayerMsgItem, new_max)
		for idx := int32(0); idx < this.max_msg_items_len; idx++ {
			new_msg_items[idx] = this.msg_items[idx]
		}

		this.msg_items = new_msg_items
		this.max_msg_items_len = new_max
	}

	new_item := &PlayerMsgItem{}
	new_item.msg_code = msg_code
	new_item.data = data
	new_item.data_len = int32(len(data))
	this.total_msg_data_len += new_item.data_len + MSG_ITEM_HEAD_LEN
	this.msg_items[this.cur_msg_items_len] = new_item

	this.cur_msg_items_len++

	return
}

func (this *Player) PopCurMsgData() []byte {
	if this.b_base_prop_chg {
		this.send_info()
	}

	if this.ChkMapBlock() > 0 {
		this.item_cat_building_change_info.send_buildings_update(this)
	}

	if this.ChkMapChest() > 0 {
		this.item_cat_building_change_info.send_buildings_update(this)
	}

	this.ChkSendNewUnlockStage()

	this.CheckAndAnouncement()

	this.CheckNewMail()

	this.msg_items_lock.Lock()
	defer this.msg_items_lock.Unlock()

	out_bytes := make([]byte, this.total_msg_data_len)
	tmp_len := int32(0)
	var tmp_item *PlayerMsgItem
	for idx := int32(0); idx < this.cur_msg_items_len; idx++ {
		tmp_item = this.msg_items[idx]
		if nil == tmp_item {
			continue
		}

		out_bytes[tmp_len] = byte(tmp_item.msg_code >> 8)
		out_bytes[tmp_len+1] = byte(tmp_item.msg_code & 0xFF)
		out_bytes[tmp_len+2] = byte(tmp_item.data_len >> 8)
		out_bytes[tmp_len+3] = byte(tmp_item.data_len & 0xFF)
		tmp_len += 4
		copy(out_bytes[tmp_len:], tmp_item.data)
		tmp_len += tmp_item.data_len
	}

	this.cur_msg_items_len = 0
	this.total_msg_data_len = 0
	return out_bytes
}

func (this *Player) Send(msg_id uint16, msg proto.Message) {
	//log.Info("[发送] [玩家%d:%s] [%s] !", this.Id, msg.MessageTypeName(), msg.String())

	data, err := proto.Marshal(msg)
	if nil != err {
		log.Error("Player Marshal msg failed err[%s] !", err.Error())
		return
	}

	this.add_msg_data(msg_id, data)
}

func (this *Player) add_all_items() {
	for i := 0; i < len(item_table_mgr.Array); i++ {
		c := item_table_mgr.Array[i]
		this.AddItem(c.CfgId, c.MaxNumber, "on_create", "player", true)
	}
	this.SendItemsUpdate()
}

func (this *Player) OnCreate() {
	// 随机初始名称
	this.first_gen_achieve_tasks()
	this.db.Info.SetHead(global_config.InitHead)
	this.db.SetLevel(1)
	this.db.Info.SetCreateUnix(int32(time.Now().Unix()))

	// 给予初始金币
	this.db.Info.SetGold(global_config.InitCoin)
	this.db.Info.SetDiamond(global_config.InitDiamond)

	// 设置初始解锁关卡
	this.db.Info.SetMaxChapter(chapter_table_mgr.InitChapterId)
	//this.db.Info.SetCurMaxStage(cfg_chapter_mgr.InitStageId)
	this.db.Info.SetMaxUnlockStage(chapter_table_mgr.InitMaxStage)
	this.db.Info.SetCurPassMaxStage(0)

	// 添加初始物品
	for i := int32(0); i < global_config.InitItem_len; i++ {
		tmp_cfgidnum := &global_config.InitItems[i]
		this.AddItemResource(tmp_cfgidnum.CfgId, tmp_cfgidnum.Num, "on_create", "player")
	}

	// 添加猫
	for i := int32(0); i < global_config.InitCats_len; i++ {
		tmp_cfgidnum := &global_config.InitCats[i]
		this.AddCat(tmp_cfgidnum.CfgId, "on_create", "player", true)
	}

	// 初始化默认建筑
	this.InitPlayerArea()
	this.ChkUpdateMyBuildingAreas()

	// 初始配方
	init_formulas := global_config.InitFormulas
	if init_formulas != nil {
		for i := 0; i < len(init_formulas); i++ {
			f := formula_table_mgr.Map[init_formulas[i]]
			if f == nil {
				log.Error("没有建筑配方[%v]配置", init_formulas[i])
				continue
			}
			var data dbPlayerDepotBuildingFormulaData
			data.Id = init_formulas[i]
			this.db.DepotBuildingFormulas.Add(&data)
		}
	}

	// 初始建筑
	init_buildings := global_config.InitBuildings
	if init_buildings != nil {
		for i := 0; i < len(init_buildings)/2; i++ {
			this.AddDepotBuilding(init_buildings[2*i], init_buildings[2*i+1], "on_create", "player", false)
		}
	}

	this.rpc_player_base_info_update()

	return
}

func (this *Player) OnInit() {
	if this.inited {
		return
	}
	this.inited = true
}

func (this *Player) OnLogin() {
	this.ChkDayHelpUnlockNum(true)

	this.db.Info.SetLastLogin(int32(time.Now().Unix()))
	atomic.StoreInt32(&this.is_lock, 0)
	atomic.StoreInt32(&this.is_login, 1)

	log.Trace("Player[%v] login", this.Id)
}

func (this *Player) OnLogout(remove_timer bool) {
	if remove_timer {
		if USE_CONN_TIMER_WHEEL == 0 {
			conn_timer_mgr.Remove(this.Id)
		} else {
			conn_timer_wheel.Remove(this.Id)
		}
	}

	if atomic.CompareAndSwapInt32(&this.is_login, 1, 0) {
		// 离线收益时间开始
		this.db.Info.SetLastLogout(int32(time.Now().Unix()))

		var notify msg_server_message.G2LAccountLogoutNotify
		notify.Account = this.Account
		login_conn_mgr.Send(uint16(msg_server_message.MSGID_G2L_ACCOUNT_LOGOUT_NOTIFY), &notify)
		log.Trace("Player[%v] log out !!!", this.Id)
	} else {
		log.Warn("Player[%v] already loged out", this.Id)
	}

	log.Info("玩家[%d] 登出 ！！", this.Id)
}

func (this *Player) IsOffline() bool {
	return atomic.LoadInt32(&this.is_login) == 0
}

func (this *Player) send_enter_game(acc string, id int32) {
	res := &msg_client_message.S2CEnterGameResponse{}
	res.Acc = acc
	res.PlayerId = id
	this.Send(uint16(msg_client_message.S2CEnterGameResponse_ProtoID), res)
	if id <= 0 {
		log.Error("Player[%v] enter game id is invalid %v", acc, id)
	}
}

func (this *Player) send_info() {
	var stamina_remain_secs int32
	stamina := this.CalcSpirit(&stamina_remain_secs)
	response := &msg_client_message.S2CPlayerInfoResponse{
		Level:                 this.db.GetLevel(),
		Exp:                   this.db.Info.GetExp(),
		Gold:                  this.db.Info.GetGold(),
		Diamond:               this.db.Info.GetDiamond(),
		Head:                  this.db.Info.GetHead(),
		VipLevel:              this.db.Info.GetVipLvl(),
		Name:                  this.db.GetName(),
		SysTime:               int32(time.Now().Unix()),
		Star:                  this.db.Info.GetTotalStars(),
		CurMaxStage:           this.db.Info.GetCurMaxStage(),
		CurUnlockMaxStage:     this.db.Info.GetMaxUnlockStage(),
		CharmVal:              this.db.Info.GetCharmVal(),
		CatFood:               this.db.Info.GetCatFood(),
		Zan:                   this.db.Info.GetZan(),
		FriendPoints:          this.db.Info.GetFriendPoints(),
		SoulStone:             this.db.Info.GetSoulStone(),
		Spirit:                stamina,
		NextStaminaRemainSecs: stamina_remain_secs,
		CharmMetal:            this.db.Info.GetCharmMedal(),
		HistoricalMaxStar:     this.db.Stages.GetTotalTopStar(),
		ChangeNameNum:         this.db.Info.GetChangeNameCount(),
		ChangeNameCostDiamond: global_config.ChangeNameCostDiamond,
		ChangeNameFreeNum:     global_config.ChangeNameFreeNum,
	}
	this.Send(uint16(msg_client_message.S2CPlayerInfoResponse_ProtoID), response)
	log.Trace("Player[%v] info: %v", this.Id, response)
}

func (this *Player) notify_enter_complete() {
	msg := &msg_client_message.S2CEnterGameCompleteNotify{}
	this.Send(uint16(msg_client_message.S2CEnterGameCompleteNotify_ProtoID), msg)
}

func (this *Player) change_head(new_head int32) int32 {
	if new_head > 0 {
		head := item_table_mgr.Get(new_head)
		if head == nil {
			log.Error("head[%v] table data not found", new_head)
			return int32(msg_client_message.E_ERR_PLAYER_HEAD_TABLE_DATA_NOT_FOUND)
		}

		if head.Type != ITEM_TYPE_HEAD {
			log.Error("item[%v] type is not head", new_head)
			return -1
		}
	}

	/*if this.get_resource(new_head) < 1 {
		log.Error("Player[%v] no head %v", this.Id, new_head)
		return int32(msg_client_message.E_ERR_PLAYER_NO_SUCH_HEAD)
	}*/

	this.db.Info.SetHead(new_head)

	response := &msg_client_message.S2CPlayerChangeHeadResponse{
		NewHead: new_head,
	}
	this.Send(uint16(msg_client_message.S2CPlayerChangeHeadResponse_ProtoID), response)

	log.Trace("Player[%v] changed to head[%v]", this.Id, new_head)

	return 1
}

func (this *Player) send_red_point_states(modules []int32) int32 {
	var states = make([]int32, msg_client_message.RED_POINT_MAX)
	var response = msg_client_message.S2CRedPointStatesResponse{
		Modules: modules,
		States:  states,
	}
	this.Send(uint16(msg_client_message.S2CRedPointStatesResponse_ProtoID), &response)

	return 1
}

// ----------------------------------------------------------------------------

// ======================================================================

func C2SGetItemInfosHandler(p *Player, msg_data []byte) int32 {
	var req msg_client_message.C2SGetItemInfos
	err := proto.Unmarshal(msg_data, &req)
	if err != nil {
		log.Error("unmarshal msg failed err(%s) !", err.Error())
		return -1
	}

	msg := &msg_client_message.S2CGetItemInfos{}

	log.Info("GetItem %v res %v", p.db.Items.GetAll(), msg)
	p.Send(uint16(msg_client_message.S2CGetItemInfos_ProtoID), msg)

	return 1
}

func C2SGetDepotBuildingInfosHandler(p *Player, msg_data []byte) int32 {
	var req msg_client_message.C2SGetDepotBuildingInfos
	err := proto.Unmarshal(msg_data, &req)
	if err != nil {
		log.Error("unmarshal msg failed err(%s) !", err.Error())
		return -1
	}

	msg := &msg_client_message.S2CGetDepotBuildingInfos{}
	p.db.BuildingDepots.FillAllMsg(msg)
	p.Send(uint16(msg_client_message.S2CGetDepotBuildingInfos_ProtoID), msg)
	return 1
}

func C2SGetCatInfosHandler(p *Player, msg_data []byte) int32 {
	var req msg_client_message.C2SGetCatInfos
	err := proto.Unmarshal(msg_data, &req)
	if err != nil {
		log.Error("unmarshal msg failed err(%s) !", err.Error())
		return -1
	}

	msg := &msg_client_message.S2CGetCatInfos{}
	p.db.Cats.FillAllMsg(msg)

	cats := msg.GetCats()
	if cats != nil {
		for i := 0; i < len(cats); i++ {
			cats[i].State = p.GetCatState(cats[i].GetId())
		}
	}
	p.Send(uint16(msg_client_message.S2CGetCatInfos_ProtoID), msg)

	return 1
}

func C2SGetStageInfosHandler(p *Player, msg_data []byte) int32 {
	var req msg_client_message.C2SGetStageInfos
	err := proto.Unmarshal(msg_data, &req)
	if err != nil {
		log.Error("unmarshal msg failed err(%s) !", err.Error())
		return -1
	}

	response := &msg_client_message.S2CGetStageInfos{}
	p.db.Stages.FillAllMsg(response)
	p.Send(uint16(msg_client_message.S2CGetStageInfos_ProtoID), response)

	return 1
}

func C2SGetOptionsHandler(p *Player, msg_data []byte) int32 {
	var req msg_client_message.C2SGetOptions
	err := proto.Unmarshal(msg_data, &req)
	if err != nil {
		log.Error("unmarshal msg failed err(%s) !", err.Error())
		return -1
	}

	msg := &msg_client_message.S2CRetOptions{}
	msg.Values = p.db.Options.GetValues()

	p.Send(uint16(msg_client_message.S2CRetOptions_ProtoID), msg)

	return 1
}

func C2SSaveOptionsHandler(p *Player, msg_data []byte) int32 {
	var req msg_client_message.C2SSaveOptions
	err := proto.Unmarshal(msg_data, &req)
	if err != nil {
		log.Error("unmarshal msg failed err(%s) !", err.Error())
		return -1
	}

	if len(req.GetValues()) > 32 {
		log.Error("C2SSaveOptionsHandler Values too long !")
		return -3
	}

	return 0
}

func C2SChgNameHandler(p *Player, msg_data []byte) int32 {
	var req msg_client_message.C2SChgName
	err := proto.Unmarshal(msg_data, &req)
	if err != nil {
		log.Error("unmarshal msg failed err(%s) !", err.Error())
		return -1
	}

	new_name := req.GetName()
	if len(new_name) == 0 || int32(len(new_name)) > global_config.MaxNameLen {
		log.Error("name len[%d] error !", len(req.GetName()))
		return int32(msg_client_message.E_ERR_PLAYER_RENAME_TOO_LONG_NAME)
	}

	cur_chg_count := p.db.Info.GetChangeNameCount()
	if cur_chg_count >= global_config.ChgNameCostLen {
		cur_chg_count = global_config.ChgNameCostLen - 1 //
	}

	cost_diamond := global_config.ChgNameCost[cur_chg_count]
	if p.GetDiamond() < cost_diamond {
		log.Error("C2SChgNameHandler not enough cost[%d<%d]", p.GetDiamond(), cost_diamond)
		return int32(msg_client_message.E_ERR_PLAYER_RENAME_NOT_ENOUGH_DIAMOND)
	}

	cur_chg_count = p.db.Info.IncbyChangeNameCount(1)

	p.db.SetName(new_name)

	msg := &msg_client_message.S2CChgName{}
	msg.Name = new_name
	msg.ChgNameCount = cur_chg_count
	p.Send(uint16(msg_client_message.S2CChgName_ProtoID), msg)

	p.rpc_player_base_info_update()

	return 1
}

func C2SChangeHeadHandler(p *Player, msg_data []byte) int32 {
	var req msg_client_message.C2SChangeHead
	err := proto.Unmarshal(msg_data, &req)
	if err != nil {
		log.Error("unmarshal msg failed err(%s) !", err.Error())
		return -1
	}

	if p.db.Info.GetHead() == req.GetNewHead() {
		return 0
	}

	p.db.Info.SetHead(req.GetNewHead())

	response := &msg_client_message.S2CChangeHead{}
	response.NewHead = req.GetNewHead()
	p.Send(uint16(msg_client_message.S2CChangeHead_ProtoID), response)

	p.rpc_player_base_info_update()

	return 1
}

func C2SZanPlayerHandler(p *Player, msg_data []byte) int32 {
	var req msg_client_message.C2SZanPlayer
	err := proto.Unmarshal(msg_data, &req)
	if err != nil {
		log.Error("unmarshal msg failed err(%s) !", err.Error())
		return -1
	}
	if p.Id == req.GetPlayerId() {
		log.Error("Player[%v] cant to zan self", p.Id)
		return -1
	}

	res := p.zan_player(req.GetPlayerId())
	if res < 0 {
		return res
	}

	// update rank list
	p.TaskUpdate(tables.TASK_COMPLETE_TYPE_WON_PRAISE, false, 0, 1)
	result := p.rpc_rank_list_update_data(common.RANK_LIST_TYPE_BE_ZANED, []int32{req.GetPlayerId()})

	response := &msg_client_message.S2CZanPlayerResult{
		PlayerId: req.GetPlayerId(),
		TotalZan: result.Result,
	}
	p.Send(uint16(msg_client_message.S2CZanPlayerResult_ProtoID), response)

	return 1
}

func C2SUseItemHandler(p *Player, msg_data []byte) int32 {
	var req msg_client_message.C2SUseItem
	err := proto.Unmarshal(msg_data, &req)
	if err != nil {
		log.Error("unmarshal msg failed err(%s) !", err.Error())
		return -1
	}
	return p.use_item(req.GetItemCfgId(), req.GetItemNum())
}

func C2SSellItemHandler(p *Player, msg_data []byte) int32 {
	var req msg_client_message.C2SSellItem
	err := proto.Unmarshal(msg_data, &req)
	if err != nil {
		log.Error("unmarshal msg failed err(%s) !", err.Error())
		return -1
	}
	return p.sell_item(req.GetItemId(), req.GetItemNum())
}

func C2SComposeCatHandler(p *Player, msg_data []byte) int32 {
	var req msg_client_message.C2SComposeCat
	err := proto.Unmarshal(msg_data, &req)
	if err != nil {
		log.Error("unmarshal msg failed err(%s) !", err.Error())
		return -1
	}
	return p.compose_cat(req.GetCatConfigId())
}

func C2SItemResourceHandler(p *Player, msg_data []byte) int32 {
	var req msg_client_message.C2SItemResource
	ids := req.GetResourceIds()
	response := &msg_client_message.S2CItemResourceResult{}
	response.Items = make([]*msg_client_message.S2CItemResourceValue, len(ids))
	for i, id := range ids {
		v := p.GetItemResourceValue(id)
		response.Items[i] = &msg_client_message.S2CItemResourceValue{}
		response.Items[i].ResourceId = id
		response.Items[i].ResourceValue = v
	}
	p.Send(uint16(msg_client_message.S2CItemResourceResult_ProtoID), response)

	return 1
}

func C2SShopItemsHandler(p *Player, msg_data []byte) int32 {
	var req msg_client_message.C2SShopItems
	err := proto.Unmarshal(msg_data, &req)
	if err != nil {
		log.Error("unmarshal msg failed err(%s) !", err.Error())
		return -1
	}
	if req.GetShopId() == 0 {
		var shop_type = []int32{
			tables.SHOP_TYPE_SPECIAL,
			tables.SHOP_TYPE_CHARM_MEDAL,
			tables.SHOP_TYPE_FRIEND_POINTS,
			tables.SHOP_TYPE_RMB,
			tables.SHOP_TYPE_SOUL_STONE,
		}
		for i := 0; i < len(shop_type); i++ {
			if res := p.fetch_shop_limit_items(shop_type[i], true); res < 0 {
				return res
			}
		}
		return 1
	}
	return p.fetch_shop_limit_items(req.GetShopId(), true)
}

func C2SBuyShopItemHandler(p *Player, msg_data []byte) int32 {
	var req msg_client_message.C2SBuyShopItem
	err := proto.Unmarshal(msg_data, &req)
	if err != nil {
		log.Error("unmarshal msg failed err(%s) !", err.Error())
		return -1
	}
	if p.check_shop_limited_days_items_refresh_by_shop_itemid(req.GetItemId(), true) {
		log.Info("刷新了商店")
		return 1
	}
	return p.buy_item(req.GetItemId(), req.GetItemNum(), true)
}

func C2SFeedCatHandler(p *Player, msg_data []byte) int32 {
	var req msg_client_message.C2SCatFeed
	err := proto.Unmarshal(msg_data, &req)
	if err != nil {
		log.Error("unmarshal msg failed err(%s) !", err.Error())
		return -1
	}
	need_food, add_exp, is_critical := p.feed_need_food(req.GetCatId())
	if need_food <= 0 {
		return need_food
	}
	curr_level, curr_exp, e := p.feed_cat(req.GetCatId(), need_food, add_exp, is_critical)
	if e < 0 {
		return e
	}

	response := &msg_client_message.S2CCatFeedResult{}
	response.CatId = req.GetCatId()
	response.CatLevel = curr_level
	response.CatExp = curr_exp
	response.IsCritical = is_critical
	p.Send(uint16(msg_client_message.S2CCatFeedResult_ProtoID), response)
	return 1
}

func C2SCatUpgradeStarHandler(p *Player, msg_data []byte) int32 {
	var req msg_client_message.C2SCatUpgradeStar
	err := proto.Unmarshal(msg_data, &req)
	if err != nil {
		log.Error("unmarshal msg failed err(%s) !", err.Error())
		return -1
	}
	return p.cat_upstar(req.GetCatId(), req.GetCostCatIds())
}

func C2SCatSkillLevelUpHandler(p *Player, msg_data []byte) int32 {
	var req msg_client_message.C2SCatSkillLevelUp
	err := proto.Unmarshal(msg_data, &req)
	if err != nil {
		log.Error("unmarshal msg failed err(%s) !", err.Error())
		return -1
	}
	if req.GetCostCatIds() == nil {
		log.Error("Player[%v] Cat[%v] skill level up need cost cat[%v]", p.Id, req.GetCatId(), req.GetCostCatIds())
		return -1
	}
	return p.cat_skill_levelup(req.GetCatId(), req.GetCostCatIds())
}

func C2SCatRenameNickHandler(p *Player, msg_data []byte) int32 {
	var req msg_client_message.C2SCatRenameNick
	err := proto.Unmarshal(msg_data, &req)
	if err != nil {
		log.Error("unmarshal msg failed err(%s) !", err.Error())
		return -1
	}
	return p.rename_cat(req.GetCatId(), req.GetNewNick())
}

func C2SCatLockHandler(p *Player, msg_data []byte) int32 {
	var req msg_client_message.C2SCatLock
	err := proto.Unmarshal(msg_data, &req)
	if err != nil {
		log.Error("unmarshal msg failed err(%s) !", err.Error())
		return -1
	}
	return p.lock_cat(req.GetCatId(), req.GetIsLock())
}

func C2SCatDecomposeHandler(p *Player, msg_data []byte) int32 {
	var req msg_client_message.C2SCatDecompose
	err := proto.Unmarshal(msg_data, &req)
	if err != nil {
		log.Error("unmarshal msg failed err(%s) !", err.Error())
		return -1
	}
	return p.decompose_cat(req.GetCatId())
}

func (p *Player) send_stage_info() {
	m := &msg_client_message.S2CGetStageInfos{}
	cur_max_stage_id := p.db.Info.GetCurMaxStage()
	log.Info("cur_max_stage_id %d", cur_max_stage_id)
	if 0 == cur_max_stage_id {
		m.CurMaxStage = chapter_table_mgr.InitStageId
		log.Info("m.CurMaxStage %d %d", cur_max_stage_id, chapter_table_mgr.InitStageId)
	} else {
		level_cfg := level_table_mgr.Map[cur_max_stage_id]
		if nil != level_cfg {
			m.CurMaxStage = level_cfg.NextLevel
		}
	}

	log.Info("m.CurMaxStage2 %d %d", cur_max_stage_id, chapter_table_mgr.InitStageId)
	m.CurUnlockMaxStage = p.db.Info.GetMaxUnlockStage()
	chapter_id := p.db.ChapterUnLock.GetChapterId()
	if chapter_id > 0 {
		chapter_cfg := chapter_table_mgr.Map[chapter_id]
		if nil != chapter_cfg {
			m.UnlockLeftSec = chapter_cfg.UnlockTime - (int32(time.Now().Unix()) - p.db.ChapterUnLock.GetStartUnix())
			if m.UnlockLeftSec < 0 {
				m.UnlockLeftSec = 0
			}
		}
	}

	m.CurUnlockStageId = p.db.ChapterUnLock.GetChapterId()

	p.db.Stages.FillAllMsg(m)
	p.Send(uint16(msg_client_message.S2CGetStageInfos_ProtoID), m)
}

func (this *Player) send_items() {
	m := &msg_client_message.S2CGetItemInfos{}
	this.db.Items.FillAllMsg(m)
	this.Send(uint16(msg_client_message.S2CGetItemInfos_ProtoID), m)
}

func (this *Player) send_cats() {
	res2cli := &msg_client_message.S2CGetCatInfos{}
	this.db.Cats.FillAllMsg(res2cli)
	cats := res2cli.GetCats()
	if cats != nil {
		for i := 0; i < len(cats); i++ {
			cats[i].State = this.GetCatState(cats[i].GetId())
		}
	}
	this.Send(uint16(msg_client_message.S2CGetCatInfos_ProtoID), res2cli)
}

func (this *Player) send_buildings() {
	res2cli := &msg_client_message.S2CGetBuildingInfos{}
	res2cli.Builds = this.check_and_fill_buildings_msg()
	this.Send(uint16(msg_client_message.S2CGetBuildingInfos_ProtoID), res2cli)

}

func (this *Player) send_depot_buildings() {
	m := &msg_client_message.S2CGetDepotBuildingInfos{}
	this.db.BuildingDepots.FillAllMsg(m)
	this.Send(uint16(msg_client_message.S2CGetDepotBuildingInfos_ProtoID), m)
}

func (this *Player) send_areas() {
	m := &msg_client_message.S2CGetAreasInfos{}
	this.db.Areas.FillAllMsg(m)
	this.Send(uint16(msg_client_message.S2CGetAreasInfos_ProtoID), m)
}

func (this *Player) send_data_on_login(new_player bool) {
	this.send_info()
	this.send_items()
	this.send_cats()
	this.send_buildings()
	this.send_depot_buildings()
	this.send_areas()
	this.get_crops()
	this.get_cathouses_info()
	this.send_surface_data()
	this.send_stage_info()
	this.get_formulas()
	this.pull_formula_building()
	this.send_task(0)
	this.guide_data()
	this.send_focus_data()
	this.send_my_picture_data()
	this.space_fashion_data()
	//this.seven_days_data()
	this.get_sign_data()
	this.charge_data()
}

func C2SPlayerInfoHandler(p *Player, msg_data []byte) int32 {
	var req msg_client_message.C2SPlayerInfoRequest
	err := proto.Unmarshal(msg_data, &req)
	if err != nil {
		log.Error("unmarshal msg failed err(%s) !", err.Error())
		return -1
	}
	p.send_info()
	return 1
}

func C2SGetInfoHandler(p *Player, msg_data []byte) int32 {
	var req msg_client_message.C2SGetInfo
	err := proto.Unmarshal(msg_data, &req)
	if err != nil {
		log.Error("unmarshal msg failed err(%s) !", err.Error())
		return -1
	}

	if req.GetBase() {
		p.send_info()
	}

	if req.GetItem() {
		p.send_items()
	}

	if req.GetCat() {
		p.send_cats()
	}

	if req.GetBuilding() {
		p.send_buildings()
	}

	if req.GetArea() {
		p.send_areas()
	}

	if req.GetStage() {
		p.send_stage_info()
	}

	if req.GetFormula() {
		p.get_formulas()
	}

	if req.GetDepotBuilding() {
		p.send_depot_buildings()
	}

	if req.GetGuide() {
		p.guide_data()
	}

	if req.GetCatHouse() {
		p.get_cathouses_info()
	}

	if req.GetWorkShop() {
		p.pull_formula_building()
	}

	if req.GetFarm() {
		p.get_crops()
	}

	log.Trace("Player %v get info %v", p.Id, req)

	return 1
}

func C2SGetMakingFormulaBuildingsHandler(p *Player, msg_data []byte) int32 {
	var req msg_client_message.C2SGetMakingFormulaBuildings
	err := proto.Unmarshal(msg_data, &req)
	if err != nil {
		log.Error("unmarshal msg failed err(%s) !", err.Error())
		return -1
	}
	return p.pull_formula_building()
}

func C2SExchangeBuildingFormulaHandler(p *Player, msg_data []byte) int32 {
	var req msg_client_message.C2SExchangeBuildingFormula
	err := proto.Unmarshal(msg_data, &req)
	if err != nil {
		log.Error("unmarshal msg failed err(%s) !", err.Error())
		return -1
	}
	return p.exchange_formula(req.GetFormulaId())
}

func C2SMakeFormulaBuildingHandler(p *Player, msg_data []byte) int32 {
	var req msg_client_message.C2SMakeFormulaBuilding
	err := proto.Unmarshal(msg_data, &req)
	if err != nil {
		log.Error("unmarshal msg failed err(%s) !", err.Error())
		return -1
	}
	return p.make_formula_building(req.GetFormulaId() /*, req.GetSlotId()*/)
}

func C2SBuyMakeBuildingSlotHandler(p *Player, msg_data []byte) int32 {
	var req msg_client_message.C2SBuyMakeBuildingSlot
	err := proto.Unmarshal(msg_data, &req)
	if err != nil {
		log.Error("unmarshal msg failed err(%s) !", err.Error())
		return -1
	}
	return p.buy_new_making_building_slot()
}

func C2SSpeedupMakeBuildingHandler(p *Player, msg_data []byte) int32 {
	var req msg_client_message.C2SSpeedupMakeBuilding
	err := proto.Unmarshal(msg_data, &req)
	if err != nil {
		log.Error("unmarshal msg failed err(%s) !", err.Error())
		return -1
	}
	return p.speedup_making_building( /*req.GetSlotId()*/ )
}

func C2SGetCompletedFormulaBuildingHandler(p *Player, msg_data []byte) int32 {
	var req msg_client_message.C2SGetCompletedFormulaBuilding
	err := proto.Unmarshal(msg_data, &req)
	if err != nil {
		log.Error("unmarshal msg failed err(%s) !", err.Error())
		return -1
	}
	return p.get_completed_formula_building( /*req.GetSlotId()*/ )
}

func C2SCancelMakingFormulaBuildingHandler(p *Player, msg_data []byte) int32 {
	var req msg_client_message.C2SCancelMakingFormulaBuilding
	err := proto.Unmarshal(msg_data, &req)
	if err != nil {
		log.Error("unmarshal msg failed err(%s) !", err.Error())
		return -1
	}
	return p.cancel_making_formula_building(req.GetSlotId())
}

func C2SGetFormulasHandler(p *Player, msg_data []byte) int32 {
	var req msg_client_message.C2SGetFormulas
	err := proto.Unmarshal(msg_data, &req)
	if err != nil {
		log.Error("unmarshal msg failed err(%s) !", err.Error())
		return -1
	}
	return p.get_formulas()
}

func C2SGetCropsHandler(p *Player, msg_data []byte) int32 {
	var req msg_client_message.C2SGetCrops
	err := proto.Unmarshal(msg_data, &req)
	if err != nil {
		log.Error("unmarshal msg failed err(%s) !", err.Error())
		return -1
	}
	return p.get_crops()
}

func C2SPlantCropHandler(p *Player, msg_data []byte) int32 {
	var req msg_client_message.C2SPlantCrop
	err := proto.Unmarshal(msg_data, &req)
	if err != nil {
		log.Error("unmarshal msg failed err(%s) !", err.Error())
		return -1
	}
	return p.plant_crop(req.GetCropId(), req.GetDestBuildingId())
}

func C2SSpeedupCropHandler(p *Player, msg_data []byte) int32 {
	var req msg_client_message.C2SCropSpeedup
	err := proto.Unmarshal(msg_data, &req)
	if err != nil {
		log.Error("unmarshal msg failed err(%s) !", err.Error())
		return -1
	}
	return p.speedup_crop(req.GetFarmBuildingId())
}

func C2SHarvestCropHandler(p *Player, msg_data []byte) int32 {
	var req msg_client_message.C2SHarvestCrop
	err := proto.Unmarshal(msg_data, &req)
	if err != nil {
		log.Error("unmarshal msg failed err(%s) !", err.Error())
		return -1
	}
	return p.harvest_crop(req.GetFarmBuildingId(), req.GetIsSpeedup())
}

func C2SHarvestCropsHandler(p *Player, msg_data []byte) int32 {
	var req msg_client_message.C2SHarvestCrops
	err := proto.Unmarshal(msg_data, &req)
	if err != nil {
		log.Error("unmarshal msg failed err(%s) !", err.Error())
		return -1
	}
	return p.harvest_crops(req.GetBuildingIds())
}

func C2SGetCatHousesInfoHandler(p *Player, msg_data []byte) int32 {
	var req msg_client_message.C2SGetCatHousesInfo
	err := proto.Unmarshal(msg_data, &req)
	if err != nil {
		log.Error("unmarshal msg failed err(%s) !", err.Error())
		return -1
	}
	return p.get_cathouses_info()
}

func C2SCatHouseAddCatHandler(p *Player, msg_data []byte) int32 {
	var req msg_client_message.C2SCatHouseAddCat
	err := proto.Unmarshal(msg_data, &req)
	if err != nil {
		log.Error("unmarshal msg failed err(%s) !", err.Error())
		return -1
	}
	return p.cathouse_add_cat(req.GetCatId(), req.GetCatHouseId())
}

func C2SCatHouseRemoveCatHandler(p *Player, msg_data []byte) int32 {
	var req msg_client_message.C2SCatHouseRemoveCat
	err := proto.Unmarshal(msg_data, &req)
	if err != nil {
		log.Error("unmarshal msg failed err(%s) !", err.Error())
		return -1
	}
	return p.cathouse_remove_cat(req.GetCatId(), req.GetCatHouseId())
}

func C2SCatHouseStartLevelupHandler(p *Player, msg_data []byte) int32 {
	var req msg_client_message.C2SCatHouseStartLevelup
	err := proto.Unmarshal(msg_data, &req)
	if err != nil {
		log.Error("unmarshal msg failed err(%s) !", err.Error())
		return -1
	}
	return p.cathouse_start_levelup(req.GetCatHouseId(), true)
}

func C2SCatHouseSpeedLevelupHandler(p *Player, msg_data []byte) int32 {
	var req msg_client_message.C2SCatHouseSpeedLevelup
	err := proto.Unmarshal(msg_data, &req)
	if err != nil {
		log.Error("unmarshal msg failed err(%s) !", err.Error())
		return -1
	}
	return p.cathouse_speed_levelup(req.GetCatHouseId())
}

func C2SCatHouseSellHandler(p *Player, msg_data []byte) int32 {
	var req msg_client_message.C2SSellCatHouse
	err := proto.Unmarshal(msg_data, &req)
	if err != nil {
		log.Error("unmarshal msg failed err(%s) !", err.Error())
		return -1
	}
	return p.cathouse_remove(req.GetCatHouseId(), true)
}

func C2SCatHouseProduceGoldHandler(p *Player, msg_data []byte) int32 {
	var req msg_client_message.C2SCatHouseProduceGold
	err := proto.Unmarshal(msg_data, &req)
	if err != nil {
		log.Error("unmarshal msg failed err(%s) !", err.Error())
		return -1
	}
	return p.cathouse_produce_gold(req.GetCatHouseId())
}

func C2SCatHouseGetGoldHandler(p *Player, msg_data []byte) int32 {
	var req msg_client_message.C2SCatHouseGetGold
	err := proto.Unmarshal(msg_data, &req)
	if err != nil {
		log.Error("unmarshal msg failed err(%s) !", err.Error())
		return -1
	}
	res := p.cathouse_collect_gold(req.GetCatHouseId())
	if res < 0 {
		return res
	}
	return 1
}

func C2SCatHousesGetGoldHandler(p *Player, msg_data []byte) int32 {
	var req msg_client_message.C2SCatHousesGetGold
	err := proto.Unmarshal(msg_data, &req)
	if err != nil {
		log.Error("unmarshal msg failed err(%s) !", err.Error())
		return -1
	}
	if req.GetCatHouseIds() == nil {
		log.Error("!!! Cat houses is empty")
		return -1
	}

	return p.cathouses_collect_gold(req.GetCatHouseIds())
}

func C2SCatHouseSetDoneHandler(p *Player, msg_data []byte) int32 {
	var req msg_client_message.C2SCatHouseSetDone
	err := proto.Unmarshal(msg_data, &req)
	if err != nil {
		log.Error("unmarshal msg failed err(%s) !", err.Error())
		return -1
	}
	return p.cathouse_setdone(req.GetCatHouseId())
}

func C2SGetHandbookHandler(p *Player, msg_data []byte) int32 {
	var req msg_client_message.C2SGetHandbook
	err := proto.Unmarshal(msg_data, &req)
	if err != nil {
		log.Error("unmarshal msg failed err(%s) !", err.Error())
		return -1
	}

	response := &msg_client_message.S2CGetHandbookResult{}
	all_ids := p.db.HandbookItems.GetAllIndex()
	if all_ids == nil || len(all_ids) == 0 {
		response.Items = make([]int32, 0)
	} else {
		n := 0
		response.Items = make([]int32, len(all_ids))
		for i := 0; i < len(all_ids); i++ {
			handbook := handbook_table_mgr.Get(all_ids[i])
			if handbook == nil {
				log.Warn("Player[%v] load handbook[%v] not found", p.Id, all_ids[i])
				continue
			}
			response.Items[n] = all_ids[i]
			n += 1
		}
		response.Items = response.Items[:n]
	}
	suit_ids := p.db.SuitAwards.GetAllIndex()
	if suit_ids == nil || len(suit_ids) == 0 {
		response.AwardSuitId = make([]int32, 0)
	} else {
		response.AwardSuitId = make([]int32, len(suit_ids))
		for i := 0; i < len(suit_ids); i++ {
			response.AwardSuitId[i] = suit_ids[i]
		}
	}
	p.Send(uint16(msg_client_message.S2CGetHandbookResult_ProtoID), response)
	return 1
}

func C2SGetHeadHandler(p *Player, msg_data []byte) int32 {
	var req msg_client_message.C2SGetHead
	err := proto.Unmarshal(msg_data, &req)
	if err != nil {
		log.Error("unmarshal msg failed err(%s) !", err.Error())
		return -1
	}

	response := &msg_client_message.S2CGetHeadResult{}
	all_ids := p.db.HeadItems.GetAllIndex()
	if all_ids == nil || len(all_ids) == 0 {
		response.Items = make([]int32, 0)
	} else {
		response.Items = make([]int32, len(all_ids))
		for i := 0; i < len(all_ids); i++ {
			response.Items[i] = all_ids[i]
		}
	}
	p.Send(uint16(msg_client_message.S2CGetHeadResult_ProtoID), response)
	return 1
}

func C2SGetSuitHandbookRewardHandler(p *Player, msg_data []byte) int32 {
	var req msg_client_message.C2SGetSuitHandbookReward
	err := proto.Unmarshal(msg_data, &req)
	if err != nil {
		log.Error("unmarshal msg failed err(%s) !", err.Error())
		return -1
	}
	suit := suit_table_mgr.Map[req.GetSuitId()]
	suit_buildings := building_table_mgr.Suits[suit.Id]

	if suit == nil || suit_buildings == nil {
		log.Error("Player[%v] suit_id[%v] is invalid", p.Id, req.GetSuitId())
		return -1
	}

	if p.db.SuitAwards.HasIndex(suit.Id) {
		log.Error("Player[%v] already award suit[%v] reward", p.Id, suit.Id)
		return -1
	}

	b := true
	for _, v := range suit_buildings.Items {
		if !p.db.HandbookItems.HasIndex(v) {
			b = false
			break
		}
	}
	if !b {
		log.Error("Player[%v] suit[%v] not collect all", p.Id, suit.Id)
		return -1
	}

	d := &dbPlayerSuitAwardData{
		Id:        suit.Id,
		AwardTime: int32(time.Now().Unix()),
	}
	p.db.SuitAwards.Add(d)

	response := &msg_client_message.S2CGetSuitHandbookRewardResult{}
	l := int32(len(suit.Rewards) / 2)
	response.Rewards = make([]*msg_client_message.ItemInfo, l)
	for i := int32(0); i < l; i++ {
		p.AddItemResource(suit.Rewards[2*i], suit.Rewards[2*i+1], "suit_award", "handbook")
		response.Rewards[i] = &msg_client_message.ItemInfo{
			ItemCfgId: suit.Rewards[2*i],
			ItemNum:   suit.Rewards[2*i+1],
		}
	}
	p.Send(uint16(msg_client_message.S2CGetSuitHandbookRewardResult_ProtoID), response)

	p.SendItemsUpdate()
	p.SendCatsUpdate()
	p.SendBuildingUpdate()

	return 1
}

func (this *Player) rank_list_get_data(rank_type, rank_start, rank_num int32, param []int32) int32 {
	var res int32 = 0
	if rank_start <= 0 {
		log.Warn("Player[%v] get rank list by type[%v] with rank_start[%v] invalid", this.Id, rank_type, rank_start)
		return -1
	}
	if rank_num <= 0 {
		log.Warn("Player[%v] get rank list by type[%v] with rank_num[%v] invalid", this.Id, rank_type, rank_num)
		return -1
	}

	if rank_num > global_config.RankingListOnceGetItemsNum {
		return int32(msg_client_message.E_ERR_RANK_GET_ITEMS_NUM_OVER_MAX)
	}

	var rank_param int32
	if rank_type == common.RANK_LIST_TYPE_CAT_OUQI {
		if param == nil || len(param) == 0 {
			return -1
		}
		rank_param = param[0]
	}

	data := this.rpc_rank_list_get_data(rank_type, rank_start, rank_num, rank_param)
	if data == nil {
		return -1
	}

	var rank_items []*msg_client_message.RankItemInfo
	var self_value, self_value2 int32
	if rank_type == common.RANK_LIST_TYPE_CAT_OUQI {
		if data.RankItems != nil {
			for i, r := range data.RankItems {
				rr := r.(*common.PlayerCatOuqiRankItem)
				if rr == nil {
					continue
				}
				rank_items = append(rank_items, &msg_client_message.RankItemInfo{
					Rank:        data.StartRank + int32(i),
					PlayerId:    rr.PlayerId,
					PlayerValue: []int32{rr.CatId, rr.Ouqi},
				})
			}
		}
		if data.SelfValue != nil {
			self_value = rank_param
			self_value2 = data.SelfValue.(int32)
		}
	} else {
		if data.RankItems != nil {
			for i, r := range data.RankItems {
				rr := r.(*common.PlayerInt32RankItem)
				if rr == nil {
					continue
				}
				var is_zaned int32
				if rank_type == common.RANK_LIST_TYPE_BE_ZANED {
					if this.db.Zans.HasIndex(rr.PlayerId) {
						is_zaned = 1
					}
				}
				rank_items = append(rank_items, &msg_client_message.RankItemInfo{
					Rank:     data.StartRank + int32(i),
					PlayerId: rr.PlayerId,
					PlayerValue: func() []int32 {
						if rank_type == common.RANK_LIST_TYPE_BE_ZANED {
							return []int32{rr.Value, is_zaned}
						} else {
							return []int32{rr.Value}
						}
					}(),
				})
			}
		}
		if data.SelfValue != nil {
			self_value = data.SelfValue.(int32)
		}
	}

	for _, item := range rank_items {
		pb := data.PlayerBaseInfos[item.GetPlayerId()]
		if pb == nil {
			continue
		}
		item.PlayerName = pb.Name
		item.PlayerLevel = pb.Level
		item.PlayerHead = pb.Head
	}

	response := &msg_client_message.S2CRankListResponse{
		RankListType:       rank_type,
		RankItems:          rank_items,
		SelfHistoryTopRank: data.SelfHistoryTopRank,
		SelfRank:           data.SelfRank,
		SelfValue:          self_value,
		SelfValue2:         self_value2,
	}
	this.Send(uint16(msg_client_message.S2CRankListResponse_ProtoID), response)

	log.Trace("Player %v get rank list data %v", this.Id, response)

	return res
}

func C2SRankingListHandler(p *Player, msg_data []byte) int32 {
	var req msg_client_message.C2SRankListRequest
	err := proto.Unmarshal(msg_data, &req)
	if err != nil {
		log.Error("unmarshal msg failed err(%s) !", err.Error())
		return -1
	}
	return p.rank_list_get_data(req.GetRankType(), req.GetStartRank(), req.GetRankNum(), req.GetParams())
}

func C2SPlayerCatInfoHandler(p *Player, msg_data []byte) int32 {
	var req msg_client_message.C2SPlayerCatInfo
	err := proto.Unmarshal(msg_data, &req)
	if err != nil {
		log.Error("unmarshal msg failed err(%s) !", err.Error())
		return -1
	}
	return p.get_player_cat_info(req.GetPlayerId(), req.GetCatId())
}

func C2SWorldChatMsgPullHandler(p *Player, msg_data []byte) int32 {
	var req msg_client_message.C2SChatMsgPullRequest
	err := proto.Unmarshal(msg_data, &req)
	if err != nil {
		log.Error("unmarshal msg failed err(%s) !", err.Error())
		return -1
	}
	return p.pull_chat(CHAT_CHANNEL_WORLD)
}

func C2SWorldChatSendHandler(p *Player, msg_data []byte) int32 {
	var req msg_client_message.C2SChatRequest
	err := proto.Unmarshal(msg_data, &req)
	if err != nil {
		log.Error("unmarshal msg failed err(%s) !", err.Error())
		return -1
	}
	return p.chat(CHAT_CHANNEL_WORLD, req.GetContent(), 0)
}
