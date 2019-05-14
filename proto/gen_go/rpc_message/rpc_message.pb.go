// Code generated by protoc-gen-go. DO NOT EDIT.
// source: rpc_message.proto

/*
Package msg_rpc_message is a generated protocol buffer package.

It is generated from these files:
	rpc_message.proto

It has these top-level messages:
	G2GPlayerInfoRequest
	G2GPlayerInfoResponse
	G2GPlayerInfoNotify
	G2GPlayerBattleInfoRequest
	G2GPlayerBattleInfoResponse
	G2GPlayerMultiInfoRequest
	PlayerInfo
	G2GPlayerMultiInfoResponse
	G2GFriendAskRequest
	G2GFriendAskResponse
	G2GFriendAgreeRequest
	G2GFriendAgreeResponse
	G2GFriendRemoveRequest
	G2GFriendRemoveResponse
	G2GFriendsInfoRequest
	FriendInfo
	G2GFriendsInfoResponse
	G2GFriendsRefreshGivePointsRequest
	G2GFriendsRefreshGivePointsResponse
	G2GFriendGivePointsRequest
	G2GFriendGivePointsResponse
	G2GFriendChatRequest
	G2GFriendChatResponse
*/
package msg_rpc_message

import proto "github.com/golang/protobuf/proto"
import fmt "fmt"
import math "math"

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf

// This is a compile-time assertion to ensure that this generated file
// is compatible with the proto package it is being compiled against.
// A compilation error at this line likely means your copy of the
// proto package needs to be updated.
const _ = proto.ProtoPackageIsVersion2 // please upgrade the proto package

type MSGID int32

const (
	MSGID_NONE                                     MSGID = 0
	MSGID_G2G_PLAYER_INFO_REQUEST                  MSGID = 1
	MSGID_G2G_PLAYER_INFO_RESPONSE                 MSGID = 2
	MSGID_G2G_PLAYER_INFO_NOTIFY                   MSGID = 3
	MSGID_G2G_PLAYER_BATTLE_INFO_REQUEST           MSGID = 4
	MSGID_G2G_PLAYER_BATTLE_INFO_RESPONSE          MSGID = 5
	MSGID_G2G_PLAYER_MULTI_INFO_REQUEST            MSGID = 10
	MSGID_G2G_PLAYER_MULTI_INFO_RESPONSE           MSGID = 11
	MSGID_G2G_FRIENDS_INFO_REQUEST                 MSGID = 100
	MSGID_G2G_FRIENDS_INFO_RESPONSE                MSGID = 101
	MSGID_G2G_FRIEND_ASK_REQUEST                   MSGID = 102
	MSGID_G2G_FRIEND_ASK_RESPONSE                  MSGID = 103
	MSGID_G2G_FRIEND_AGREE_REQUEST                 MSGID = 104
	MSGID_G2G_FRIEND_AGREE_RESPONSE                MSGID = 105
	MSGID_G2G_FRIEND_REFUSE_REQUEST                MSGID = 106
	MSGID_G2G_FRIEND_REFUSE_RESPONSE               MSGID = 107
	MSGID_G2G_FRIEND_REMOVE_REQUEST                MSGID = 108
	MSGID_G2G_FRIEND_REMOVE_RESPONSE               MSGID = 109
	MSGID_G2G_FRIEND_REMOVE_NOTIFY                 MSGID = 110
	MSGID_G2G_FRIEND_GIVE_POINTS_REQUEST           MSGID = 111
	MSGID_G2G_FRIEND_GIVE_POINTS_RESPONSE          MSGID = 112
	MSGID_G2G_FRIEND_GET_POINTS_REQUEST            MSGID = 113
	MSGID_G2G_FRIEND_GET_POINTS_RESPONSE           MSGID = 114
	MSGID_G2G_FRIENDS_REFRESH_GIVE_POINTS_REQUEST  MSGID = 115
	MSGID_G2G_FRIENDS_REFRESH_GIVE_POINTS_RESPONSE MSGID = 116
	MSGID_G2G_FRIEND_CHAT_REQUEST                  MSGID = 117
	MSGID_G2G_FRIEND_CHAT_RESPONSE                 MSGID = 118
)

