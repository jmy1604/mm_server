package main

import (
	"mm_server/libs/log"
	"mm_server/proto/gen_go/client_message"
	"mm_server/proto/gen_go/server_message"

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
	//cur_area_use_count      map[int32]int32
	cur_areablocknum_map map[int32]int32

	b_base_prop_chg bool

	item_cat_building_change_info ItemCatBuildingChangeInfo // 物品猫建筑数量状态变化

	new_unlock_chapter_id int32

	used_drop_ids map[int32]int32 // 抽卡掉落ID统计
	//world_chat_data  PlayerWorldChatData   // 世界聊天缓存数据
	world_chat_data  PlayerChatData        // 世界聊天缓存数据
	anouncement_data PlayerAnouncementData // 公告缓存数据

	stage_id     int32
	stage_cat_id int32
	stage_state  int32

	is_paying int32 // 是否正在支付

	new_mail_list_locker sync.RWMutex // 新邮件列表锁
	new_mail_ids         []int32      // 新邮件ID列表
	receive_mail_locker  sync.RWMutex // 接受邮件锁
}

func new_player(id int32, uid, account, token string, db *dbPlayerRow) *Player {
	ret_p := &Player{}
	ret_p.UniqueId = uid
	ret_p.Id = id
	ret_p.Account = account
	//ret_p.Token = token
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

	//log.Info("add_msg_data %d, %v at %d", msg_code, data, this.cur_msg_items_len)

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

func (this *Player) SendBaseInfo() {
	var msg msg_client_message.S2CRetBaseInfo
	msg.Nick = this.db.GetName()
	msg.Coins = this.db.Info.GetGold()
	msg.Diamonds = this.db.Info.GetDiamond()
	msg.Lvl = this.db.Info.GetLvl()
	msg.Exp = this.db.Info.GetExp()
	msg.Head = this.db.Info.GetHead()
	msg.CurMaxStage = this.db.Info.GetCurMaxStage()
	msg.CurUnlockMaxStage = this.db.Info.GetMaxUnlockStage()
	msg.CharmVal = this.db.Info.GetCharmVal()
	msg.CatFood = this.db.Info.GetCatFood()
	msg.Zan = this.db.Info.GetZan()
	msg.FriendPoints = this.db.Info.GetFriendPoints()
	msg.SoulStone = this.db.Info.GetSoulStone()
	msg.Star = this.db.Info.GetTotalStars()
	msg.Spirit = this.CalcSpirit()
	msg.CharmMetal = this.db.Info.GetCharmMedal()
	//msg.HistoricalMaxStar = this.db.Stages.GetTotalTopStar()
	msg.ChangeNameNum = this.db.Info.GetChangeNameCount()
	msg.ChangeNameCostDiamond = global_config.ChangeNameCostDiamond
	msg.ChangeNameFreeNum = global_config.ChangeNameFreeNum

	this.Send(uint16(msg_client_message.S2CRetBaseInfo_ProtoID), &msg)
}

func (this *Player) PopCurMsgData() []byte {
	if this.b_base_prop_chg {
		this.SendBaseInfo()
	}

	if this.ChkMapBlock() > 0 {
		this.item_cat_building_change_info.send_buildings_update(this)
	}

	if this.ChkMapChest() > 0 {
		this.item_cat_building_change_info.send_buildings_update(this)
	}

	this.ChkSendNewUnlockStage()

	this.CheckAndAnouncement()

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
	tmp_acc := this.Account
	if len(tmp_acc) > 6 {
		tmp_acc = string([]byte(tmp_acc)[0:6])
	}

	//this.db.SetName(fmt.Sprintf("MM_%s_%d", tmp_acc, this.Id))
	this.db.Info.SetLvl(1)
	this.db.Info.SetCreateUnix(int32(time.Now().Unix()))
	// 新任务
	//this.UpdateNewTasks(1, false)

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

	/*res2co := &msg_server_message.SetPlayerOnOffline{}
	res2co.PlayerId = this.Id
	res2co.OnOffLine = 1
	center_conn.Send(res2co)*/

	/*result := this.rpc_call_update_base_info()
	if result.Error < 0 {
		log.Warn("rpc update player[%v] base info error[%v]", result.Error)
	}*/

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

	/*res2co := &msg_server_message.SetPlayerOnOffline{}
	res2co.PlayerId = this.Id
	res2co.OnOffLine = 1
	center_conn.Send(res2co)*/

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
	response := &msg_client_message.S2CPlayerInfoResponse{
		Level:    this.db.Info.GetLvl(),
		Exp:      this.db.Info.GetExp(),
		Gold:     this.db.Info.GetGold(),
		Diamond:  this.db.Info.GetDiamond(),
		Icon:     this.db.Info.GetHead(),
		VipLevel: this.db.Info.GetVipLvl(),
		Name:     this.db.GetName(),
		SysTime:  int32(time.Now().Unix()),
	}
	this.Send(uint16(msg_client_message.S2CPlayerInfoResponse_ProtoID), response)
	log.Trace("Player[%v] info: %v", this.Id, response)
}

func (this *Player) notify_enter_complete() {
	msg := &msg_client_message.S2CEnterGameCompleteNotify{}
	this.Send(uint16(msg_client_message.S2CEnterGameCompleteNotify_ProtoID), msg)
}

func (this *Player) change_head(new_head int32) int32 {
	head := item_table_mgr.Get(new_head)
	if head == nil {
		log.Error("head[%v] table data not found", new_head)
		return int32(msg_client_message.E_ERR_PLAYER_HEAD_TABLE_DATA_NOT_FOUND)
	}

	if head.Type != ITEM_TYPE_HEAD {
		log.Error("item[%v] type is not head", new_head)
		return -1
	}

	if this.get_resource(new_head) < 1 {
		log.Error("Player[%v] no head %v", this.Id, new_head)
		return int32(msg_client_message.E_ERR_PLAYER_NO_SUCH_HEAD)
	}

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

func reg_player_base_info_msg() {
	msg_handler_mgr.SetPlayerMsgHandler(uint16(msg_client_message.C2S_TEST_COMMAND_ProtoID), C2STestCommandHandler)

	msg_handler_mgr.SetPlayerMsgHandler(uint16(msg_client_message.C2SGetInfo_ProtoID), C2SGetInfoHandler)
	msg_handler_mgr.SetPlayerMsgHandler(uint16(msg_client_message.C2SGetBaseInfo_ProtoID), C2SGetBaseInfoHandler)
	msg_handler_mgr.SetPlayerMsgHandler(uint16(msg_client_message.C2SGetItemInfos_ProtoID), C2SGetItemInfosHandler)
	msg_handler_mgr.SetPlayerMsgHandler(uint16(msg_client_message.C2SGetDepotBuildingInfos_ProtoID), C2SGetDepotBuildingInfosHandler)
	msg_handler_mgr.SetPlayerMsgHandler(uint16(msg_client_message.C2SGetCatInfos_ProtoID), C2SGetCatInfosHandler)
	msg_handler_mgr.SetPlayerMsgHandler(uint16(msg_client_message.C2SGetStageInfos_ProtoID), C2SGetStageInfosHandler)
	msg_handler_mgr.SetPlayerMsgHandler(uint16(msg_client_message.C2SGetOptions_ProtoID), C2SGetOptionsHandler)
	msg_handler_mgr.SetPlayerMsgHandler(uint16(msg_client_message.C2SSaveOptions_ProtoID), C2SSaveOptionsHandler)
	msg_handler_mgr.SetPlayerMsgHandler(uint16(msg_client_message.C2SChgName_ProtoID), C2SChgNameHandler)
	msg_handler_mgr.SetPlayerMsgHandler(uint16(msg_client_message.C2SChangeHead_ProtoID), C2SChangeHeadHandler)
	msg_handler_mgr.SetPlayerMsgHandler(uint16(msg_client_message.C2SZanPlayer_ProtoID), C2SZanPlayerHandler)

	// 物品
	msg_handler_mgr.SetPlayerMsgHandler(uint16(msg_client_message.C2SUseItem_ProtoID), C2SUseItemHandler)
	msg_handler_mgr.SetPlayerMsgHandler(uint16(msg_client_message.C2SSellItem_ProtoID), C2SSellItemHandler)
	msg_handler_mgr.SetPlayerMsgHandler(uint16(msg_client_message.C2SComposeCat_ProtoID), C2SComposeCatHandler)
	msg_handler_mgr.SetPlayerMsgHandler(uint16(msg_client_message.C2SItemResource_ProtoID), C2SItemResourceHandler)

	// 商店
	msg_handler_mgr.SetPlayerMsgHandler(uint16(msg_client_message.C2SShopItems_ProtoID), C2SShopItemsHandler)
	msg_handler_mgr.SetPlayerMsgHandler(uint16(msg_client_message.C2SBuyShopItem_ProtoID), C2SBuyShopItemHandler)

	// 猫
	msg_handler_mgr.SetPlayerMsgHandler(uint16(msg_client_message.C2SCatFeed_ProtoID), C2SFeedCatHandler)
	msg_handler_mgr.SetPlayerMsgHandler(uint16(msg_client_message.C2SCatUpgradeStar_ProtoID), C2SCatUpgradeStarHandler)
	msg_handler_mgr.SetPlayerMsgHandler(uint16(msg_client_message.C2SCatSkillLevelUp_ProtoID), C2SCatSkillLevelUpHandler)
	msg_handler_mgr.SetPlayerMsgHandler(uint16(msg_client_message.C2SCatRenameNick_ProtoID), C2SCatRenameNickHandler)
	msg_handler_mgr.SetPlayerMsgHandler(uint16(msg_client_message.C2SCatLock_ProtoID), C2SCatLockHandler)
	msg_handler_mgr.SetPlayerMsgHandler(uint16(msg_client_message.C2SCatDecompose_ProtoID), C2SCatDecomposeHandler)
	msg_handler_mgr.SetPlayerMsgHandler(uint16(msg_client_message.C2SPlayerCatInfo_ProtoID), C2SPlayerCatInfoHandler)

	// 配方
	msg_handler_mgr.SetPlayerMsgHandler(uint16(msg_client_message.C2SGetMakingFormulaBuildings_ProtoID), C2SGetMakingFormulaBuildingsHandler)
	msg_handler_mgr.SetPlayerMsgHandler(uint16(msg_client_message.C2SExchangeBuildingFormula_ProtoID), C2SExchangeBuildingFormulaHandler)
	msg_handler_mgr.SetPlayerMsgHandler(uint16(msg_client_message.C2SMakeFormulaBuilding_ProtoID), C2SMakeFormulaBuildingHandler)
	msg_handler_mgr.SetPlayerMsgHandler(uint16(msg_client_message.C2SBuyMakeBuildingSlot_ProtoID), C2SBuyMakeBuildingSlotHandler)
	msg_handler_mgr.SetPlayerMsgHandler(uint16(msg_client_message.C2SSpeedupMakeBuilding_ProtoID), C2SSpeedupMakeBuildingHandler)
	msg_handler_mgr.SetPlayerMsgHandler(uint16(msg_client_message.C2SGetCompletedFormulaBuilding_ProtoID), C2SGetCompletedFormulaBuildingHandler)
	msg_handler_mgr.SetPlayerMsgHandler(uint16(msg_client_message.C2SCancelMakingFormulaBuilding_ProtoID), C2SCancelMakingFormulaBuildingHandler)
	msg_handler_mgr.SetPlayerMsgHandler(uint16(msg_client_message.C2SGetFormulas_ProtoID), C2SGetFormulasHandler)

	// 农田
	msg_handler_mgr.SetPlayerMsgHandler(uint16(msg_client_message.C2SGetCrops_ProtoID), C2SGetCropsHandler)
	msg_handler_mgr.SetPlayerMsgHandler(uint16(msg_client_message.C2SPlantCrop_ProtoID), C2SPlantCropHandler)
	msg_handler_mgr.SetPlayerMsgHandler(uint16(msg_client_message.C2SHarvestCrops_ProtoID), C2SHarvestCropHandler)
	msg_handler_mgr.SetPlayerMsgHandler(uint16(msg_client_message.C2SCropSpeedup_ProtoID), C2SSpeedupCropHandler)
	msg_handler_mgr.SetPlayerMsgHandler(uint16(msg_client_message.C2SHarvestCrops_ProtoID), C2SHarvestCropsHandler)

	// 猫舍
	msg_handler_mgr.SetPlayerMsgHandler(uint16(msg_client_message.C2SGetCatHouseInfo_ProtoID), C2SGetCatHousesInfoHandler)
	msg_handler_mgr.SetPlayerMsgHandler(uint16(msg_client_message.C2SCatHouseAddCat_ProtoID), C2SCatHouseAddCatHandler)
	msg_handler_mgr.SetPlayerMsgHandler(uint16(msg_client_message.C2SCatHouseRemoveCat_ProtoID), C2SCatHouseRemoveCatHandler)
	msg_handler_mgr.SetPlayerMsgHandler(uint16(msg_client_message.C2SCatHouseStartLevelup_ProtoID), C2SCatHouseStartLevelupHandler)
	msg_handler_mgr.SetPlayerMsgHandler(uint16(msg_client_message.C2SCatHouseSpeedLevelup_ProtoID), C2SCatHouseSpeedLevelupHandler)
	msg_handler_mgr.SetPlayerMsgHandler(uint16(msg_client_message.C2SSellCatHouse_ProtoID), C2SCatHouseSellHandler)
	msg_handler_mgr.SetPlayerMsgHandler(uint16(msg_client_message.C2SCatHouseGetGold_ProtoID), C2SCatHouseGetGoldHandler)
	msg_handler_mgr.SetPlayerMsgHandler(uint16(msg_client_message.C2SCatHouseSetDone_ProtoID), C2SCatHouseSetDoneHandler)

	// 任务
	//msg_handler_mgr.SetPlayerMsgHandler(uint16(msg_client_message.C2SGetDialyTaskInfo_ProtoID), C2SGetDialyTaskInfoHanlder)
	//msg_handler_mgr.SetPlayerMsgHandler(uint16(msg_client_message.C2SGetAchieve_ProtoID), C2SGetAchieveHandler)
	//msg_handler_mgr.SetPlayerMsgHandler(uint16(msg_client_message.C2SGetTaskReward_ProtoID), C2SGetTaskRewardHandler)
	//msg_handler_mgr.SetPlayerMsgHandler(uint16(msg_client_message.C2SGetAchieveReward_ProtoID), C2SGetAchieveRewardHandler)

	// 图鉴
	msg_handler_mgr.SetPlayerMsgHandler(uint16(msg_client_message.C2SGetHandbook_ProtoID), C2SGetHandbookHandler)
	msg_handler_mgr.SetPlayerMsgHandler(uint16(msg_client_message.C2SGetHead_ProtoID), C2SGetHeadHandler)
	msg_handler_mgr.SetPlayerMsgHandler(uint16(msg_client_message.C2SGetSuitHandbookReward_ProtoID), C2SGetSuitHandbookRewardHandler)

	// 排行榜
	msg_handler_mgr.SetPlayerMsgHandler(uint16(msg_client_message.C2SRankListRequest_ProtoID), C2SPullRankingListHandler)

	// 世界聊天
	msg_handler_mgr.SetPlayerMsgHandler(uint16(msg_client_message.C2SChatMsgPullRequest_ProtoID), C2SWorldChatMsgPullHandler)
	msg_handler_mgr.SetPlayerMsgHandler(uint16(msg_client_message.C2SChatRequest_ProtoID), C2SWorldChatSendHandler)

	// 心跳
	msg_handler_mgr.SetPlayerMsgHandler(uint16(msg_client_message.C2SHeartbeat_ProtoID), C2SHeartbeatHandler)
}

func C2SGetBaseInfoHandler(p *Player, msg_data []byte) int32 {
	var req msg_client_message.C2SGetBaseInfo
	err := proto.Unmarshal(msg_data, &req)
	if err != nil {
		log.Error("unmarshal msg failed err(%s) !", err.Error())
		return -1
	}

	p.SendBaseInfo()

	return 1
}

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

	// rpc update base info
	/*result := p.rpc_call_update_base_info()
	if result.Error < 0 {
		log.Warn("Player[%v] update base info error[%v]", p.Id, result.Error)
	}*/

	msg := &msg_client_message.S2CChgName{}
	msg.Name = new_name
	msg.ChgNameCount = cur_chg_count
	p.Send(uint16(msg_client_message.S2CChgName_ProtoID), msg)

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

	// rpc update base info
	/*result := p.rpc_call_update_base_info()
	if result.Error < 0 {
		log.Warn("Player[%v] update base info error[%v]", p.Id, result.Error)
	}*/

	response := &msg_client_message.S2CChangeHead{}
	response.NewHead = req.GetNewHead()
	p.Send(uint16(msg_client_message.S2CChangeHead_ProtoID), response)

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

	zan := int32(0)
	to_player := player_mgr.GetPlayerById(req.GetPlayerId())
	if to_player != nil {
		zan = to_player.db.Info.IncbyZan(1)
	} else {
		/*result := p.rpc_call_zan_player2(req.GetPlayerId())
		if result == nil {
			return -1
		}
		zan = result.ToPlayerZanNum*/
	}

	// update rank list
	if zan > 0 {
		/*if p.rpc_call_rank_update_zaned(req.GetPlayerId(), zan) == nil {
			log.Warn("Player[%v] remote update zan rank list failed", p.Id)
		}*/
		p.TaskUpdate(tables.TASK_COMPLETE_TYPE_WON_PRAISE, false, 0, 1)
	}

	response := &msg_client_message.S2CZanPlayerResult{
		PlayerId: req.GetPlayerId(),
		TotalZan: zan,
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

func C2SGetInfoHandler(p *Player, msg_data []byte) int32 {
	var req msg_client_message.C2SGetInfo
	err := proto.Unmarshal(msg_data, &req)
	if err != nil {
		log.Error("unmarshal msg failed err(%s) !", err.Error())
		return -1
	}

	if req.GetBase() {
		p.SendBaseInfo()
	}

	if req.GetItem() {
		m := &msg_client_message.S2CGetItemInfos{}
		p.db.Items.FillAllMsg(m)
		p.Send(uint16(msg_client_message.S2CGetItemInfos_ProtoID), m)
	}

	if req.GetCat() {
		res2cli := &msg_client_message.S2CGetCatInfos{}
		p.db.Cats.FillAllMsg(res2cli)
		cats := res2cli.GetCats()
		if cats != nil {
			for i := 0; i < len(cats); i++ {
				cats[i].State = p.GetCatState(cats[i].GetId())
			}
		}
		p.Send(uint16(msg_client_message.S2CGetCatInfos_ProtoID), res2cli)
	}

	if req.GetBuilding() {
		res2cli := &msg_client_message.S2CGetBuildingInfos{}
		//p.db.Buildings.FillAllMsg(res2cli)
		res2cli.Builds = p.check_and_fill_buildings_msg()
		p.Send(uint16(msg_client_message.S2CGetBuildingInfos_ProtoID), res2cli)
	}

	if req.GetArea() {
		m := &msg_client_message.S2CGetAreasInfos{}
		p.db.Areas.FillAllMsg(m)
		p.Send(uint16(msg_client_message.S2CGetAreasInfos_ProtoID), m)
	}

	if req.GetStage() {
		p.send_stage_info()
	}

	if req.GetFormula() {
		p.get_formulas()
	}

	if req.GetDepotBuilding() {
		m := &msg_client_message.S2CGetDepotBuildingInfos{}
		p.db.BuildingDepots.FillAllMsg(m)
		p.Send(uint16(msg_client_message.S2CGetDepotBuildingInfos_ProtoID), m)
	}

	if req.GetGuide() {
		p.SyncPlayerGuideData()
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
	p.Send(uint16(msg_client_message.C2SGetHandbook_ProtoID), response)
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

func (this *Player) get_stage_total_score_rank_list(rank_start, rank_num int32) int32 {
	if rank_num > global_config.RankingListOnceGetItemsNum {
		return int32(msg_client_message.E_ERR_RANK_GET_ITEMS_NUM_OVER_MAX)
	}

	/*result := this.rpc_call_ranklist_stage_total_score(rank_start, rank_num)
	if result == nil {
		log.Error("Player[%v] rpc get stages total score rank list range[%v,%v] failed", this.Id, rank_start, rank_num)
		return -1
	}

	var items []*msg_client_message.RankingListItemInfo
	if result.RankItems == nil {
		items = make([]*msg_client_message.RankingListItemInfo, 0)
	} else {
		now_time := time.Now()
		items = make([]*msg_client_message.RankingListItemInfo, len(result.RankItems))
		for i := int32(0); i < int32(len(result.RankItems)); i++ {
			r := result.RankItems[i]
			is_friend := this.db.Friends.HasIndex(r.PlayerId)
			is_zaned := this.is_today_zan(r.PlayerId, now_time)
			name, level, head := GetPlayerBaseInfo(r.PlayerId)
			items[i] = &msg_client_message.RankingListItemInfo{
				Rank:                  rank_start + i,
				PlayerId:              r.PlayerId,
				PlayerName:            name,
				PlayerLevel:           level,
				PlayerHead:            head,
				PlayerStageTotalScore: r.TotalScore,
				IsFriend:              is_friend,
				IsZaned:               is_zaned,
			}
		}
	}

	response := &msg_client_message.S2CPullRankingListResult{}
	response.ItemList = items
	response.RankType = 1
	response.StartRank = rank_start
	response.SelfRank = result.SelfRank
	if result.SelfRank == 0 {
		response.SelfValue1 = this.db.Stages.GetTotalScore()
	} else {
		response.SelfValue1 = result.SelfTotalScore
	}
	this.Send(uint16(msg_client_message.S2CPullRankingListResult_ProtoID), response)*/

	return 1
}

func (this *Player) get_stage_score_rank_list(stage_id, rank_start, rank_num int32) int32 {
	if rank_num > global_config.RankingListOnceGetItemsNum {
		return int32(msg_client_message.E_ERR_RANK_GET_ITEMS_NUM_OVER_MAX)
	}

	/*result := this.rpc_call_ranklist_stage_score(stage_id, rank_start, rank_num)
	if result == nil {
		log.Error("Player[%v] rpc get stage[%v] score rank list range[%v,%v] failed", this.Id, stage_id, rank_start, rank_num)
		return -1
	}

	var items []*msg_client_message.RankingListItemInfo
	if result.RankItems == nil {
		items = make([]*msg_client_message.RankingListItemInfo, 0)
	} else {
		now_time := time.Now()
		items = make([]*msg_client_message.RankingListItemInfo, len(result.RankItems))
		for i := int32(0); i < int32(len(result.RankItems)); i++ {
			r := result.RankItems[i]
			is_friend := this.db.Friends.HasIndex(r.PlayerId)
			is_zaned := this.is_today_zan(r.PlayerId, now_time)
			name, level, head := GetPlayerBaseInfo(r.PlayerId)
			items[i] = &msg_client_message.RankingListItemInfo{
				Rank:             rank_start + i,
				PlayerId:         r.PlayerId,
				PlayerName:       name,
				PlayerLevel:      level,
				PlayerHead:       head,
				PlayerStageId:    r.StageId,
				PlayerStageScore: r.StageScore,
				IsFriend:         is_friend,
				IsZaned:          is_zaned,
			}
		}
	}

	response := &msg_client_message.S2CPullRankingListResult{}
	response.ItemList = items
	response.RankType = 2
	response.StartRank = rank_start
	response.SelfRank = result.SelfRank
	if result.SelfRank == 0 {
		score, _ := this.db.Stages.GetTopScore(stage_id)
		response.SelfValue1 = score
	} else {
		response.SelfValue1 = result.SelfScore
	}

	this.Send(uint16(msg_client_message.S2CPullRankingListResult_ProtoID), response)*/

	return 1
}

func (this *Player) get_charm_rank_list(rank_start, rank_num int32) int32 {
	if rank_num > global_config.RankingListOnceGetItemsNum {
		return int32(msg_client_message.E_ERR_RANK_GET_ITEMS_NUM_OVER_MAX)
	}

	/*result := this.rpc_call_ranklist_charm(rank_start, rank_num)
	if result == nil {
		log.Error("Player[%v] rpc get charm rank list range[%v,%v] failed", this.Id, rank_start, rank_num)
		return -1
	}

	var items []*msg_client_message.RankingListItemInfo
	if result.RankItems == nil {
		items = make([]*msg_client_message.RankingListItemInfo, 0)
	} else {
		now_time := time.Now()
		items = make([]*msg_client_message.RankingListItemInfo, len(result.RankItems))
		for i := int32(0); i < int32(len(result.RankItems)); i++ {
			r := result.RankItems[i]
			is_friend := this.db.Friends.HasIndex(r.PlayerId)
			is_zaned := this.is_today_zan(r.PlayerId, now_time)
			name, level, head := GetPlayerBaseInfo(r.PlayerId)
			items[i] = &msg_client_message.RankingListItemInfo{
				Rank:        rank_start + i,
				PlayerId:    r.PlayerId,
				PlayerName:  name,
				PlayerLevel: level,
				PlayerHead:  head,
				PlayerCharm: r.Charm,
				IsFriend:    is_friend,
				IsZaned:     is_zaned,
			}
		}
	}

	response := &msg_client_message.S2CPullRankingListResult{}
	response.ItemList = items
	response.RankType = 3
	response.StartRank = rank_start
	response.SelfRank = result.SelfRank
	if result.SelfRank == 0 {
		response.SelfValue1 = this.db.Info.GetCharmVal()
	} else {
		response.SelfValue1 = result.SelfCharm
	}

	this.Send(uint16(msg_client_message.S2CPullRankingListResult_ProtoID), response)*/

	return 1
}

func (this *Player) get_cat_ouqi_rank_list(param, rank_start, rank_num int32) int32 {
	if rank_num > global_config.RankingListOnceGetItemsNum {
		return int32(msg_client_message.E_ERR_RANK_GET_ITEMS_NUM_OVER_MAX)
	}

	/*result := this.rpc_call_ranklist_cat_ouqi(rank_start, rank_num, param)
	if result == nil {
		log.Error("Player[%v] rpc get cat ouqi rank list range[%v,%v] failed", this.Id, rank_start, rank_num)
		return -1
	}

	var items []*msg_client_message.RankingListItemInfo
	if result.RankItems == nil {
		items = make([]*msg_client_message.RankingListItemInfo, 0)
	} else {
		now_time := time.Now()
		items = make([]*msg_client_message.RankingListItemInfo, len(result.RankItems))
		for i := int32(0); i < int32(len(result.RankItems)); i++ {
			r := result.RankItems[i]
			is_friend := this.db.Friends.HasIndex(r.PlayerId)
			is_zaned := this.is_today_zan(r.PlayerId, now_time)
			name, level, head := GetPlayerBaseInfo(r.PlayerId)
			items[i] = &msg_client_message.RankingListItemInfo{
				Rank:        rank_start + i,
				PlayerId:    r.PlayerId,
				PlayerName:  name,
				PlayerLevel: level,
				PlayerHead:  head,
				CatId:       r.CatId,
				CatTableId:  r.CatTableId,
				CatLevel:    r.CatLevel,
				CatStar:     r.CatStar,
				CatNick:     r.CatNick,
				CatOuqi:     r.CatOuqi,
				IsFriend:    is_friend,
				IsZaned:     is_zaned,
			}
		}
	}
	response := &msg_client_message.S2CPullRankingListResult{}
	response.ItemList = items
	response.RankType = 4
	response.StartRank = rank_start
	response.SelfRank = result.SelfRank
	response.SelfValue1 = result.SelfCatId
	response.SelfValue2 = result.SelfCatOuqi

	this.Send(uint16(msg_client_message.S2CPullRankingListResult_ProtoID), response)*/

	return 1
}

func (this *Player) get_zaned_rank_list(rank_start, rank_num int32) int32 {
	if rank_num > global_config.RankingListOnceGetItemsNum {
		return int32(msg_client_message.E_ERR_RANK_GET_ITEMS_NUM_OVER_MAX)
	}

	/*result := this.rpc_call_ranklist_get_zaned(rank_start, rank_num)
	if result == nil {
		log.Error("Player[%v] rpc get zaned rank list range[%v,%v] failed", this.Id, rank_start, rank_num)
		return -1
	}

	var items []*msg_client_message.RankingListItemInfo
	if result.RankItems == nil {
		items = make([]*msg_client_message.RankingListItemInfo, 0)
	} else {
		now_time := time.Now()
		items = make([]*msg_client_message.RankingListItemInfo, len(result.RankItems))
		for i := int32(0); i < int32(len(result.RankItems)); i++ {
			r := result.RankItems[i]
			is_friend := this.db.Friends.HasIndex(r.PlayerId)
			is_zaned := this.is_today_zan(r.PlayerId, now_time)
			name, level, head := GetPlayerBaseInfo(r.PlayerId)
			items[i] = &msg_client_message.RankingListItemInfo{
				Rank:        rank_start + i,
				PlayerId:    r.PlayerId,
				PlayerName:  name,
				PlayerLevel: level,
				PlayerHead:  head,
				PlayerZaned: r.Zaned,
				IsFriend:    is_friend,
				IsZaned:     is_zaned,
			}
		}
	}
	response := &msg_client_message.S2CPullRankingListResult{}
	response.ItemList = items
	response.RankType = 5
	response.StartRank = rank_start
	response.SelfRank = result.SelfRank
	response.SelfValue1 = result.SendZaned
	this.Send(uint16(msg_client_message.S2CPullRankingListResult_ProtoID), response)*/

	return 1
}

func C2SPullRankingListHandler(p *Player, msg_data []byte) int32 {
	var req msg_client_message.C2SRankListRequest
	err := proto.Unmarshal(msg_data, &req)
	if err != nil {
		log.Error("unmarshal msg failed err(%s) !", err.Error())
		return -1
	}

	var res int32 = 0
	rank_type := req.GetRankType()
	rank_start := req.GetStartRank()
	if rank_start <= 0 {
		log.Warn("Player[%v] get rank list by type[%v] with rank_start[%v] invalid", p.Id, rank_type, rank_start)
		return -1
	}
	rank_num := req.GetRankNum()
	if rank_num <= 0 {
		log.Warn("Player[%v] get rank list by type[%v] with rank_num[%v] invalid", p.Id, rank_type, rank_num)
		return -1
	}
	/*param := req.GetParam()
	if rank_type == 1 {
		// 关卡总分
		res = p.get_stage_total_score_rank_list(rank_start, rank_num)
	} else if rank_type == 2 {
		// 关卡积分
		res = p.get_stage_score_rank_list(param, rank_start, rank_num)
	} else if rank_type == 3 {
		// 魅力
		res = p.get_charm_rank_list(rank_start, rank_num)
	} else if rank_type == 4 {
		// 欧气值
		res = p.get_cat_ouqi_rank_list(param, rank_start, rank_num)
	} else if rank_type == 5 {
		// 被赞
		res = p.get_zaned_rank_list(rank_start, rank_num)
	} else {
		res = -1
		log.Error("Player[%v] pull rank_type[%v] invalid", p.Id, rank_type)
	}*/

	return res
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
