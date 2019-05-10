package rpc_proto

import (
	"mm_server/libs/utils"
)

// 转发消息
type G2R_Transfer struct {
	Method          string
	Args            interface{}
	ReceivePlayerId int32
}
type G2R_TransferResult struct {
	Result interface{}
}

// ping RPC服务
type G2R_Ping struct {
}

type G2R_Pong struct {
}

// 大厅通知RPC监听端口
type G2R_ListenIPNoitfy struct {
	ListenIP string
	ServerId int32
}
type G2R_ListenIPResult struct {
}

type G2R_SearchFriend struct {
	Key string
}

// 玩家搜索好友数据
type SearchPlayerInfo struct {
	Id        int32
	Nick      string
	Level     int32
	VipLevel  int32
	Head      int32
	LastLogin int32
}

// 搜索好友结果
type G2R_SearchFriendResult struct {
	Players []*SearchPlayerInfo
}

// 好友申请
type G2R_AddFriendById struct {
	PlayerId    int32
	PlayerName  string
	AddPlayerId int32
}
type G2R_AddFriendByName struct {
	PlayerId      int32
	PlayerName    string
	AddPlayerName string
}
type G2R_AddFriendResult struct {
	PlayerId    int32
	AddPlayerId int32
	Error       int32
}

// 同意或拒绝好友申请
type G2R_AgreeAddFriend struct {
	IsAgree       bool
	PlayerId      int32
	PlayerName    string
	AgreePlayerId int32
}
type G2R_AgreeAddFriendResult struct {
	IsAgree              bool
	PlayerId             int32
	AgreePlayerId        int32
	AgreePlayerName      string
	AgreePlayerLevel     int32
	AgreePlayerVipLevel  int32
	AgreePlayerHead      string
	AgreePlayerLastLogin int32
}

// 删除好友
type G2R_RemoveFriend struct {
	PlayerId       int32
	RemovePlayerId int32
}
type G2R_RemoveFriendResult struct {
}

// 获取好友数据
type G2R_GetFriendInfo struct {
	PlayerId int32
}
type G2R_GetFriendInfoResult struct {
	PlayerId   int32
	PlayerName string
	Head       string
	Level      int32
	VipLevel   int32
	LastLogin  int32
}

// 删除玩家排名
type G2R_RankDelete struct {
	PlayerId int32
	RankType int32
	Param    int32
}
type G2R_RankDeleteResult struct {
	PlayerId int32
	RankType int32
	Param    int32
}

// 充值记录
type G2R_ChargeSave struct {
	Channel    int32
	OrderId    string
	BundleId   string
	Account    string
	PlayerId   int32
	PayTime    int32
	PayTimeStr string
}

type G2R_ChargeSaveResult struct {
}

// 玩家基础信息
type PlayerBaseInfo struct {
	Id    int32
	Name  string
	Level int32
	Head  int32
}

// 更新玩家基础信息
type G2R_PlayerBaseInfoUpdate struct {
	Info *PlayerBaseInfo
}

type G2R_PlayerBaseInfoUpdateResult struct {
}

// 排行榜数据更新
type G2R_RankListDataUpdate struct {
	RankType  int32
	PlayerId  int32
	RankParam []int32
}

type G2R_RankListDataUpdateResult struct {
}

// 排行榜获取数据
type G2R_RankListGetData struct {
	RankType  int32
	PlayerId  int32
	StartRank int32
	RankNum   int32
	RankParam int32
}

type G2R_RankListGetDataResult struct {
	RankType           int32
	PlayerId           int32
	StartRank          int32
	RankNum            int32
	RankItems          []utils.SkiplistNode
	SelfRank           int32
	SelfValue          interface{}
	SelfHistoryTopRank int32
	PlayerBaseInfos    map[int32]*PlayerBaseInfo
}

// 获取好友关卡积分
type G2R_GetFriendStageScore struct {
	PlayerId  int32
	FriendIds []int32
	StageId   int32
}

type FriendStageScoreData struct {
	Id         int32
	Name       string
	Level      int32
	Head       int32
	StageScore int32
}

type G2R_GetFriendStageScoreResult struct {
	FriendsScoreData []*FriendStageScoreData
}

// 获得多个玩家基础信息
type G2R_GetPlayersBaseInfo struct {
	PlayerIds []int32
}

type G2R_GetPlayersBaseInfoResult struct {
	PlayersInfo []*PlayerBaseInfo
}