var MSGID_name = map[int32]string{
	0:   "NONE",
	1:   "G2G_PLAYER_INFO_REQUEST",
	2:   "G2G_PLAYER_INFO_RESPONSE",
	3:   "G2G_PLAYER_INFO_NOTIFY",
	4:   "G2G_PLAYER_BATTLE_INFO_REQUEST",
	5:   "G2G_PLAYER_BATTLE_INFO_RESPONSE",
	10:  "G2G_PLAYER_MULTI_INFO_REQUEST",
	11:  "G2G_PLAYER_MULTI_INFO_RESPONSE",
	100: "G2G_FRIENDS_INFO_REQUEST",
	101: "G2G_FRIENDS_INFO_RESPONSE",
	102: "G2G_FRIEND_ASK_REQUEST",
	103: "G2G_FRIEND_ASK_RESPONSE",
	104: "G2G_FRIEND_AGREE_REQUEST",
	105: "G2G_FRIEND_AGREE_RESPONSE",
	106: "G2G_FRIEND_REFUSE_REQUEST",
	107: "G2G_FRIEND_REFUSE_RESPONSE",
	108: "G2G_FRIEND_REMOVE_REQUEST",
	109: "G2G_FRIEND_REMOVE_RESPONSE",
	110: "G2G_FRIEND_REMOVE_NOTIFY",
	111: "G2G_FRIEND_GIVE_POINTS_REQUEST",
	112: "G2G_FRIEND_GIVE_POINTS_RESPONSE",
	113: "G2G_FRIEND_GET_POINTS_REQUEST",
	114: "G2G_FRIEND_GET_POINTS_RESPONSE",
	115: "G2G_FRIENDS_REFRESH_GIVE_POINTS_REQUEST",
	116: "G2G_FRIENDS_REFRESH_GIVE_POINTS_RESPONSE",
	117: "G2G_FRIEND_CHAT_REQUEST",
	118: "G2G_FRIEND_CHAT_RESPONSE",
}
var MSGID_value = map[string]int32{
	"NONE": 0,
	"G2G_PLAYER_INFO_REQUEST":                  1,
	"G2G_PLAYER_INFO_RESPONSE":                 2,
	"G2G_PLAYER_INFO_NOTIFY":                   3,
	"G2G_PLAYER_BATTLE_INFO_REQUEST":           4,
	"G2G_PLAYER_BATTLE_INFO_RESPONSE":          5,
	"G2G_PLAYER_MULTI_INFO_REQUEST":            10,
	"G2G_PLAYER_MULTI_INFO_RESPONSE":           11,
	"G2G_FRIENDS_INFO_REQUEST":                 100,
	"G2G_FRIENDS_INFO_RESPONSE":                101,
	"G2G_FRIEND_ASK_REQUEST":                   102,
	"G2G_FRIEND_ASK_RESPONSE":                  103,
	"G2G_FRIEND_AGREE_REQUEST":                 104,
	"G2G_FRIEND_AGREE_RESPONSE":                105,
	"G2G_FRIEND_REFUSE_REQUEST":                106,
	"G2G_FRIEND_REFUSE_RESPONSE":               107,
	"G2G_FRIEND_REMOVE_REQUEST":                108,
	"G2G_FRIEND_REMOVE_RESPONSE":               109,
	"G2G_FRIEND_REMOVE_NOTIFY":                 110,
	"G2G_FRIEND_GIVE_POINTS_REQUEST":           111,
	"G2G_FRIEND_GIVE_POINTS_RESPONSE":          112,
	"G2G_FRIEND_GET_POINTS_REQUEST":            113,
	"G2G_FRIEND_GET_POINTS_RESPONSE":           114,
	"G2G_FRIENDS_REFRESH_GIVE_POINTS_REQUEST":  115,
	"G2G_FRIENDS_REFRESH_GIVE_POINTS_RESPONSE": 116,
	"G2G_FRIEND_CHAT_REQUEST":                  117,
	"G2G_FRIEND_CHAT_RESPONSE":                 118,
}

func (x MSGID) String() string {
	return proto.EnumName(MSGID_name, int32(x))
}
func (MSGID) EnumDescriptor() ([]byte, []int) { return fileDescriptor0, []int{0} }

// 玩家信息请求
type G2GPlayerInfoRequest struct {
}

func (m *G2GPlayerInfoRequest) Reset()                    { *m = G2GPlayerInfoRequest{} }
func (m *G2GPlayerInfoRequest) String() string            { return proto.CompactTextString(m) }
func (*G2GPlayerInfoRequest) ProtoMessage()               {}
func (*G2GPlayerInfoRequest) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{0} }

// 玩家信息返回
type G2GPlayerInfoResponse struct {
	UniqueId string `protobuf:"bytes,2,opt,name=UniqueId" json:"UniqueId,omitempty"`
	Account  string `protobuf:"bytes,3,opt,name=Account" json:"Account,omitempty"`
	Level    int32  `protobuf:"varint,4,opt,name=Level" json:"Level,omitempty"`
	Head     int32  `protobuf:"varint,5,opt,name=Head" json:"Head,omitempty"`
}

func (m *G2GPlayerInfoResponse) Reset()                    { *m = G2GPlayerInfoResponse{} }
func (m *G2GPlayerInfoResponse) String() string            { return proto.CompactTextString(m) }
func (*G2GPlayerInfoResponse) ProtoMessage()               {}
func (*G2GPlayerInfoResponse) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{1} }

func (m *G2GPlayerInfoResponse) GetUniqueId() string {
	if m != nil {
		return m.UniqueId
	}
	return ""
}

func (m *G2GPlayerInfoResponse) GetAccount() string {
	if m != nil {
		return m.Account
	}
	return ""
}

func (m *G2GPlayerInfoResponse) GetLevel() int32 {
	if m != nil {
		return m.Level
	}
	return 0
}

func (m *G2GPlayerInfoResponse) GetHead() int32 {
	if m != nil {
		return m.Head
	}
	return 0
}

// 玩家信息通知
type G2GPlayerInfoNotify struct {
}

func (m *G2GPlayerInfoNotify) Reset()                    { *m = G2GPlayerInfoNotify{} }
func (m *G2GPlayerInfoNotify) String() string            { return proto.CompactTextString(m) }
func (*G2GPlayerInfoNotify) ProtoMessage()               {}
func (*G2GPlayerInfoNotify) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{2} }

