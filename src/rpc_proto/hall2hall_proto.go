package rpc_proto

import (
	"mm_server/libs/rpc"
)

// 修改基本信息
type H2H_BaseInfo struct {
	FromPlayerId int32
	Nick         string
	Level        int32
	Head         string
}
type H2H_BaseInfoResult struct {
	Error int32
}

// 搜索好友
type H2H_SearchFriend struct {
	PlayerId int32
}
type H2H_SearchFriendResult struct {
	PlayerId   int32
	PlayerName string
}

// 添加好友
type H2H_AddFriend struct {
	FromPlayerId    int32
	FromPlayerName  string
	FromPlayerHead  string
	FromPlayerLevel int32
	ToPlayerId      int32
}
type H2H_AddFriendResult struct {
	FromPlayerId int32
	ToPlayerId   int32
	Error        int32 // 1 对方好友已满
}

// 同意加为好友
type H2H_AgreeAddFriend struct {
	FromPlayerId int32
	ToPlayerId   int32
}
type H2H_AgreeAddFriendResult struct {
	FromPlayerId    int32
	FromPlayerName  string
	FromPlayerHead  string
	FromPlayerLevel int32
	ToPlayerId      int32
}

// 删除好友
type H2H_RemoveFriend struct {
	FromPlayerId int32
	ToPlayerId   int32
}
type H2H_RemoveFriendResult struct {
	FromPlayerId int32
	ToPlayerId   int32
}

// 获取好友数据
type H2H_GetFriendInfo struct {
	PlayerId int32
}
type H2H_GetFriendInfoResult struct {
	PlayerId   int32
	PlayerName string
	Level      int32
	VipLevel   int32
	Head       string
	LastLogin  int32
}

// 赠送友情点
type H2H_GiveFriendPoints struct {
	FromPlayerId int32
	ToPlayerId   int32
	GivePoints   int32
}
type H2H_GiveFriendPointsResult struct {
	FromPlayerId  int32
	ToPlayerId    int32
	GivePoints    int32
	LastSave      int32
	RemainSeconds int32
	Error         int32
}

// 好友聊天
type H2H_FriendChat struct {
	FromPlayerId int32
	ToPlayerId   int32
	Message      []byte
}
type H2H_FriendChatResult struct {
	FromPlayerId int32
	ToPlayerId   int32
	Message      []byte
	Error        int32
}

// 刷新赠送友情点
type H2H_RefreshGiveFriendPoints struct {
	FromPlayerId int32
	ToPlayerId   int32
}
type H2H_RefreshGiveFriendPointsResult struct {
}

// 赞
type H2H_ZanPlayer struct {
	FromPlayerId int32
	ToPlayerId   int32
}
type H2H_ZanPlayerResult struct {
	FromPlayerId   int32
	ToPlayerId     int32
	ToPlayerZanNum int32
}

