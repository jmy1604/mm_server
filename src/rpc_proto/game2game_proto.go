package rpc_proto

import (
	"mm_server/libs/rpc"
	"mm_server/src/common"

	"mm_server/proto/gen_go/client_message"
)

// 修改基本信息
type G2G_BaseInfo struct {
	FromPlayerId int32
	Nick         string
	Level        int32
	Head         string
}
type G2G_BaseInfoResult struct {
	Error int32
}

// 搜索好友
type G2G_SearchFriend struct {
	PlayerId int32
}
type G2G_SearchFriendResult struct {
	PlayerId   int32
	PlayerName string
}

// 添加好友
type G2G_AddFriend struct {
	FromPlayerId    int32
	FromPlayerName  string
	FromPlayerHead  string
	FromPlayerLevel int32
	ToPlayerId      int32
}
type G2G_AddFriendResult struct {
	FromPlayerId int32
	ToPlayerId   int32
	Error        int32 // 1 对方好友已满
}

// 同意加为好友
type G2G_AgreeAddFriend struct {
	FromPlayerId int32
	ToPlayerId   int32
}
type G2G_AgreeAddFriendResult struct {
	FromPlayerId    int32
	FromPlayerName  string
	FromPlayerHead  string
	FromPlayerLevel int32
	ToPlayerId      int32
}

// 删除好友
type G2G_RemoveFriend struct {
	FromPlayerId int32
	ToPlayerId   int32
}
type G2G_RemoveFriendResult struct {
	FromPlayerId int32
	ToPlayerId   int32
}

// 获取好友数据
type G2G_GetFriendInfo struct {
	PlayerId int32
}
type G2G_GetFriendInfoResult struct {
	PlayerId   int32
	PlayerName string
	Level      int32
	VipLevel   int32
	Head       string
	LastLogin  int32
}

// 赠送友情点
type G2G_GiveFriendPoints struct {
	FromPlayerId int32
	ToPlayerId   int32
	GivePoints   int32
}
type G2G_GiveFriendPointsResult struct {
	FromPlayerId  int32
	ToPlayerId    int32
	GivePoints    int32
	LastSave      int32
	RemainSeconds int32
	Error         int32
}

// 好友聊天
type G2G_FriendChat struct {
	FromPlayerId int32
	ToPlayerId   int32
	Message      []byte
}
type G2G_FriendChatResult struct {
	FromPlayerId int32
	ToPlayerId   int32
	Message      []byte
	Error        int32
}

// 刷新赠送友情点
type G2G_RefreshGiveFriendPoints struct {
	FromPlayerId int32
	ToPlayerId   int32
}
type G2G_RefreshGiveFriendPointsResult struct {
}

// 赞
type G2G_ZanPlayer struct {
	FromPlayerId int32
	ToPlayerId   int32
}
type G2G_ZanPlayerResult struct {
	FromPlayerId   int32
	ToPlayerId     int32
	ToPlayerZanNum int32
}

// 寄养猫到好友寄养所
type G2G_FosterCat2Friend struct {
	FromPlayerId         int32
	FromPlayerLevel      int32
	FromPlayerName       string
	FromPlayerHead       string
	FromPlayerCatId      int32
	FromPlayerCatTableId int32
	FromPlayerCatLevel   int32
	FromPlayerCatStar    int32
	ToFriendId           int32
}
type G2G_FosterCat2FriendResult struct {
	//FromPlayerId    int32
	//FromPlayerCatId int32
	//ToFriendId      int32
}

// 结算给好友寄养所收益
type G2G_FosterSettlement2Friend struct {
	FromPlayerId   int32
	ToPlayerId     int32
	ToPlayerCatId  int32
	ToPlayerCatExp int32
	ToPlayerItems  map[int32]int32
}
type G2G_FosterSettlement2FriendResult struct {
	// 不需要返回任何数据
}

// 结算好友的寄养所
type G2G_FosterSettlementPlayersCatWithFriend struct {
	FromPlayerId int32
	ToPlayerId   int32
}
type G2G_FosterSettlementPlayersCatWithFriendResult struct {
}