// 玩家战斗信息请求
type G2GPlayerBattleInfoRequest struct {
}

func (m *G2GPlayerBattleInfoRequest) Reset()                    { *m = G2GPlayerBattleInfoRequest{} }
func (m *G2GPlayerBattleInfoRequest) String() string            { return proto.CompactTextString(m) }
func (*G2GPlayerBattleInfoRequest) ProtoMessage()               {}
func (*G2GPlayerBattleInfoRequest) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{3} }

// 玩家战斗信息返回
type G2GPlayerBattleInfoResponse struct {
}

func (m *G2GPlayerBattleInfoResponse) Reset()                    { *m = G2GPlayerBattleInfoResponse{} }
func (m *G2GPlayerBattleInfoResponse) String() string            { return proto.CompactTextString(m) }
func (*G2GPlayerBattleInfoResponse) ProtoMessage()               {}
func (*G2GPlayerBattleInfoResponse) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{4} }

// 多个玩家信息请求
type G2GPlayerMultiInfoRequest struct {
}

func (m *G2GPlayerMultiInfoRequest) Reset()                    { *m = G2GPlayerMultiInfoRequest{} }
func (m *G2GPlayerMultiInfoRequest) String() string            { return proto.CompactTextString(m) }
func (*G2GPlayerMultiInfoRequest) ProtoMessage()               {}
func (*G2GPlayerMultiInfoRequest) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{5} }

type PlayerInfo struct {
	PlayerId int32  `protobuf:"varint,1,opt,name=PlayerId" json:"PlayerId,omitempty"`
	UniqueId string `protobuf:"bytes,2,opt,name=UniqueId" json:"UniqueId,omitempty"`
	Account  string `protobuf:"bytes,3,opt,name=Account" json:"Account,omitempty"`
	Level    int32  `protobuf:"varint,4,opt,name=Level" json:"Level,omitempty"`
	Head     int32  `protobuf:"varint,5,opt,name=Head" json:"Head,omitempty"`
}

func (m *PlayerInfo) Reset()                    { *m = PlayerInfo{} }
func (m *PlayerInfo) String() string            { return proto.CompactTextString(m) }
func (*PlayerInfo) ProtoMessage()               {}
func (*PlayerInfo) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{6} }

func (m *PlayerInfo) GetPlayerId() int32 {
	if m != nil {
		return m.PlayerId
	}
	return 0
}

func (m *PlayerInfo) GetUniqueId() string {
	if m != nil {
		return m.UniqueId
	}
	return ""
}

func (m *PlayerInfo) GetAccount() string {
	if m != nil {
		return m.Account
	}
	return ""
}

func (m *PlayerInfo) GetLevel() int32 {
	if m != nil {
		return m.Level
	}
	return 0
}

func (m *PlayerInfo) GetHead() int32 {
	if m != nil {
		return m.Head
	}
	return 0
}

// 多个玩家信息返回
type G2GPlayerMultiInfoResponse struct {
	PlayerInfos []*PlayerInfo `protobuf:"bytes,1,rep,name=PlayerInfos" json:"PlayerInfos,omitempty"`
}

func (m *G2GPlayerMultiInfoResponse) Reset()                    { *m = G2GPlayerMultiInfoResponse{} }
func (m *G2GPlayerMultiInfoResponse) String() string            { return proto.CompactTextString(m) }
func (*G2GPlayerMultiInfoResponse) ProtoMessage()               {}
func (*G2GPlayerMultiInfoResponse) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{7} }

func (m *G2GPlayerMultiInfoResponse) GetPlayerInfos() []*PlayerInfo {
	if m != nil {
		return m.PlayerInfos
	}
	return nil
}

// 添加好友请求
type G2GFriendAskRequest struct {
	FromPlayerId    int32  `protobuf:"varint,1,opt,name=FromPlayerId" json:"FromPlayerId,omitempty"`
	FromPlayerName  string `protobuf:"bytes,2,opt,name=FromPlayerName" json:"FromPlayerName,omitempty"`
	FromPlayerLevel int32  `protobuf:"varint,3,opt,name=FromPlayerLevel" json:"FromPlayerLevel,omitempty"`
	FromPlayerHead  int32  `protobuf:"varint,4,opt,name=FromPlayerHead" json:"FromPlayerHead,omitempty"`
}

func (m *G2GFriendAskRequest) Reset()                    { *m = G2GFriendAskRequest{} }
func (m *G2GFriendAskRequest) String() string            { return proto.CompactTextString(m) }
func (*G2GFriendAskRequest) ProtoMessage()               {}
func (*G2GFriendAskRequest) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{8} }

func (m *G2GFriendAskRequest) GetFromPlayerId() int32 {
	if m != nil {
		return m.FromPlayerId
	}
	return 0
}

func (m *G2GFriendAskRequest) GetFromPlayerName() string {
	if m != nil {
		return m.FromPlayerName
	}
	return ""
}

func (m *G2GFriendAskRequest) GetFromPlayerLevel() int32 {
	if m != nil {
		return m.FromPlayerLevel
	}
	return 0
}

func (m *G2GFriendAskRequest) GetFromPlayerHead() int32 {
	if m != nil {
		return m.FromPlayerHead
	}
	return 0
}