// 寄养猫到好友寄养所
type H2H_FosterCat2Friend struct {
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
type H2H_FosterCat2FriendResult struct {
	//FromPlayerId    int32
	//FromPlayerCatId int32
	//ToFriendId      int32
}

// 结算给好友寄养所收益
type H2H_FosterSettlement2Friend struct {
	FromPlayerId   int32
	ToPlayerId     int32
	ToPlayerCatId  int32
	ToPlayerCatExp int32
	ToPlayerItems  map[int32]int32
}
type H2H_FosterSettlement2FriendResult struct {
	// 不需要返回任何数据
}

// 结算好友的寄养所
type H2H_FosterSettlementPlayersCatWithFriend struct {
	FromPlayerId int32
	ToPlayerId   int32
}
type H2H_FosterSettlementPlayersCatWithFriendResult struct {
}

// 获得寄养在好友的猫
type H2H_FosterGetCatInfoOnFriend struct {
	FromPlayerId    int32
	ToPlayerId      int32
	FromPlayerCatId int32
}
type H2H_FosterGetCatInfoOnFriendResult struct {
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
type H2H_FosterGetPlayerFosterData struct {
	FromPlayerId int32
	ToPlayerId   int32
}
type H2H_FosterCat struct {
	CatTableId int32
	CatLevel   int32
	CatStar    int32
}
type H2H_FosteredCat struct {
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
type H2H_FosterGetPlayerFosterDataResult struct {
	FromPlayerId      int32
	ToPlayerId        int32
	FosterCardId      int32             // 寄养卡ID
	CardRemainSeconds int32             // 寄养卡剩余时间
	PlayerCats        []H2H_FosterCat   // 玩家寄养的猫
	PlayerFriendCats  []H2H_FosteredCat // 玩家好友寄养的猫
	FosteredSlot      int32             // 好友寄养所总槽位
}

// 获取有寄养空位的好友数据
type H2H_FosterGetEmptySlotFriendInfo struct {
	FromPlayerId int32
	ToPlayerId   int32
}
type H2H_FosterGetEmptySlotFriendInfoResult struct {
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
type H2H_WorldChat struct {
	FromPlayerId    int32
	FromPlayerLevel int32
	FromPlayerName  string
	FromPlayerHead  string
	ChatContent     []byte
}
type H2H_WorldChatResult struct {
}

// 公告
type H2H_Anouncement struct {
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
type H2H_AnouncementResult struct {
}

// 拜访玩家基地
type H2H_VisitPlayer struct {
	FromPlayerId int32
	ToPlayerId   int32
}
type H2H_CropData struct {
	CropId        int32
	RemainSeconds int32
}
type H2H_CatHouseData struct {
	CatIds        []int32
	CatHouseLevel int32
	IsDone        bool
}
type H2H_BuildingInfo struct {
	BuildingId      int32
	BuildingTableId int32
	CordX           int32
	CordY           int32
	Dir             int32
	CropData        *H2H_CropData
	CatHouseData    *H2H_CatHouseData
}
type H2H_AreaInfo struct {
	TableId int32
}
type H2H_VisitPlayerResult struct {
	FromPlayerId     int32
	ToPlayerId       int32
	ToPlayerName     string // 昵称
	ToPlayerLevel    int32  // 等级
	ToPlayerVipLevel int32  // VIP等级
	ToPlayerHead     string // 头像
	ToPlayerGold     int32  // 金币
	ToPlayerDiamond  int32  // 钻石
	ToPlayerCharm    int32  // 魅力值
	Buildings        []*H2H_BuildingInfo
	Areas            []*H2H_AreaInfo
}

// 获取玩家宝箱配置ID
type H2H_GetPlayerChestTableId struct {
	FromPlayerId int32
	ToPlayerId   int32
	ChestId      int32
}
type H2H_GetPlayerChestTableIdResult struct {
	FromPlayerId int32
	ToPlayerId   int32
	ChestId      int32
	ChestTableId int32
	Error        int32
}

// 打开好友宝箱
type H2H_OpenFriendChest struct {
	FromPlayerId int32
	ToPlayerId   int32
	ChestId      int32
}
type H2H_OpenFriendChestResult struct {
	FromPlayerId int32
	ToPlayerId   int32
	ChestTableId int32
}

// 获取玩家猫的数据
type H2H_PlayerCatInfo struct {
	FromPlayerId  int32
	ToPlayerId    int32
	ToPlayerCatId int32
}
type H2H_PlayerCatInfoResult struct {
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

func RegisterRpcUserType() {
	rpc.RegisterUserType(&H2H_BaseInfo{})
	rpc.RegisterUserType(&H2H_BaseInfoResult{})
	rpc.RegisterUserType(&H2H_GetFriendInfo{})
	rpc.RegisterUserType(&H2H_GetFriendInfoResult{})
	rpc.RegisterUserType(&H2H_SearchFriend{})
	rpc.RegisterUserType(&H2H_AddFriend{})
	rpc.RegisterUserType(&H2H_AgreeAddFriend{})
	rpc.RegisterUserType(&H2H_RemoveFriend{})
	rpc.RegisterUserType(&H2H_GiveFriendPoints{})
	rpc.RegisterUserType(&H2H_VisitPlayer{})
	rpc.RegisterUserType(&H2H_CropData{})
	rpc.RegisterUserType(&H2H_CatHouseData{})
	rpc.RegisterUserType(&H2H_BuildingInfo{})
	rpc.RegisterUserType(&H2H_FriendChat{})
	rpc.RegisterUserType(&H2H_ZanPlayer{})
	rpc.RegisterUserType(&H2H_FosterCat2Friend{})
	rpc.RegisterUserType(&H2H_FosterSettlement2Friend{})
	rpc.RegisterUserType(&H2H_FosterSettlementPlayersCatWithFriend{})
	rpc.RegisterUserType(&H2H_FosterGetCatInfoOnFriend{})
	rpc.RegisterUserType(&H2H_FosterGetPlayerFosterData{})
	rpc.RegisterUserType(&H2H_FosterCat{})
	rpc.RegisterUserType(&H2H_FosteredCat{})
	rpc.RegisterUserType(&H2H_FosterGetPlayerFosterData{})
	rpc.RegisterUserType(&H2H_FosterGetEmptySlotFriendInfo{})
	rpc.RegisterUserType(&H2H_FosterGetEmptySlotFriendInfoResult{})
	rpc.RegisterUserType(&H2H_WorldChat{})
	rpc.RegisterUserType(&H2H_WorldChatResult{})
	rpc.RegisterUserType(&H2H_Anouncement{})
	rpc.RegisterUserType(&H2H_AnouncementResult{})
	rpc.RegisterUserType(&H2H_VisitPlayerResult{})
	rpc.RegisterUserType(&H2H_OpenFriendChest{})
	rpc.RegisterUserType(&H2H_OpenFriendChestResult{})
	rpc.RegisterUserType(&H2H_PlayerCatInfo{})
	rpc.RegisterUserType(&H2H_PlayerCatInfoResult{})
}