// 获得寄养在好友的猫
type G2G_FosterGetCatInfoOnFriend struct {
	FromPlayerId    int32
	ToPlayerId      int32
	FromPlayerCatId int32
}
type G2G_FosterGetCatInfoOnFriendResult struct {
	FromPlayerId     int32
	FromPlayerCatId  int32
	ToFriendId       int32
	RemainSeconds    int32  // 剩余寄养时间
	ToFriendLevel    int32  // 好友等级
	ToFriendName     string // 好友昵称
	ToFriendHead     string // 好友头像
	StartCardId      int32  // 放入猫时的寄养卡
	FromPlayerCatExp int32
	FromPlayerItems  map[int32]int32
}

// 获取玩家的寄养数据
type G2G_FosterGetPlayerFosterData struct {
	FromPlayerId int32
	ToPlayerId   int32
}
type G2G_FosterCat struct {
	CatTableId int32
	CatLevel   int32
	CatStar    int32
}
type G2G_FosteredCat struct {
	StartCardId    int32
	RemainSeconds  int32
	CatTableId     int32
	CatLevel       int32
	CatStar        int32
	CatNick        string
	PlayerId       int32
	PlayerLevel    int32
	PlayerVipLevel int32
	PlayerName     string
	PlayerHead     string
}
type G2G_FosterGetPlayerFosterDataResult struct {
	FromPlayerId      int32
	ToPlayerId        int32
	FosterCardId      int32             // 寄养卡ID
	CardRemainSeconds int32             // 寄养卡剩余时间
	PlayerCats        []G2G_FosterCat   // 玩家寄养的猫
	PlayerFriendCats  []G2G_FosteredCat // 玩家好友寄养的猫
	FosteredSlot      int32             // 好友寄养所总槽位
}

// 获取有寄养空位的好友数据
type G2G_FosterGetEmptySlotFriendInfo struct {
	FromPlayerId int32
	ToPlayerId   int32
}
type G2G_FosterGetEmptySlotFriendInfoResult struct {
	FromPlayerId      int32
	ToPlayerId        int32
	ToPlayerName      string
	ToPlayerLevel     int32
	ToPlayerHead      string
	ToPlayerVipLevel  int32
	ToPlayerLastLogin int32
	FosterCardId      int32
}

// 通知世界聊天
type G2G_WorldChat struct {
	FromPlayerId    int32
	FromPlayerLevel int32
	FromPlayerName  string
	FromPlayerHead  string
	ChatContent     []byte
}
type G2G_WorldChatResult struct {
}

// 公告
type G2G_Anouncement struct {
	MsgType         int32
	FromPlayerId    int32
	FromPlayerLevel int32
	FromPlayerName  string
	FromPlayerHead  string
	MsgParam1       int32
	MsgParam2       int32
	MsgParam3       int32
	MsgText         string
}
type G2G_AnouncementResult struct {
}

// 拜访玩家基地
type G2G_VisitPlayer struct {
	FromPlayerId int32
	ToPlayerId   int32
}
type G2G_CropData struct {
	CropId        int32
	RemainSeconds int32
}
type G2G_CatHouseData struct {
	CatIds        []int32
	CatHouseLevel int32
	IsDone        bool
}
type G2G_BuildingInfo struct {
	BuildingId      int32
	BuildingTableId int32
	CordX           int32
	CordY           int32
	Dir             int32
	CropData        *G2G_CropData
	CatHouseData    *G2G_CatHouseData
}
type G2G_AreaInfo struct {
	TableId int32
}
type G2G_VisitPlayerResult struct {
	FromPlayerId     int32
	ToPlayerId       int32
	ToPlayerName     string // 昵称
	ToPlayerLevel    int32  // 等级
	ToPlayerVipLevel int32  // VIP等级
	ToPlayerHead     string // 头像
	ToPlayerGold     int32  // 金币
	ToPlayerDiamond  int32  // 钻石
	ToPlayerCharm    int32  // 魅力值
	Buildings        []*G2G_BuildingInfo
	Areas            []*G2G_AreaInfo
}

// 获取玩家宝箱配置ID
type G2G_GetPlayerChestTableId struct {
	FromPlayerId int32
	ToPlayerId   int32
	ChestId      int32
}
type G2G_GetPlayerChestTableIdResult struct {
	FromPlayerId int32
	ToPlayerId   int32
	ChestId      int32
	ChestTableId int32
	Error        int32
}