// 添加好友返回
type G2GFriendAskResponse struct {
}

func (m *G2GFriendAskResponse) Reset()                    { *m = G2GFriendAskResponse{} }
func (m *G2GFriendAskResponse) String() string            { return proto.CompactTextString(m) }
func (*G2GFriendAskResponse) ProtoMessage()               {}
func (*G2GFriendAskResponse) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{9} }

// 好友同意
type G2GFriendAgreeRequest struct {
	Info *FriendInfo `protobuf:"bytes,1,opt,name=Info" json:"Info,omitempty"`
}

func (m *G2GFriendAgreeRequest) Reset()                    { *m = G2GFriendAgreeRequest{} }
func (m *G2GFriendAgreeRequest) String() string            { return proto.CompactTextString(m) }
func (*G2GFriendAgreeRequest) ProtoMessage()               {}
func (*G2GFriendAgreeRequest) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{10} }

func (m *G2GFriendAgreeRequest) GetInfo() *FriendInfo {
	if m != nil {
		return m.Info
	}
	return nil
}

// 好友同意返回
type G2GFriendAgreeResponse struct {
}

func (m *G2GFriendAgreeResponse) Reset()                    { *m = G2GFriendAgreeResponse{} }
func (m *G2GFriendAgreeResponse) String() string            { return proto.CompactTextString(m) }
func (*G2GFriendAgreeResponse) ProtoMessage()               {}
func (*G2GFriendAgreeResponse) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{11} }

// 好友删除
type G2GFriendRemoveRequest struct {
}

func (m *G2GFriendRemoveRequest) Reset()                    { *m = G2GFriendRemoveRequest{} }
func (m *G2GFriendRemoveRequest) String() string            { return proto.CompactTextString(m) }
func (*G2GFriendRemoveRequest) ProtoMessage()               {}
func (*G2GFriendRemoveRequest) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{12} }

// 好友删除返回
type G2GFriendRemoveResponse struct {
}

func (m *G2GFriendRemoveResponse) Reset()                    { *m = G2GFriendRemoveResponse{} }
func (m *G2GFriendRemoveResponse) String() string            { return proto.CompactTextString(m) }
func (*G2GFriendRemoveResponse) ProtoMessage()               {}
func (*G2GFriendRemoveResponse) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{13} }

// 多个好友信息
type G2GFriendsInfoRequest struct {
}

func (m *G2GFriendsInfoRequest) Reset()                    { *m = G2GFriendsInfoRequest{} }
func (m *G2GFriendsInfoRequest) String() string            { return proto.CompactTextString(m) }
func (*G2GFriendsInfoRequest) ProtoMessage()               {}
func (*G2GFriendsInfoRequest) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{14} }

// 好友信息
type FriendInfo struct {
	PlayerId         int32  `protobuf:"varint,1,opt,name=PlayerId" json:"PlayerId,omitempty"`
	Name             string `protobuf:"bytes,2,opt,name=Name" json:"Name,omitempty"`
	Head             int32  `protobuf:"varint,3,opt,name=Head" json:"Head,omitempty"`
	Level            int32  `protobuf:"varint,4,opt,name=Level" json:"Level,omitempty"`
	VipLevel         int32  `protobuf:"varint,5,opt,name=VipLevel" json:"VipLevel,omitempty"`
	LastLogin        int32  `protobuf:"varint,6,opt,name=LastLogin" json:"LastLogin,omitempty"`
	FriendPoints     int32  `protobuf:"varint,7,opt,name=FriendPoints" json:"FriendPoints,omitempty"`
	LeftGiveSeconds  int32  `protobuf:"varint,8,opt,name=LeftGiveSeconds" json:"LeftGiveSeconds,omitempty"`
	UnreadMessageNum int32  `protobuf:"varint,9,opt,name=UnreadMessageNum" json:"UnreadMessageNum,omitempty"`
	Zan              int32  `protobuf:"varint,10,opt,name=Zan" json:"Zan,omitempty"`
	IsZanToday       bool   `protobuf:"varint,11,opt,name=IsZanToday" json:"IsZanToday,omitempty"`
	IsOnline         bool   `protobuf:"varint,12,opt,name=IsOnline" json:"IsOnline,omitempty"`
}

func (m *FriendInfo) Reset()                    { *m = FriendInfo{} }
func (m *FriendInfo) String() string            { return proto.CompactTextString(m) }
func (*FriendInfo) ProtoMessage()               {}
func (*FriendInfo) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{15} }

func (m *FriendInfo) GetPlayerId() int32 {
	if m != nil {
		return m.PlayerId
	}
	return 0
}

func (m *FriendInfo) GetName() string {
	if m != nil {
		return m.Name
	}
	return ""
}

func (m *FriendInfo) GetHead() int32 {
	if m != nil {
		return m.Head
	}
	return 0
}

func (m *FriendInfo) GetLevel() int32 {
	if m != nil {
		return m.Level
	}
	return 0
}

func (m *FriendInfo) GetVipLevel() int32 {
	if m != nil {
		return m.VipLevel
	}
	return 0
}

func (m *FriendInfo) GetLastLogin() int32 {
	if m != nil {
		return m.LastLogin
	}
	return 0
}

func (m *FriendInfo) GetFriendPoints() int32 {
	if m != nil {
		return m.FriendPoints
	}
	return 0
}

func (m *FriendInfo) GetLeftGiveSeconds() int32 {
	if m != nil {
		return m.LeftGiveSeconds
	}
	return 0
}

func (m *FriendInfo) GetUnreadMessageNum() int32 {
	if m != nil {
		return m.UnreadMessageNum
	}
	return 0
}

func (m *FriendInfo) GetZan() int32 {
	if m != nil {
		return m.Zan
	}
	return 0
}

func (m *FriendInfo) GetIsZanToday() bool {
	if m != nil {
		return m.IsZanToday
	}
	return false
}

func (m *FriendInfo) GetIsOnline() bool {
	if m != nil {
		return m.IsOnline
	}
	return false
}

// 多个好友信息返回
type G2GFriendsInfoResponse struct {
	Friends []*FriendInfo `protobuf:"bytes,1,rep,name=Friends" json:"Friends,omitempty"`
}

func (m *G2GFriendsInfoResponse) Reset()                    { *m = G2GFriendsInfoResponse{} }
func (m *G2GFriendsInfoResponse) String() string            { return proto.CompactTextString(m) }
func (*G2GFriendsInfoResponse) ProtoMessage()               {}
func (*G2GFriendsInfoResponse) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{16} }

func (m *G2GFriendsInfoResponse) GetFriends() []*FriendInfo {
	if m != nil {
		return m.Friends
	}
	return nil
}

// 刷新赠送好友点数
type G2GFriendsRefreshGivePointsRequest struct {
}

func (m *G2GFriendsRefreshGivePointsRequest) Reset()         { *m = G2GFriendsRefreshGivePointsRequest{} }
func (m *G2GFriendsRefreshGivePointsRequest) String() string { return proto.CompactTextString(m) }
func (*G2GFriendsRefreshGivePointsRequest) ProtoMessage()    {}
func (*G2GFriendsRefreshGivePointsRequest) Descriptor() ([]byte, []int) {
	return fileDescriptor0, []int{17}
}

type G2GFriendsRefreshGivePointsResponse struct {
}

func (m *G2GFriendsRefreshGivePointsResponse) Reset()         { *m = G2GFriendsRefreshGivePointsResponse{} }
func (m *G2GFriendsRefreshGivePointsResponse) String() string { return proto.CompactTextString(m) }
func (*G2GFriendsRefreshGivePointsResponse) ProtoMessage()    {}
func (*G2GFriendsRefreshGivePointsResponse) Descriptor() ([]byte, []int) {
	return fileDescriptor0, []int{18}
}

// 赠送好友点数
type G2GFriendGivePointsRequest struct {
}

func (m *G2GFriendGivePointsRequest) Reset()                    { *m = G2GFriendGivePointsRequest{} }
func (m *G2GFriendGivePointsRequest) String() string            { return proto.CompactTextString(m) }
func (*G2GFriendGivePointsRequest) ProtoMessage()               {}
func (*G2GFriendGivePointsRequest) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{19} }

type G2GFriendGivePointsResponse struct {
	LastSave      int32 `protobuf:"varint,1,opt,name=LastSave" json:"LastSave,omitempty"`
	RemainSeconds int32 `protobuf:"varint,2,opt,name=RemainSeconds" json:"RemainSeconds,omitempty"`
}

func (m *G2GFriendGivePointsResponse) Reset()                    { *m = G2GFriendGivePointsResponse{} }
func (m *G2GFriendGivePointsResponse) String() string            { return proto.CompactTextString(m) }
func (*G2GFriendGivePointsResponse) ProtoMessage()               {}
func (*G2GFriendGivePointsResponse) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{20} }

func (m *G2GFriendGivePointsResponse) GetLastSave() int32 {
	if m != nil {
		return m.LastSave
	}
	return 0
}

func (m *G2GFriendGivePointsResponse) GetRemainSeconds() int32 {
	if m != nil {
		return m.RemainSeconds
	}
	return 0
}

// 好友聊天
type G2GFriendChatRequest struct {
	Content []byte `protobuf:"bytes,1,opt,name=Content,proto3" json:"Content,omitempty"`
}

func (m *G2GFriendChatRequest) Reset()                    { *m = G2GFriendChatRequest{} }
func (m *G2GFriendChatRequest) String() string            { return proto.CompactTextString(m) }
func (*G2GFriendChatRequest) ProtoMessage()               {}
func (*G2GFriendChatRequest) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{21} }

func (m *G2GFriendChatRequest) GetContent() []byte {
	if m != nil {
		return m.Content
	}
	return nil
}

type G2GFriendChatResponse struct {
}

func (m *G2GFriendChatResponse) Reset()                    { *m = G2GFriendChatResponse{} }
func (m *G2GFriendChatResponse) String() string            { return proto.CompactTextString(m) }
func (*G2GFriendChatResponse) ProtoMessage()               {}
func (*G2GFriendChatResponse) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{22} }