// 打开好友宝箱
type G2G_OpenFriendChest struct {
	FromPlayerId int32
	ToPlayerId   int32
	ChestId      int32
}
type G2G_OpenFriendChestResult struct {
	FromPlayerId int32
	ToPlayerId   int32
	ChestTableId int32
}

// 获取玩家猫的数据
type G2G_PlayerCatInfo struct {
	FromPlayerId  int32
	ToPlayerId    int32
	ToPlayerCatId int32
}
type G2G_PlayerCatInfoResult struct {
	FromPlayerId          int32
	ToPlayerId            int32
	ToPlayerCatId         int32
	ToPlayerCatLevel      int32
	ToPlayerCatStar       int32
	ToPlayerCatExp        int32
	ToPlayerCatSkillLevel int32
	ToPlayerCatAddCoin    int32
	ToPlayerCatAddMatch   int32
	ToPlayerCatAddExplore int32
	Error                 int32
}

// 访问玩家
type G2G_PlayerVisitPlayer struct {
	FromPlayerId int32
	ToPlayerId   int32
}
type G2G_PlayerVisitPlayerResult struct {
	FromPlayerId int32
	ToPlayerId   int32
	VisitData    *msg_client_message.S2CVisitPlayerResult
}

func RegisterRpcUserType() {
	rpc.RegisterUserType(&G2G_BaseInfo{})
	rpc.RegisterUserType(&G2G_BaseInfoResult{})
	rpc.RegisterUserType(&G2G_GetFriendInfo{})
	rpc.RegisterUserType(&G2G_GetFriendInfoResult{})
	rpc.RegisterUserType(&G2G_SearchFriend{})
	rpc.RegisterUserType(&G2G_AddFriend{})
	rpc.RegisterUserType(&G2G_AgreeAddFriend{})
	rpc.RegisterUserType(&G2G_RemoveFriend{})
	rpc.RegisterUserType(&G2G_GiveFriendPoints{})
	rpc.RegisterUserType(&G2G_VisitPlayer{})
	rpc.RegisterUserType(&G2G_CropData{})
	rpc.RegisterUserType(&G2G_CatHouseData{})
	rpc.RegisterUserType(&G2G_BuildingInfo{})
	rpc.RegisterUserType(&G2G_FriendChat{})
	rpc.RegisterUserType(&G2G_ZanPlayer{})
	rpc.RegisterUserType(&G2G_FosterCat2Friend{})
	rpc.RegisterUserType(&G2G_FosterSettlement2Friend{})
	rpc.RegisterUserType(&G2G_FosterSettlementPlayersCatWithFriend{})
	rpc.RegisterUserType(&G2G_FosterGetCatInfoOnFriend{})
	rpc.RegisterUserType(&G2G_FosterGetPlayerFosterData{})
	rpc.RegisterUserType(&G2G_FosterCat{})
	rpc.RegisterUserType(&G2G_FosteredCat{})
	rpc.RegisterUserType(&G2G_FosterGetPlayerFosterData{})
	rpc.RegisterUserType(&G2G_FosterGetEmptySlotFriendInfo{})
	rpc.RegisterUserType(&G2G_FosterGetEmptySlotFriendInfoResult{})
	rpc.RegisterUserType(&G2G_WorldChat{})
	rpc.RegisterUserType(&G2G_WorldChatResult{})
	rpc.RegisterUserType(&G2G_Anouncement{})
	rpc.RegisterUserType(&G2G_AnouncementResult{})
	rpc.RegisterUserType(&G2G_VisitPlayerResult{})
	rpc.RegisterUserType(&G2G_OpenFriendChest{})
	rpc.RegisterUserType(&G2G_OpenFriendChestResult{})
	rpc.RegisterUserType(&G2G_PlayerCatInfo{})
	rpc.RegisterUserType(&G2G_PlayerCatInfoResult{})
	rpc.RegisterUserType(&common.PlayerInt32RankItem{})
	rpc.RegisterUserType(&common.PlayerInt64RankItem{})
	rpc.RegisterUserType(&common.PlayerCatOuqiRankItem{})
	rpc.RegisterUserType(&msg_client_message.ViewBuildingInfo{})
	rpc.RegisterUserType(&msg_client_message.AreaInfo{})
	rpc.RegisterUserType(&msg_client_message.S2CVisitPlayerResult{})
}