func init() {
	proto.RegisterType((*G2GPlayerInfoRequest)(nil), "msg.rpc_message.G2GPlayerInfoRequest")
	proto.RegisterType((*G2GPlayerInfoResponse)(nil), "msg.rpc_message.G2GPlayerInfoResponse")
	proto.RegisterType((*G2GPlayerInfoNotify)(nil), "msg.rpc_message.G2GPlayerInfoNotify")
	proto.RegisterType((*G2GPlayerBattleInfoRequest)(nil), "msg.rpc_message.G2GPlayerBattleInfoRequest")
	proto.RegisterType((*G2GPlayerBattleInfoResponse)(nil), "msg.rpc_message.G2GPlayerBattleInfoResponse")
	proto.RegisterType((*G2GPlayerMultiInfoRequest)(nil), "msg.rpc_message.G2GPlayerMultiInfoRequest")
	proto.RegisterType((*PlayerInfo)(nil), "msg.rpc_message.PlayerInfo")
	proto.RegisterType((*G2GPlayerMultiInfoResponse)(nil), "msg.rpc_message.G2GPlayerMultiInfoResponse")
	proto.RegisterType((*G2GFriendAskRequest)(nil), "msg.rpc_message.G2GFriendAskRequest")
	proto.RegisterType((*G2GFriendAskResponse)(nil), "msg.rpc_message.G2GFriendAskResponse")
	proto.RegisterType((*G2GFriendAgreeRequest)(nil), "msg.rpc_message.G2GFriendAgreeRequest")
	proto.RegisterType((*G2GFriendAgreeResponse)(nil), "msg.rpc_message.G2GFriendAgreeResponse")
	proto.RegisterType((*G2GFriendRemoveRequest)(nil), "msg.rpc_message.G2GFriendRemoveRequest")
	proto.RegisterType((*G2GFriendRemoveResponse)(nil), "msg.rpc_message.G2GFriendRemoveResponse")
	proto.RegisterType((*G2GFriendsInfoRequest)(nil), "msg.rpc_message.G2GFriendsInfoRequest")
	proto.RegisterType((*FriendInfo)(nil), "msg.rpc_message.FriendInfo")
	proto.RegisterType((*G2GFriendsInfoResponse)(nil), "msg.rpc_message.G2GFriendsInfoResponse")
	proto.RegisterType((*G2GFriendsRefreshGivePointsRequest)(nil), "msg.rpc_message.G2GFriendsRefreshGivePointsRequest")
	proto.RegisterType((*G2GFriendsRefreshGivePointsResponse)(nil), "msg.rpc_message.G2GFriendsRefreshGivePointsResponse")
	proto.RegisterType((*G2GFriendGivePointsRequest)(nil), "msg.rpc_message.G2GFriendGivePointsRequest")
	proto.RegisterType((*G2GFriendGivePointsResponse)(nil), "msg.rpc_message.G2GFriendGivePointsResponse")
	proto.RegisterType((*G2GFriendChatRequest)(nil), "msg.rpc_message.G2GFriendChatRequest")
	proto.RegisterType((*G2GFriendChatResponse)(nil), "msg.rpc_message.G2GFriendChatResponse")
	proto.RegisterEnum("msg.rpc_message.MSGID", MSGID_name, MSGID_value)
}

func init() { proto.RegisterFile("rpc_message.proto", fileDescriptor0) }

var fileDescriptor0 = []byte{
	// 905 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x09, 0x6e, 0x88, 0x02, 0xff, 0xb4, 0x56, 0xcb, 0x72, 0xe3, 0x44,
	0x14, 0xc5, 0xb1, 0x9d, 0x38, 0x37, 0x81, 0x31, 0xcd, 0x3c, 0x7a, 0xf2, 0x22, 0x68, 0x06, 0x70,
	0x0d, 0x54, 0xa0, 0x86, 0x62, 0xc9, 0xc2, 0x93, 0x91, 0x1d, 0x15, 0xb6, 0x6c, 0x24, 0x39, 0x55,
	0x33, 0x2c, 0x5c, 0x22, 0x6e, 0x3b, 0x62, 0xec, 0x96, 0xa3, 0x96, 0x4d, 0xe5, 0x0f, 0xf8, 0x1f,
	0x7e, 0x8a, 0x0d, 0xff, 0x40, 0xb9, 0x1f, 0x52, 0xab, 0x95, 0x84, 0x15, 0x3b, 0xdf, 0x73, 0xee,
	0xb9, 0xef, 0x56, 0x02, 0x9f, 0x26, 0xcb, 0xab, 0xf1, 0x82, 0x30, 0x16, 0xce, 0xc8, 0xd9, 0x32,
	0x89, 0xd3, 0x18, 0x3d, 0x5a, 0xb0, 0xd9, 0x99, 0x06, 0x5b, 0x4f, 0xe1, 0x71, 0xf7, 0x75, 0x77,
	0x38, 0x0f, 0x6f, 0x49, 0xe2, 0xd0, 0x69, 0xec, 0x91, 0x9b, 0x15, 0x61, 0xa9, 0xf5, 0x07, 0x3c,
	0x31, 0x70, 0xb6, 0x8c, 0x29, 0x23, 0xe8, 0x00, 0x1a, 0x23, 0x1a, 0xdd, 0xac, 0x88, 0x33, 0xc1,
	0x5b, 0xa7, 0x95, 0xd6, 0xae, 0x97, 0xd9, 0x08, 0xc3, 0x4e, 0xfb, 0xea, 0x2a, 0x5e, 0xd1, 0x14,
	0x57, 0x39, 0xa5, 0x4c, 0xf4, 0x18, 0xea, 0x3d, 0xb2, 0x26, 0x73, 0x5c, 0x3b, 0xad, 0xb4, 0xea,
	0x9e, 0x30, 0x10, 0x82, 0xda, 0x05, 0x09, 0x27, 0xb8, 0xce, 0x41, 0xfe, 0xdb, 0x7a, 0x02, 0x9f,
	0x15, 0x12, 0xbb, 0x71, 0x1a, 0x4d, 0x6f, 0xad, 0x23, 0x38, 0xc8, 0xe0, 0x37, 0x61, 0x9a, 0xce,
	0x89, 0x5e, 0xed, 0x31, 0x1c, 0xde, 0xc9, 0x8a, 0x9a, 0xad, 0x43, 0x78, 0x9e, 0xd1, 0xfd, 0xd5,
	0x3c, 0x8d, 0x74, 0xed, 0x9f, 0x15, 0x80, 0x3c, 0xdd, 0xa6, 0x3f, 0x69, 0x4d, 0x70, 0x85, 0xd7,
	0x95, 0xd9, 0xff, 0x7b, 0xef, 0xbf, 0x6a, 0x4d, 0x6a, 0x75, 0xca, 0xc9, 0xff, 0x04, 0x7b, 0x79,
	0x9d, 0x0c, 0x57, 0x4e, 0xab, 0xad, 0xbd, 0xd7, 0x87, 0x67, 0xc6, 0x46, 0xcf, 0xb4, 0x9d, 0xe9,
	0xfe, 0xd6, 0x5f, 0x15, 0x3e, 0xd9, 0x4e, 0x12, 0x11, 0x3a, 0x69, 0xb3, 0x0f, 0xb2, 0x7f, 0x64,
	0xc1, 0x7e, 0x27, 0x89, 0x17, 0x46, 0xd3, 0x05, 0x0c, 0x7d, 0x05, 0x9f, 0xe4, 0xb6, 0x1b, 0x2e,
	0x88, 0x6c, 0xdf, 0x40, 0x51, 0x0b, 0x1e, 0xe5, 0x88, 0x68, 0xba, 0xca, 0xc3, 0x99, 0x70, 0x31,
	0x22, 0x1f, 0x84, 0x98, 0x8e, 0x81, 0xca, 0xfb, 0xd4, 0x8a, 0x96, 0x2b, 0xbd, 0xe0, 0xf7, 0x29,
	0xf1, 0x59, 0x42, 0x88, 0x6a, 0xe7, 0x3b, 0xa8, 0x6d, 0xfa, 0xe5, 0x6d, 0xdc, 0x35, 0x1e, 0x21,
	0xe1, 0xe3, 0xe1, 0x8e, 0x16, 0x86, 0xa7, 0x66, 0x24, 0x99, 0x43, 0x67, 0x3c, 0xb2, 0x88, 0xd7,
	0x2a, 0x89, 0xf5, 0x1c, 0x9e, 0x95, 0x18, 0x29, 0x7a, 0xa6, 0x15, 0xc6, 0xf4, 0x3b, 0xfb, 0x7b,
	0x0b, 0x20, 0x4f, 0xfe, 0xe0, 0x9d, 0x21, 0xa8, 0x69, 0x43, 0xe6, 0xbf, 0xb3, 0x7b, 0xa9, 0xe6,
	0xf7, 0x72, 0xcf, 0x65, 0x1d, 0x40, 0xe3, 0x32, 0x5a, 0x0a, 0x42, 0x5c, 0x57, 0x66, 0xa3, 0x23,
	0xd8, 0xed, 0x85, 0x2c, 0xed, 0xc5, 0xb3, 0x88, 0xe2, 0x6d, 0x4e, 0xe6, 0x80, 0x38, 0x85, 0x4d,
	0x85, 0xc3, 0x38, 0xa2, 0x29, 0xc3, 0x3b, 0xea, 0x14, 0x72, 0x6c, 0xb3, 0xe2, 0x1e, 0x99, 0xa6,
	0xdd, 0x68, 0x4d, 0x7c, 0x72, 0x15, 0xd3, 0x09, 0xc3, 0x0d, 0xb1, 0x62, 0x03, 0x46, 0xaf, 0xa0,
	0x39, 0xa2, 0x09, 0x09, 0x27, 0x7d, 0x31, 0x7a, 0x77, 0xb5, 0xc0, 0xbb, 0xdc, 0xb5, 0x84, 0xa3,
	0x26, 0x54, 0xdf, 0x87, 0x14, 0x03, 0xa7, 0x37, 0x3f, 0xd1, 0x09, 0x80, 0xc3, 0xde, 0x87, 0x34,
	0x88, 0x27, 0xe1, 0x2d, 0xde, 0x3b, 0xad, 0xb4, 0x1a, 0x9e, 0x86, 0x6c, 0xba, 0x74, 0xd8, 0x80,
	0xce, 0x23, 0x4a, 0xf0, 0x3e, 0x67, 0x33, 0xdb, 0x1a, 0x68, 0x8b, 0x63, 0x85, 0x37, 0xf4, 0x23,
	0xec, 0x48, 0xf8, 0xde, 0xf7, 0xa3, 0x1d, 0x88, 0xf2, 0xb5, 0x5e, 0x82, 0x95, 0x07, 0xf4, 0xc8,
	0x34, 0x21, 0xec, 0x7a, 0xd3, 0xaa, 0x98, 0x89, 0xda, 0xf0, 0x97, 0xf0, 0xe2, 0x41, 0x2f, 0x79,
	0x21, 0xe2, 0x53, 0x26, 0xdc, 0xca, 0x41, 0xc6, 0xfc, 0x53, 0x56, 0x66, 0xf3, 0xcf, 0xef, 0x66,
	0x5f, 0x7e, 0xb8, 0x26, 0xea, 0x6c, 0x94, 0x8d, 0x5e, 0xc2, 0xc7, 0x1e, 0x59, 0x84, 0x11, 0x55,
	0x8b, 0xd9, 0xe2, 0x0e, 0x45, 0xd0, 0xfa, 0x5e, 0x7b, 0x51, 0xe7, 0xd7, 0x61, 0xaa, 0x1e, 0x0e,
	0x86, 0x9d, 0xf3, 0x98, 0xa6, 0x84, 0xa6, 0x3c, 0xf0, 0xbe, 0xa7, 0xcc, 0xc2, 0x49, 0x0b, 0x85,
	0x28, 0xe6, 0xd5, 0x3f, 0xdb, 0x50, 0xef, 0xfb, 0x5d, 0xe7, 0x2d, 0x6a, 0x40, 0xcd, 0x1d, 0xb8,
	0x76, 0xf3, 0x23, 0x74, 0xc8, 0x9f, 0xc6, 0x78, 0xd8, 0x6b, 0xbf, 0xb3, 0xbd, 0xb1, 0xe3, 0x76,
	0x06, 0x63, 0xcf, 0xfe, 0x65, 0x64, 0xfb, 0x41, 0xb3, 0x82, 0x8e, 0x00, 0x97, 0x49, 0x7f, 0x38,
	0x70, 0x7d, 0xbb, 0xb9, 0x85, 0x0e, 0xf8, 0xda, 0x0a, 0xac, 0x3b, 0x08, 0x9c, 0xce, 0xbb, 0x66,
	0x15, 0x59, 0x70, 0xa2, 0x71, 0x6f, 0xda, 0x41, 0xd0, 0xb3, 0x8b, 0xd1, 0x6b, 0xe8, 0x05, 0x7c,
	0x7e, 0xaf, 0x8f, 0x4c, 0x52, 0x47, 0x5f, 0xc0, 0xb1, 0xe6, 0xd4, 0x1f, 0xf5, 0x02, 0xa7, 0x18,
	0x07, 0x8c, 0x5c, 0x05, 0x17, 0x19, 0x66, 0x4f, 0x75, 0xd2, 0xf1, 0x1c, 0xdb, 0x7d, 0xeb, 0x17,
	0x23, 0x4c, 0xd0, 0x31, 0xff, 0x83, 0x63, 0xb2, 0x52, 0x4c, 0x54, 0xa3, 0x82, 0x1e, 0xb7, 0xfd,
	0x9f, 0x33, 0xe9, 0x54, 0xcd, 0xaf, 0xc0, 0x49, 0xe1, 0xac, 0x98, 0x75, 0xdc, 0xee, 0x7a, 0xb6,
	0x9d, 0x49, 0xaf, 0x8b, 0x59, 0x33, 0x56, 0x8a, 0x23, 0x83, 0xf6, 0xec, 0xce, 0xc8, 0xcf, 0xd5,
	0xbf, 0xa3, 0x13, 0x7e, 0x96, 0x25, 0x5a, 0xca, 0x3f, 0x94, 0xe4, 0xfd, 0xc1, 0x65, 0x2e, 0x9f,
	0x97, 0xe4, 0x92, 0x96, 0xf2, 0x85, 0x51, 0xba, 0xe4, 0xe5, 0x7a, 0xa9, 0x1a, 0xb9, 0x64, 0xbb,
	0xce, 0xa5, 0x3d, 0x1e, 0x0e, 0x1c, 0x37, 0xf0, 0xb3, 0x0c, 0xb1, 0x5a, 0xef, 0x9d, 0x3e, 0x32,
	0xcd, 0x52, 0xad, 0x57, 0x39, 0xd9, 0x81, 0x19, 0xe7, 0xc6, 0xcc, 0xa5, 0xbb, 0xc8, 0x30, 0x09,
	0xfa, 0x06, 0xbe, 0xd6, 0x17, 0xe8, 0xd9, 0x1d, 0xcf, 0xf6, 0x2f, 0xee, 0x2c, 0x8c, 0xa1, 0x6f,
	0xa1, 0xf5, 0xdf, 0xce, 0x32, 0x74, 0x6a, 0x2c, 0xf8, 0xfc, 0xa2, 0x1d, 0x64, 0xa1, 0x56, 0xc6,
	0x94, 0x24, 0x29, 0xa5, 0xeb, 0xdf, 0xb6, 0xf9, 0x3f, 0x71, 0x3f, 0xfc, 0x1b, 0x00, 0x00, 0xff,
	0xff, 0xce, 0x21, 0x9d, 0xc7, 0xd9, 0x09, 0x00, 0x00,
}
