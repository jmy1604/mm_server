package main

import (
	"math/rand"
	"mm_server/libs/log"
	"mm_server/libs/utils"
	"mm_server/proto/gen_go/client_message"
	"mm_server/proto/gen_go/rpc_message"
	"mm_server/src/rpc_proto"
	"mm_server/src/tables"
	"sync"
	"sync/atomic"
	"time"

	"github.com/golang/protobuf/proto"
)

const FRIEND_UNREAD_MESSAGE_MAX_NUM int = 200
const FRIEND_MESSAGE_MAX_LENGTH int = 200

const MAX_FRIEND_RECOMMEND_PLAYER_NUM int32 = 10000

type FriendRecommendMgr struct {
	player_ids    map[int32]int32
	players_array []int32
	locker        *sync.RWMutex
	add_chan      chan int32
	to_end        int32
}

var friend_recommend_mgr FriendRecommendMgr

func (this *FriendRecommendMgr) Init() {
	this.player_ids = make(map[int32]int32)
	this.players_array = make([]int32, MAX_FRIEND_RECOMMEND_PLAYER_NUM)
	this.locker = &sync.RWMutex{}
	this.add_chan = make(chan int32, 10000)
	this.to_end = 0
}

func (this *FriendRecommendMgr) AddPlayer(player_id int32) {
	this.add_chan <- player_id
	log.Debug("Friend Recommend Manager to add player[%v]", player_id)
}

func (this *FriendRecommendMgr) CheckAndAddPlayer(player_id int32) bool {
	p := player_mgr.GetPlayerById(player_id)
	if p == nil {
		return false
	}

	if _, o := this.player_ids[player_id]; o {
		//log.Warn("Player[%v] already added Friend Recommend mgr", player_id)
		return false
	}

	var add_pos int32
	num := int32(len(this.player_ids))
	if num >= MAX_FRIEND_RECOMMEND_PLAYER_NUM {
		add_pos = rand.Int31n(num)
		// 删掉一个随机位置的
		delete(this.player_ids, this.players_array[add_pos])
		this.players_array[add_pos] = 0
	} else {
		add_pos = num
	}

	now_time := int32(time.Now().Unix())
	if now_time-p.db.Info.GetLastLogout() > 24*3600*2 && atomic.LoadInt32(&p.is_login) == 0 {
		return false
	}

	if p.db.Friends.NumAll() >= global_config.FriendMaxNum {
		return false
	}

	this.player_ids[player_id] = add_pos
	this.players_array[add_pos] = player_id

	//log.Debug("Friend Recommend Manager add player[%v], total count[%v], player_ids: %v, players_array: %v", player_id, len(this.player_ids), this.player_ids, this.players_array[:len(this.player_ids)])

	return true
}

func (this *FriendRecommendMgr) Run() {
	defer func() {
		if err := recover(); err != nil {
			log.Stack(err)
		}
	}()

	var last_check_remove_time int32
	for {
		if atomic.LoadInt32(&this.to_end) > 0 {
			break
		}
		// 处理操作队列
		is_break := false
		for !is_break {
			select {
			case player_id, ok := <-this.add_chan:
				{
					if !ok {
						log.Error("conn timer wheel op chan receive invalid !!!!!")
						return
					}
					this.CheckAndAddPlayer(player_id)
				}
			default:
				{
					is_break = true
				}
			}
		}

		now_time := int32(time.Now().Unix())
		if now_time-last_check_remove_time >= 60*10 {
			this.locker.Lock()
			player_num := len(this.player_ids)
			for i := 0; i < player_num; i++ {
				p := player_mgr.GetPlayerById(this.players_array[i])
				if p == nil {
					continue
				}
				if (now_time-p.db.Info.GetLastLogout() >= 2*24*3600 && atomic.LoadInt32(&p.is_login) == 0) || p.db.Friends.NumAll() >= global_config.FriendMaxNum {
					delete(this.player_ids, this.players_array[i])
					this.players_array[i] = this.players_array[player_num-1]
					player_num -= 1
				}
			}
			this.locker.Unlock()
			last_check_remove_time = now_time
		}

		time.Sleep(time.Second * 1)
	}
}

func (this *FriendRecommendMgr) Random(player_id int32) (ids []int32) {
	player := player_mgr.GetPlayerById(player_id)
	if player == nil {
		return
	}

	this.locker.RLock()
	defer this.locker.RUnlock()

	cnt := int32(len(this.player_ids))
	if cnt == 0 {
		return
	}

	if cnt > global_config.FriendRecommendNum {
		cnt = global_config.FriendRecommendNum
	}

	rand.Seed(time.Now().Unix() + time.Now().UnixNano())
	for i := int32(0); i < cnt; i++ {
		var pid int32
		r := rand.Int31n(int32(len(this.player_ids)))
		sr := r
		for {
			pid = this.players_array[sr]
			p := player_mgr.GetPlayerById(pid)
			has := false
			if pid == player_id || player.db.Friends.HasIndex(pid) || (p != nil && p.db.FriendAsks.HasIndex(player_id)) || player.db.FriendAsks.HasIndex(pid) {
				has = true
			} else {
				if ids != nil {
					for n := 0; n < len(ids); n++ {
						if ids[n] == pid {
							has = true
							break
						}
					}
				}
			}
			if !has {
				break
			}
			sr = (sr + 1) % int32(len(this.player_ids))
			if sr == r {
				log.Info("Friend Recommend Mgr player count[%v] not enough to random a player to recommend", len(this.player_ids))
				return
			}
		}
		if pid <= 0 {
			break
		}
		ids = append(ids, pid)
	}
	return ids
}

// ----------------------------------------------------------------------------

func send_search_player_msg(p *Player, players_info []*rpc_proto.SearchPlayerInfo) {
	var results []*msg_client_message.FriendInfo
	if players_info == nil || len(players_info) == 0 {
		results = make([]*msg_client_message.FriendInfo, 0)
	} else {
		results = make([]*msg_client_message.FriendInfo, len(players_info))
		for i := 0; i < len(players_info); i++ {
			r := &msg_client_message.FriendInfo{
				PlayerId:  players_info[i].Id,
				Name:      players_info[i].Nick,
				Head:      players_info[i].Head,
				Level:     players_info[i].Level,
				VipLevel:  players_info[i].VipLevel,
				LastLogin: players_info[i].LastLogin,
			}
			results[i] = r
		}

	}
	response := msg_client_message.S2CFriendSearchResult{}
	response.Result = results
	p.Send(uint16(msg_client_message.S2CFriendSearchResult_ProtoID), &response)
}

func (this *dbPlayerFriendColumn) GetFriendInfoMsg(friend_id int32) *msg_client_message.FriendInfo {
	this.m_row.m_lock.UnSafeRLock("dbPlayerFriendColumn.GetMsgFriendInfo")
	defer this.m_row.m_lock.UnSafeRUnlock()

	d := this.m_data[friend_id]
	if d == nil {
		return nil
	}

	return &msg_client_message.FriendInfo{
		PlayerId:  d.FriendId,
		Name:      d.FriendName,
		Head:      d.Head,
		Level:     d.Level,
		VipLevel:  d.VipLevel,
		LastLogin: d.LastLogin,
	}
}

func (this *dbPlayerFriendChatUnreadIdColumn) CheckUnreadNumFull(friend_id int32) (full bool, next_id int32) {
	this.m_row.m_lock.UnSafeLock("dbPlayerFriendChatUnreadIdColumn.CheckUnreadNumFull")
	defer this.m_row.m_lock.UnSafeUnlock()

	d := this.m_data[friend_id]
	if d == nil {
		next_id = 1
		this.m_changed = true
	} else if len(d.MessageIds) < FRIEND_UNREAD_MESSAGE_MAX_NUM {
		if d.CurrMessageId >= int32(FRIEND_UNREAD_MESSAGE_MAX_NUM) {
			d.CurrMessageId = 1
		} else {
			d.CurrMessageId += 1
		}
		next_id = d.CurrMessageId
		this.m_changed = true
	} else {
		full = true
	}

	return
}

func (this *dbPlayerFriendChatUnreadIdColumn) AddNewMessageId(friend_id, message_id int32) int32 {
	this.m_row.m_lock.UnSafeLock("dbPlayerFriendChatUnreadIdColumn.AddNewMessageId")
	defer this.m_row.m_lock.UnSafeUnlock()

	d := this.m_data[friend_id]
	if d == nil {
		this.m_data[friend_id] = &dbPlayerFriendChatUnreadIdData{
			FriendId:   friend_id,
			MessageIds: []int32{message_id},
		}
	} else {
		if len(d.MessageIds) >= FRIEND_UNREAD_MESSAGE_MAX_NUM {
			return int32(msg_client_message.E_ERR_FRIEND_MESSAGE_NUM_MAX)
		}
		d.MessageIds = append(d.MessageIds, message_id)
	}

	this.m_changed = true

	return 1
}

func (this *dbPlayerFriendChatUnreadIdColumn) GetUnreadMessageNum(friend_id int32) int32 {
	this.m_row.m_lock.UnSafeRLock("dbPlayerFriendChatUnreadIdColumn.GetUnreadMessageNum")
	defer this.m_row.m_lock.UnSafeRUnlock()

	d := this.m_data[friend_id]
	if d == nil {
		return 0
	}

	return int32(len(d.MessageIds))
}

func (this *dbPlayerFriendChatUnreadIdColumn) ConfirmUnreadIds(friend_id, unread_num int32) (res int32, remove_ids []int32) {
	this.m_row.m_lock.UnSafeLock("dbPlayerFriendChatUnreadIdColumn.ConfirmUnreadIds")
	defer this.m_row.m_lock.UnSafeUnlock()

	d := this.m_data[friend_id]
	if d == nil {
		log.Error("Player[%v] no unread message from friend[%v]", this.m_row.m_PlayerId, friend_id)
		res = int32(msg_client_message.E_ERR_FRIEND_NO_UNREAD_MESSAGE)
		return
	}

	if unread_num == 0 || len(d.MessageIds) <= int(unread_num) {
		remove_ids = d.MessageIds
		d.MessageIds = make([]int32, 0)
	} else {
		remove_ids = d.MessageIds[:unread_num]
		d.MessageIds = d.MessageIds[unread_num:]
	}

	this.m_changed = true

	res = 1
	return
}

func (this *dbPlayerFriendChatUnreadMessageColumn) RemoveMessages(friend_id int32, message_ids []int32) {
	if message_ids == nil || len(message_ids) == 0 {
		return
	}

	this.m_row.m_lock.UnSafeLock("dbPlayerFriendChatUnreadMessageColumn.RemoveMessages")
	defer this.m_row.m_lock.UnSafeUnlock()

	for i := 0; i < len(message_ids); i++ {
		player_message_id := utils.Int64From2Int32(friend_id, message_ids[i])
		delete(this.m_data, player_message_id)
	}
	this.m_changed = true
}

func (this *Player) search_friend(key string) int32 {
	result := this.rpc_search_friends(key)
	if result == nil {
		log.Error("查找玩家[%v]数据失败", key)
		return -1
	}

	send_search_player_msg(this, result.Players)

	log.Trace("Player[%v] searched friend %v with key[%v]", this.Id, result.Players, key)

	return 1
}

func remote_add_friend_by_id(from_player_id, to_player_id int32) (resp *msg_rpc_message.G2GFriendAskResponse, err_code int32) {
	var req msg_rpc_message.G2GFriendAskRequest
	resp = &msg_rpc_message.G2GFriendAskResponse{}
	err_code = RemoteGetUsePB(from_player_id, rpc_proto.OBJECT_TYPE_PLAYER, to_player_id, int32(msg_rpc_message.MSGID_G2G_FRIEND_ASK_REQUEST), &req, resp)
	return
}

func remote_add_friend_by_id_response(from_player_id, to_player_id int32, req_data []byte) (resp_data []byte, err_code int32) {
	var req msg_rpc_message.G2GFriendAskRequest
	err := _unmarshal_msg(req_data, &req)
	if err != nil {
		err_code = -1
		return
	}
	to_player := player_mgr.GetPlayerById(to_player_id)
	if to_player == nil {
		return
	}
	err_code = to_player.db.FriendReqs.CheckAndAdd(from_player_id)
	if err_code >= 0 {
		var response = msg_rpc_message.G2GFriendAskResponse{}
		var err error
		resp_data, err = _marshal_msg(&response)
		if err != nil {
			err_code = -1
			return
		}
	}
	to_player.notify_friend_add(req.FromPlayerId, req.FromPlayerName, req.FromPlayerLevel, req.FromPlayerHead)
	return
}

func (this *Player) notify_friend_add(player_id int32, player_name string, player_level int32, player_head int32) {
	notify := &msg_client_message.S2CFriendReqNotify{
		Req: &msg_client_message.FriendReq{
			PlayerId: player_id,
			Name:     player_name,
			Level:    player_level,
			Head:     player_head,
		},
	}
	this.Send(uint16(msg_client_message.S2CFriendReqNotify_ProtoID), notify)
}

func (this *Player) add_friend_by_id(id int32) int32 {
	if id == this.Id {
		log.Error("Cant add self to friend!!!")
		return -1
	}

	// 已是好友
	if this.db.Friends.HasIndex(id) {
		log.Error("!!! Player[%v] already added player[%v] to friend", this.Id, id)
		return int32(msg_client_message.E_ERR_FRIEND_THE_PLAYER_ALREADY_FRIEND)
	}

	add_player := player_mgr.GetPlayerById(id)
	if add_player != nil {
		res := add_player.db.FriendReqs.CheckAndAdd(this.Id)
		if res < 0 {
			log.Error("!!! Player[%v] request add friend to other player[%v] already exist", this.Id, id)
			return res
		}
		add_player.notify_friend_add(this.Id, this.db.GetName(), this.db.GetLevel(), this.db.Info.GetHead())
	} else {
		// remote调用
		resp, err_code := remote_add_friend_by_id(this.Id, id)
		if resp == nil {
			log.Error("!!! Player[%v] request add friend to other player[%v] failed", this.Id, id)
			return -1
		}
		if err_code < 0 {
			log.Error("!!! Player %v add player %v to friend err %v", this.Id, id, err_code)
			return err_code
		}
	}

	response := &msg_client_message.S2CAddFriendResult{}
	response.PlayerId = id
	this.Send(uint16(msg_client_message.S2CAddFriendResult_ProtoID), response)

	log.Info("Player[%v] requested add friend[%v]", this.Id, id)

	return 1
}

func remote_agree_add_friend(from_player_id, to_player_id int32) (resp *msg_rpc_message.G2GFriendAgreeResponse, err_code int32) {
	p := player_mgr.GetPlayerById(from_player_id)
	if p == nil {
		err_code = -1
		return
	}
	var req = msg_rpc_message.G2GFriendAgreeRequest{
		Info: &msg_rpc_message.FriendInfo{
			PlayerId:  from_player_id,
			Name:      p.db.GetName(),
			Head:      p.db.Info.GetHead(),
			Level:     p.db.GetLevel(),
			VipLevel:  p.db.Info.GetVipLvl(),
			LastLogin: p.db.Info.GetLastLogin(),
			Zan:       p.db.Info.GetZan(),
		},
	}
	resp = &msg_rpc_message.G2GFriendAgreeResponse{}
	err_code = RemoteGetUsePB(from_player_id, rpc_proto.OBJECT_TYPE_PLAYER, to_player_id, int32(msg_rpc_message.MSGID_G2G_FRIEND_AGREE_REQUEST), &req, resp)
	return
}

func remote_agree_add_friend_response(from_player_id, to_player_id int32, req_data []byte) (resp_data []byte, err_code int32) {
	var req msg_rpc_message.G2GFriendAgreeRequest
	err := _unmarshal_msg(req_data, &req)
	if err != nil {
		err_code = -1
		return
	}
	to_player := player_mgr.GetPlayerById(to_player_id)
	if to_player == nil {
		return
	}

	if !to_player.db.Friends.HasIndex(from_player_id) {
		to_player.db.Friends.Add(&dbPlayerFriendData{
			FriendId: from_player_id,
		})
	}

	friend_info := &msg_client_message.FriendInfo{
		PlayerId:     req.Info.PlayerId,
		Name:         req.Info.Name,
		Head:         req.Info.Head,
		Level:        req.Info.Level,
		VipLevel:     req.Info.VipLevel,
		LastLogin:    req.Info.LastLogin,
		FriendPoints: req.Info.FriendPoints,
	}

	to_player.notify_friend_info(1, friend_info)

	var response = msg_rpc_message.G2GFriendAgreeResponse{}
	resp_data, err = _marshal_msg(&response)
	if err != nil {
		err_code = -1
		return
	}

	return
}

func (this *Player) notify_friend_info(state int32, friend_info *msg_client_message.FriendInfo) {
	notify := &msg_client_message.S2CFriendStateNotify{
		StateType: state,
		Info:      friend_info,
	}
	this.Send(uint16(msg_client_message.S2CFriendStateNotify_ProtoID), notify)
}

func (this *Player) agree_add_friend(id int32) int32 {
	// 该玩家已是好友
	if this.db.Friends.HasIndex(id) {
		log.Error("Player[%v] already have friend[%v]", this.Id, id)
		return int32(msg_client_message.E_ERR_FRIEND_THE_PLAYER_ALREADY_FRIEND)
	}

	var data dbPlayerFriendData
	agree_player := player_mgr.GetPlayerById(id)
	if agree_player == nil {
		resp, err_code := remote_agree_add_friend(this.Id, id)
		if resp == nil {
			log.Error("Player[%v] agree add friend with player[%v] failed", this.Id, id)
			return -1
		}

		if err_code < 0 {
			log.Error("Player %v agree player %v to friend err %v", this.Id, id, err_code)
			return err_code
		}

		// 加到自己的好友列表
		data.FriendId = id
	} else {
		// 加到对方的好友列表
		data.FriendId = this.Id
		agree_player.db.Friends.Add(&data)

		// 加到自己的好友列表
		data.FriendId = id
		data.LastLogin = agree_player.db.Info.GetLastLogin()
	}

	this.db.Friends.Add(&data)

	// request remove
	this.db.FriendReqs.Remove(id)

	response := &msg_client_message.S2CAgreeFriendResult{
		PlayerId: id,
	}
	this.Send(uint16(msg_client_message.S2CAgreeFriendResult_ProtoID), response)

	log.Debug("Player[%v] agree add friend request of player[%v][%v]", this.Id, id, data.FriendName)

	if agree_player != nil {
		friend_info := agree_player.get_friend_info(this.Id, time.Now())
		agree_player.notify_friend_info(1, friend_info)
	}

	return 1
}

func (this *Player) refuse_add_friend(player_id int32) int32 {
	name, o := this.db.FriendReqs.GetPlayerName(player_id)
	if !o {
		log.Error("Player[%v] not have player[%v] friend request", this.Id, player_id)
		return -1
	}

	this.db.FriendReqs.Remove(player_id)

	response := &msg_client_message.S2CRefuseFriendResult{}
	response.Name = name
	response.PlayerId = player_id
	this.Send(uint16(msg_client_message.S2CRefuseFriendResult_ProtoID), response)

	log.Debug("Player[%v] refuse add friend request of player[%v]", this.Id, player_id)

	return 1
}

func (this *Player) remove_friend_data(friend_id int32) {
	this.db.Friends.Remove(friend_id)
	// 防止频繁的删除添加好友刷友情点
	//this.db.FriendPoints.Remove(friend_id)
	message_ids, o := this.db.FriendChatUnreadIds.GetMessageIds(friend_id)
	if o {
		this.db.FriendChatUnreadMessages.RemoveMessages(friend_id, message_ids)
		this.db.FriendChatUnreadIds.Remove(friend_id)
	}

	this.Send(uint16(msg_client_message.S2CRemoveFriendResult_ProtoID), &msg_client_message.S2CRemoveFriendResult{
		PlayerId: friend_id,
	})
}

func remote_remove_friend(from_player_id, to_player_id int32) (resp *msg_rpc_message.G2GFriendRemoveResponse, err_code int32) {
	var req msg_rpc_message.G2GFriendRemoveRequest
	resp = &msg_rpc_message.G2GFriendRemoveResponse{}
	err_code = RemoteGetUsePB(from_player_id, rpc_proto.OBJECT_TYPE_PLAYER, to_player_id, int32(msg_rpc_message.MSGID_G2G_FRIEND_REMOVE_REQUEST), &req, resp)
	return
}

func remote_remove_friend_response(from_player_id, to_player_id int32, req_data []byte) (resp_data []byte, err_code int32) {
	to_player := player_mgr.GetPlayerById(to_player_id)
	if to_player == nil {
		return
	}

	to_player.remove_friend_data(from_player_id)

	var response msg_rpc_message.G2GFriendRemoveResponse
	var err error
	resp_data, err = _marshal_msg(&response)
	if err != nil {
		err_code = -1
		return
	}

	return
}

func (this *Player) remove_friend(player_id int32) int32 {
	if !this.db.Friends.HasIndex(player_id) {
		log.Error("Player[%v] have not friend[%v], remove failed", this.Id, player_id)
		return -1
	}

	p := player_mgr.GetPlayerById(player_id)
	if p == nil {
		_, err_code := remote_remove_friend(this.Id, player_id)
		if err_code < 0 {
			log.Error("Player[%v] remove friend[%v] err_code %v", this.Id, player_id, err_code)
			return int32(msg_client_message.E_ERR_FRIEND_REMOVE_FRIEND_FAILED)
		}
	} else {
		p.remove_friend_data(this.Id)
	}

	this.remove_friend_data(player_id)

	response := &msg_client_message.S2CRemoveFriendResult{}
	response.PlayerId = player_id
	this.Send(uint16(msg_client_message.S2CRemoveFriendResult_ProtoID), response)

	log.Debug("Player[%v] removed friend[%v]", this.Id, player_id)

	return 1
}

func (this *Player) refresh_friend_give_points(friend_id int32) bool {
	this.db.FriendPoints.SetIsTodayGive(friend_id, 0)
	return true
}

func remote_friends_refresh_give_points(from_player_id int32, to_player_ids []int32) (err_code int32) {
	var req msg_rpc_message.G2GFriendsRefreshGivePointsRequest

	req_data, err := _marshal_msg(&req)
	if err != nil {
		err_code = -1
		return
	}

	datas := rpc_proto.RpcCommonMultiGet(game_server.rpc_client, "G2G_CommonProc.MultiGet", from_player_id, rpc_proto.OBJECT_TYPE_PLAYER, to_player_ids, int32(msg_rpc_message.MSGID_G2G_FRIENDS_REFRESH_GIVE_POINTS_REQUEST), req_data)
	if datas == nil || len(datas) == 0 {
		log.Error("get multi players %v empty", to_player_ids)
		err_code = -1
		return
	}

	err_code = 1
	return
}

func remote_friends_refresh_give_points_response(from_player_id int32, to_player_ids []int32, req_data []byte) (resp_data []byte, err_code int32) {
	for i := 0; i < len(to_player_ids); i++ {
		id := to_player_ids[i]
		player := player_mgr.GetPlayerById(id)
		if player == nil {
			continue
		}
		player.refresh_friend_give_points(from_player_id)
	}

	err_code = 1
	return
}

func (this *Player) check_friends_give_points_refresh() (remain_seconds int32) {
	friends := this.db.Friends.GetAllIndex()
	if friends == nil || len(friends) <= 0 {
		return
	}

	remain_seconds = utils.GetRemainSeconds2NextDayTime(this.db.FriendRelative.GetLastRefreshTime(), global_config.FriendGivePointsRefreshTime)

	if remain_seconds <= 0 {
		idx := SplitLocalAndRemotePlayers(friends)
		if idx >= 0 {
			for i := int32(0); i < idx; i++ {
				friend := player_mgr.GetPlayerById(friends[i])
				if friend != nil {
					friend.refresh_friend_give_points(this.Id)
				}
			}
		}

		if int(idx) > len(friends)+1 {
			err_code := remote_friends_refresh_give_points(this.Id, friends[idx+1:])
			if err_code < 0 {
				log.Error("Player[%v] to refresh friend[%v] give points error %v", this.Id, friends[idx+1:], err_code)
			}
		}

		this.db.FriendRelative.SetLastRefreshTime(int32(time.Now().Unix()))
		this.db.FriendRelative.SetGiveNumToday(0)
	}

	return
}

func (this *Player) get_friend_info(friend_id int32, now_time time.Time) *msg_client_message.FriendInfo {
	f := player_mgr.GetPlayerById(friend_id)
	if f == nil {
		return nil
	}

	last_save, _ := this.db.Friends.GetLastGivePointsTime(friend_id)
	remain_seconds := utils.GetRemainSeconds2NextDayTime(last_save, global_config.FriendGivePointsRefreshTime)
	friend_points, _ := this.db.FriendPoints.GetGivePoints(friend_id)
	return &msg_client_message.FriendInfo{
		PlayerId:         friend_id,
		Name:             f.db.GetName(),
		Head:             f.db.Info.GetHead(),
		Level:            f.db.GetLevel(),
		VipLevel:         f.db.Info.GetVipLvl(),
		LastLogin:        f.db.Info.GetLastLogin(),
		FriendPoints:     friend_points,
		UnreadMessageNum: this.db.FriendChatUnreadIds.GetUnreadMessageNum(friend_id),
		Zan:              f.rpc_get_zan(),
		IsZanToday:       this.is_today_zan(friend_id, now_time),
		LeftGiveSeconds:  remain_seconds,
	}
}

func (this *Player) get_friends_info(friend_ids []int32) []*msg_client_message.FriendInfo {
	now_time := time.Now()
	var friend_list []*msg_client_message.FriendInfo
	for i := 0; i < len(friend_ids); i++ {
		friend_info := this.get_friend_info(friend_ids[i], now_time)
		if friend_info == nil {
			continue
		}
		friend_list = append(friend_list, friend_info)
	}
	return friend_list
}

func (this *Player) get_friend_req_info(friend_id int32) *msg_client_message.FriendReq {
	f := player_mgr.GetPlayerById(friend_id)
	if f == nil {
		return nil
	}

	return &msg_client_message.FriendReq{
		PlayerId: friend_id,
		Name:     f.db.GetName(),
		Head:     f.db.Info.GetHead(),
		Level:    f.db.GetLevel(),
	}
}

func (this *Player) get_friends_req_info(friend_ids []int32) []*msg_client_message.FriendReq {
	var req_list []*msg_client_message.FriendReq
	for i := 0; i < len(friend_ids); i++ {
		req_info := this.get_friend_req_info(friend_ids[i])
		if req_info == nil {
			continue
		}
		req_list = append(req_list, req_info)
	}
	return req_list
}

func remote_friends_info(from_player_id int32, to_player_ids []int32) (friends_info []*msg_client_message.FriendInfo, err_code int32) {
	var req msg_rpc_message.G2GFriendsInfoRequest

	req_data, err := _marshal_msg(&req)
	if err != nil {
		err_code = -1
		return
	}

	datas := rpc_proto.RpcCommonMultiGet(game_server.rpc_client, "G2G_CommonProc.MultiGet", from_player_id, rpc_proto.OBJECT_TYPE_PLAYER, to_player_ids, int32(msg_rpc_message.MSGID_G2G_FRIENDS_INFO_REQUEST), req_data)
	if datas == nil || len(datas) == 0 {
		log.Error("get multi players %v empty", to_player_ids)
		err_code = -1
		return
	}

	var response msg_rpc_message.G2GFriendsInfoResponse
	for i := 0; i < len(datas); i++ {
		if datas[i].ErrorCode < 0 {
			err_code = -1
			log.Error("get multi players %v error %v with index %v", to_player_ids, datas[i].ErrorCode, i)
			return
		}
		err = _unmarshal_msg(datas[i].ResultData, &response)
		if err != nil {
			err_code = -1
			return
		}
		for _, f := range response.Friends {
			friends_info = append(friends_info, &msg_client_message.FriendInfo{
				PlayerId:         f.PlayerId,
				Name:             f.Name,
				Level:            f.Level,
				Head:             f.Head,
				VipLevel:         f.VipLevel,
				LastLogin:        f.LastLogin,
				FriendPoints:     f.FriendPoints,
				LeftGiveSeconds:  f.LeftGiveSeconds,
				UnreadMessageNum: f.UnreadMessageNum,
			})
		}
	}
	err_code = 1
	return
}

func remote_friends_info_response(from_player_id int32, to_player_ids []int32, req_data []byte) (resp_data []byte, err_code int32) {
	now_time := time.Now()
	var friends_info []*msg_rpc_message.FriendInfo
	for i := 0; i < len(to_player_ids); i++ {
		id := to_player_ids[i]
		player := player_mgr.GetPlayerById(id)
		if player == nil {
			continue
		}
		fi := player.get_friend_info(id, now_time)
		if fi == nil {
			log.Warn("remote request get player info by id %v from %v not found", id, to_player_ids)
			continue
		}
		friends_info = append(friends_info, &msg_rpc_message.FriendInfo{
			PlayerId:         id,
			Name:             fi.Name,
			Head:             fi.Head,
			Level:            fi.Level,
			VipLevel:         fi.VipLevel,
			LastLogin:        fi.LastLogin,
			FriendPoints:     fi.FriendPoints,
			LeftGiveSeconds:  fi.LeftGiveSeconds,
			UnreadMessageNum: fi.UnreadMessageNum,
		})
	}

	if err_code < 0 {
		return
	}

	if err_code >= 0 {
		var response = msg_rpc_message.G2GFriendsInfoResponse{
			Friends: friends_info,
		}
		var err error
		resp_data, err = _marshal_msg(&response)
		if err != nil {
			err_code = -1
			return
		}
	}

	err_code = 1
	return
}

func (this *Player) get_friend_list() int32 {
	this.check_friends_give_points_refresh()

	var friend_list []*msg_client_message.FriendInfo
	var req_list []*msg_client_message.FriendReq

	friend_ids := this.db.Friends.GetAllIndex()
	if friend_ids != nil && len(friend_ids) > 0 {
		idx := SplitLocalAndRemotePlayers(friend_ids)
		if idx >= 0 {
			friend_list = this.get_friends_info(friend_ids)
		}
		if int(idx) < len(friend_ids)-1 {
			friends_info, err_code := remote_friends_info(this.Id, friend_ids[idx+1:])
			if err_code >= 0 {
				friend_list = append(friend_list, friends_info...)
			}
		}
	}

	req_ids := this.db.FriendReqs.GetAllIndex()
	if req_ids != nil && len(req_ids) > 0 {
		idx := SplitLocalAndRemotePlayers(req_ids)
		if idx >= 0 {
			req_list = this.get_friends_req_info(req_ids)
		}
		if int(idx) < len(req_ids)-1 {
			result := this.rpc_get_players_base_info(req_ids[idx+1:])
			if result == nil {
				log.Error("")
			} else {
				for _, r := range result.PlayersInfo {
					req_list = append(req_list, &msg_client_message.FriendReq{
						PlayerId: r.Id,
						Name:     r.Name,
						Level:    r.Level,
						Head:     r.Head,
					})
				}
			}
		}
	}

	this.Send(uint16(msg_client_message.S2CRetFriendListResult_ProtoID), &msg_client_message.S2CRetFriendListResult{
		FriendList:        friend_list,
		Reqs:              req_list,
		LeftGivePointsNum: global_config.FriendGivePointsPlayerNumOneDay - this.db.FriendRelative.GetGiveNumToday(),
	})

	log.Trace("Player %v get friend list: %v %v", this.Id, friend_list, req_list)

	return 1
}

func (this *Player) store_friend_points(friend_id int32) (err int32, last_save int32, remain_seconds int32) {
	last_save, o := this.db.FriendPoints.GetLastGiveTime(friend_id)
	remain_seconds = utils.GetRemainSeconds2NextDayTime(last_save, global_config.FriendGivePointsRefreshTime)
	if remain_seconds > 0 {
		err = int32(msg_client_message.E_ERR_FRIEND_GIVE_POINTS_FREQUENTLY)
		return
	}

	now_time := time.Now()
	if !o {
		var data dbPlayerFriendPointData
		data.FromPlayerId = friend_id
		data.GivePoints = global_config.GiveFriendPointsOnce
		data.LastGiveTime = int32(now_time.Unix())
		//data.IsTodayGive = 1
		this.db.FriendPoints.Add(&data)
	} else {
		this.db.FriendPoints.SetGivePoints(friend_id, global_config.GiveFriendPointsOnce)
		this.db.FriendPoints.SetLastGiveTime(friend_id, int32(now_time.Unix()))
		//this.db.FriendPoints.SetIsTodayGive(friend_id, 1)
	}
	last_save = int32(now_time.Unix())
	remain_seconds = global_config.GiveFriendPointsOnce
	log.Debug("!!!!!!!! err[%v] last_save[%v] remain_seconds[%v]", err, last_save, remain_seconds)
	return
}

func remote_give_friend_points(from_player_id, to_player_id int32) (resp *msg_rpc_message.G2GFriendGivePointsResponse, err_code int32) {
	var req msg_rpc_message.G2GFriendGivePointsRequest
	resp = &msg_rpc_message.G2GFriendGivePointsResponse{}
	err_code = RemoteGetUsePB(from_player_id, rpc_proto.OBJECT_TYPE_PLAYER, to_player_id, int32(msg_rpc_message.MSGID_G2G_FRIEND_GIVE_POINTS_REQUEST), &req, resp)
	return
}

func remote_give_friend_points_response(from_player_id, to_player_id int32, req_data []byte) (resp_data []byte, err_code int32) {
	to_player := player_mgr.GetPlayerById(to_player_id)
	if to_player == nil {
		return
	}

	err_code, last_save, remain_seconds := to_player.store_friend_points(from_player_id)

	var response = msg_rpc_message.G2GFriendGivePointsResponse{
		LastSave:      last_save,
		RemainSeconds: remain_seconds,
	}
	var err error
	resp_data, err = _marshal_msg(&response)
	if err != nil {
		err_code = -1
		return
	}

	return
}

func (this *Player) give_friend_points(friend_list []int32) int32 {
	this.check_friends_give_points_refresh()

	if int32(len(friend_list)) > global_config.GiveFriendPointsPlayersCount {
		return int32(msg_client_message.E_ERR_FRIEND_TOO_MANY_FRIEND_GIVE_POINTS)
	}

	today_num := this.db.FriendRelative.GetGiveNumToday()
	today_max_num := global_config.FriendGivePointsPlayerNumOneDay
	if today_num >= today_max_num {
		log.Error("Player[%v] give friend points num is max", this.Id)
		return int32(msg_client_message.E_ERR_FRIEND_GIVE_POINTS_MAX_NUM_LIMIT)
	}

	n := int32(0)
	points_result := make([]*msg_client_message.FriendPointsResult, len(friend_list))
	for _, id := range friend_list {
		if id == this.Id {
			continue
		}
		if !this.db.Friends.HasIndex(id) {
			log.Error("Player[%v] no friend[%v]", this.Id, id)
			continue
		}

		if today_num >= today_max_num {
			log.Warn("Player[%v] give friend[%v] points today_num[%v] >= max_num[%v]", this.Id, id, today_num, today_max_num)
			break
		}

		points_result[n] = &msg_client_message.FriendPointsResult{}
		var err_code, last_save, remain_seconds int32
		p := player_mgr.GetPlayerById(id)
		if p != nil {
			err_code, last_save, remain_seconds = p.store_friend_points(this.Id)
		} else {
			var resp *msg_rpc_message.G2GFriendGivePointsResponse
			resp, err_code = remote_give_friend_points(this.Id, id)
			if resp == nil {
				log.Error("Player[%v] remote rpc give friend[%v] points failed", this.Id, id)
				continue
			}
			last_save = resp.LastSave
			remain_seconds = resp.RemainSeconds
		}

		if err_code < 0 {
			log.Warn("Player[%v] give friend[%v] points error[%v]", this.Id, id, err_code)
		} else {
			today_num = this.db.FriendRelative.IncbyGiveNumToday(1)
			this.AddFriendPoints(global_config.GiveFriendPointsOnce, "back_points", "friend")
		}
		this.db.Friends.SetLastGivePointsTime(id, last_save)

		points_result[n].FriendId = id
		points_result[n].Error = err_code
		points_result[n].RemainSeconds = remain_seconds
		//points_result[n].IsTodayGive = proto.Bool(true)
		if err_code >= 0 {
			points_result[n].Points = global_config.GiveFriendPointsOnce
			points_result[n].BackPoints = global_config.GiveFriendPointsOnce
		} else {
			points_result[n].Points = 0
			points_result[n].BackPoints = 0
		}

		n += 1
	}

	response := &msg_client_message.S2CGiveFriendPointsResult{
		PointsData:        points_result[:n],
		LeftGivePointsNum: global_config.FriendGivePointsPlayerNumOneDay - this.db.FriendRelative.GetGiveNumToday(),
	}

	this.Send(uint16(msg_client_message.S2CGiveFriendPointsResult_ProtoID), response)

	this.TaskUpdate(tables.TASK_COMPLETE_TYPE_GIVE_FRIEND_POINT, false, 0, n)

	log.Debug("Player[%v] give friend points: %v", this.Id, points_result)

	return 1
}

func (this *Player) get_friend_points(friend_list []int32) int32 {
	this.check_friends_give_points_refresh()

	points_result := make([]*msg_client_message.FriendPoints, len(friend_list))
	for i, id := range friend_list {
		points_result[i] = &msg_client_message.FriendPoints{}
		points, o := this.db.FriendPoints.GetGivePoints(id)
		if o && points > 0 {
			this.AddFriendPoints(points, "from_friend", "friend")
			this.db.FriendPoints.SetGivePoints(id, 0)
		}
		points_result[i].FriendId = id
		points_result[i].Points = points
	}
	response := &msg_client_message.S2CGetFriendPointsResult{}
	response.PointsData = points_result
	this.Send(uint16(msg_client_message.S2CGetFriendPointsResult_ProtoID), response)

	log.Debug("Player[%v] get friend points: %v", this.Id, points_result)
	return 1
}

func (this *Player) friend_chat_add(friend_id int32, message []byte) int32 {
	// 未读消息数量
	is_full, next_id := this.db.FriendChatUnreadIds.CheckUnreadNumFull(friend_id)
	if is_full {
		log.Debug("Player[%v] chat message from friend[%v] is full", this.Id, friend_id)
		return int32(msg_client_message.E_ERR_FRIEND_MESSAGE_NUM_MAX)
	}

	new_long_id := utils.Int64From2Int32(friend_id, next_id)
	message_data := &dbPlayerFriendChatUnreadMessageData{
		PlayerMessageId: new_long_id,
		Message:         message,
		SendTime:        int32(time.Now().Unix()),
		IsRead:          int32(0),
	}

	if !this.db.FriendChatUnreadMessages.Add(message_data) {
		log.Error("Player[%v] add friend[%v] chat message failed", this.Id, friend_id)
		return -1
	}

	res := this.db.FriendChatUnreadIds.AddNewMessageId(friend_id, next_id)
	if res < 0 {
		// 增加新ID失败则删除刚加入的消息
		this.db.FriendChatUnreadMessages.Remove(new_long_id)
		log.Error("Player[%v] add new message id[%v,%v] from friend[%v] failed", this.Id, next_id, new_long_id)
		return res
	}

	log.Debug("Player[%v] add friend[%v] chat message[id:%v, long_id:%v, content:%v]", this.Id, friend_id, next_id, new_long_id, message)

	return 1
}

func remote_friend_chat(from_player_id, to_player_id int32, message []byte) (err_code int32) {
	var req msg_rpc_message.G2GFriendChatRequest
	resp := &msg_rpc_message.G2GFriendChatResponse{}
	err_code = RemoteGetUsePB(from_player_id, rpc_proto.OBJECT_TYPE_PLAYER, to_player_id, int32(msg_rpc_message.MSGID_G2G_FRIEND_CHAT_REQUEST), &req, resp)
	return
}

func remote_friend_chat_response(from_player_id, to_player_id int32, req_data []byte) (resp_data []byte, err_code int32) {
	var req msg_rpc_message.G2GFriendChatRequest
	err := _unmarshal_msg(req_data, &req)
	if err != nil {
		err_code = -1
		return
	}

	to_player := player_mgr.GetPlayerById(to_player_id)
	if to_player == nil {
		return
	}

	err_code = to_player.friend_chat_add(from_player_id, req.Content)
	if err_code < 0 {
		return
	}

	var response = msg_rpc_message.G2GFriendChatResponse{}
	resp_data, err = _marshal_msg(&response)
	if err != nil {
		err_code = -1
		return
	}

	return
}

func (this *Player) friend_chat(friend_id int32, message []byte) int32 {
	if !this.db.Friends.HasIndex(friend_id) {
		log.Error("Player[%v] no friend[%v], chat failed", this.Id, friend_id)
		return int32(msg_client_message.E_ERR_FRIEND_NO_THE_FRIEND)
	}

	if len(message) > FRIEND_MESSAGE_MAX_LENGTH {
		log.Error("Player[%v] from friend[%v] chat content is too long[%v]", this.Id, friend_id, len(message))
		return int32(msg_client_message.E_ERR_FRIEND_MESSAGE_TOO_LONG)
	}

	friend := player_mgr.GetPlayerById(friend_id)
	if friend != nil {
		res := friend.friend_chat_add(this.Id, message)
		if res < 0 {
			return res
		}
	} else {
		err_code := remote_friend_chat(this.Id, friend_id, message)
		if err_code < 0 {
			log.Error("Player[%v] chat message[%v] to friend[%v] error[%v]", this.Id, message, friend_id, err_code)
			return err_code
		}
	}

	response := &msg_client_message.S2CFriendChatResult{}
	response.PlayerId = friend_id
	response.Content = message
	this.Send(uint16(msg_client_message.S2CFriendChatResult_ProtoID), response)

	return 1
}

func (this *Player) friend_get_unread_message_num(friend_ids []int32) int32 {
	data := make([]*msg_client_message.FriendUnreadMessageNumData, len(friend_ids))
	for i := 0; i < len(friend_ids); i++ {
		message_num := int32(0)
		if !this.db.Friends.HasIndex(friend_ids[i]) {
			message_num = int32(msg_client_message.E_ERR_FRIEND_NO_THE_FRIEND)
			log.Error("Player[%v] no friend[%v], get unread message num failed", this.Id, friend_ids[i])
		} else {
			message_num = this.db.FriendChatUnreadIds.GetUnreadMessageNum(friend_ids[i])
		}
		data[i] = &msg_client_message.FriendUnreadMessageNumData{
			FriendId:   friend_ids[i],
			MessageNum: message_num,
		}
	}

	response := &msg_client_message.S2CFriendGetUnreadMessageNumResult{}
	response.Data = data
	this.Send(uint16(msg_client_message.S2CFriendGetUnreadMessageNumResult_ProtoID), response)
	return 1
}

func (this *Player) friend_pull_unread_message(friend_id int32) int32 {
	if !this.db.Friends.HasIndex(friend_id) {
		log.Error("Player[%v] no friend[%v], pull unread message failed", this.Id, friend_id)
		return int32(msg_client_message.E_ERR_FRIEND_NO_THE_FRIEND)
	}

	c := 0
	var data []*msg_client_message.FriendChatData
	all_unread_ids, o := this.db.FriendChatUnreadIds.GetMessageIds(friend_id)
	if !o || all_unread_ids == nil || len(all_unread_ids) == 0 {
		data = make([]*msg_client_message.FriendChatData, 0)
	} else {
		data = make([]*msg_client_message.FriendChatData, len(all_unread_ids))
		for i := 0; i < len(all_unread_ids); i++ {
			long_id := utils.Int64From2Int32(friend_id, all_unread_ids[i])
			content, o := this.db.FriendChatUnreadMessages.GetMessage(long_id)
			if !o {
				log.Warn("Player[%v] no unread message[%v] from friend[%v]", this.Id, all_unread_ids[i], friend_id)
				continue
			}
			send_time, _ := this.db.FriendChatUnreadMessages.GetSendTime(long_id)
			data[c] = &msg_client_message.FriendChatData{
				Content:  content,
				SendTime: send_time,
			}
			c += 1
		}
	}

	response := &msg_client_message.S2CFriendPullUnreadMessageResult{}
	response.Data = data[:c]
	response.FriendId = friend_id
	this.Send(uint16(msg_client_message.S2CFriendPullUnreadMessageResult_ProtoID), response)

	log.Debug("Player[%v] pull unread message[%v] from friend[%v]", this.Id, response.Data, friend_id)

	return 1
}

func (this *Player) friend_confirm_unread_message(friend_id int32, message_num int32) int32 {
	if !this.db.Friends.HasIndex(friend_id) {
		log.Error("Player[%v] no friend[%v], confirm unread message failed", this.Id, friend_id)
		return int32(msg_client_message.E_ERR_FRIEND_NO_THE_FRIEND)
	}

	res, remove_ids := this.db.FriendChatUnreadIds.ConfirmUnreadIds(friend_id, message_num)
	if res < 0 {
		return res
	}

	this.db.FriendChatUnreadMessages.RemoveMessages(friend_id, remove_ids)

	response := &msg_client_message.S2CFriendConfirmUnreadMessageResult{}
	response.FriendId = friend_id
	response.MessageNum = message_num
	this.Send(uint16(msg_client_message.S2CFriendConfirmUnreadMessageResult_ProtoID), response)

	log.Debug("Player[%v] confirm friend[%v] unread message num[%v]", this.Id, friend_id, message_num)

	return 1
}

func (this *Player) open_local_chest(chest_id int32) (chest_table_id int32) {
	_, o := this.db.Buildings.GetCfgId(chest_id)
	if !o {
		log.Error("Player[%v] no building[%v]", this.Id, chest_id)
		return int32(msg_client_message.E_ERR_BUILDING_NOT_EXIST)
	}
	box := map_chest_mgr.Map[chest_table_id]
	if box == nil {
		log.Error("no chest[%v] box table data", chest_table_id)
		return int32(msg_client_message.E_ERR_BUILDING_AREA_NO_CFG)
	}
	chest_table_id, _ = this.OpenMapChest(chest_id, false)
	return
}

// 远程获取玩家宝箱配置ID
func remote_get_chest_table_id(from_player_id, to_player_id, chest_id int32) (chest_table_id, err_code int32) {
	var req msg_rpc_message.G2GGetPlayerChestTableIdRequest
	var response msg_rpc_message.G2GGetPlayerChestTableIdResponse
	err_code = RemoteGetUsePB(from_player_id, rpc_proto.OBJECT_TYPE_PLAYER, to_player_id, int32(msg_rpc_message.MSGID_G2G_GET_PLAYER_CHEST_TABLE_ID_REQUEST), &req, &response)
	chest_table_id = response.GetChestTableId()
	return
}

// 远程获取玩家宝箱配置ID返回
func remote_get_chest_table_id_response(from_player_id int32, to_player_id int32, req_data []byte) (resp_data []byte, err_code int32) {
	var req msg_rpc_message.G2GGetPlayerChestTableIdRequest
	err := _unmarshal_msg(req_data, &req)
	if err != nil {
		err_code = -1
		return
	}

	player := player_mgr.GetPlayerById(to_player_id)
	if player == nil {
		log.Error("remote request get chest table id by player id %v and chest_id %v not found", to_player_id, req.GetChestId())
		err_code = int32(msg_client_message.E_ERR_PLAYER_NOT_EXIST)
		return
	}

	chest_table_id, o := player.db.Buildings.GetCfgId(req.GetChestId())
	if !o {
		log.Error("remote get chest building %v not found", req.GetChestId())
		err_code = int32(msg_client_message.E_ERR_BUILDING_NOT_EXIST)
	}

	var response = msg_rpc_message.G2GGetPlayerChestTableIdResponse{
		ChestTableId: chest_table_id,
	}
	resp_data, err = _marshal_msg(&response)
	if err != nil {
		err_code = -1
		return
	}

	err_code = 1
	return
}

// 远程打开玩家宝箱
func remote_open_chest(from_player_id, to_player_id, chest_id int32) (chest_table_id, err_code int32) {
	var req msg_rpc_message.G2GOpenPlayerChestRequest
	var response msg_rpc_message.G2GOpenPlayerChestResposne
	err_code = RemoteGetUsePB(from_player_id, rpc_proto.OBJECT_TYPE_PLAYER, to_player_id, int32(msg_rpc_message.MSGID_G2G_OPEN_PLAYER_CHEST_TABLE_REQUEST), &req, &response)
	chest_table_id = response.GetChestTableId()
	return
}

// 远程打开玩家宝箱返回
func remote_open_chest_response(from_player_id int32, to_player_id int32, req_data []byte) (resp_data []byte, err_code int32) {
	var req msg_rpc_message.G2GOpenPlayerChestRequest
	err := _unmarshal_msg(req_data, &req)
	if err != nil {
		err_code = -1
		return
	}

	chest_id := req.GetChestId()
	player := player_mgr.GetPlayerById(to_player_id)
	if player == nil {
		err_code = int32(msg_client_message.E_ERR_PLAYER_NOT_EXIST)
		log.Error("remote open chest by player id %v and chest_id %v not found", to_player_id, chest_id)
		return
	}

	chest_table_id, o := player.db.Buildings.GetCfgId(chest_id)
	if !o {
		err_code = int32(msg_client_message.E_ERR_BUILDING_NOT_EXIST)
		log.Error("remote open chest building %v not found", chest_id)
		return
	}

	dir, _ := player.db.Buildings.GetDir(chest_id)
	building_cfg := building_table_mgr.Map[chest_table_id]
	if nil == building_cfg {
		err_code = int32(msg_client_message.E_ERR_BUILDING_NOT_CFG)
		log.Error("remote open chest building table[%d] data not found !", chest_table_id)
		return
	}
	var width, height int32
	if tables.BUILDING_DIR_BIG_X_DIR == dir {
		width, height = building_cfg.MapSizes[0], building_cfg.MapSizes[1]
	} else {
		width, height = building_cfg.MapSizes[1], building_cfg.MapSizes[0]
	}

	player.ChkUpdateMyBuildingAreas()
	x, _ := player.db.Buildings.GetX(chest_id)
	y, _ := player.db.Buildings.GetY(chest_id)
	player.RemoveAreaBuilding(chest_id, x, y, width, height)

	player.item_cat_building_change_info.building_remove(player, chest_id)
	player.item_cat_building_change_info.send_buildings_update(player)

	var response = msg_rpc_message.G2GOpenPlayerChestResposne{
		ChestTableId: chest_table_id,
	}
	resp_data, err = _marshal_msg(&response)
	if err != nil {
		err_code = -1
		return
	}

	err_code = 1
	return
}

func (this *Player) open_friend_chest(friend_id int32, chest_id int32) int32 {
	if !this.db.Friends.HasIndex(friend_id) {
		return int32(msg_client_message.E_ERR_FRIEND_NO_THE_FRIEND)
	}
	var result *msg_client_message.S2COpenMapChest
	friend := player_mgr.GetPlayerById(friend_id)
	if friend != nil {
		res := friend.open_local_chest(chest_id)
		if res < 0 {
			return res
		}
		result = this.open_chest_result(res)
		if result == nil {
			this.return_chest_cost_by_id(chest_id)
			return -1
		}
	} else {
		// 获得宝箱配置ID
		chest_table_id, err_code := remote_get_chest_table_id(this.Id, friend_id, chest_id)
		if err_code < 0 {
			log.Error("Player[%v] get friend[%v] chest[%v] table id err %v", this.Id, friend_id, chest_id, err_code)
			return err_code
		}
		// 花费
		err_code = this.open_chest_cost(chest_table_id)
		if err_code < 0 {
			return err_code
		}
		// 打开
		var ct_id int32
		ct_id, err_code = remote_open_chest(this.Id, friend_id, chest_id)
		if err_code < 0 {
			this.return_chest_cost(chest_table_id)
			log.Error("Player[%v] open friend[%v] chest[%v] err: %v", this.Id, friend_id, chest_id, err_code)
			return err_code
		}
		if ct_id != chest_table_id {
			this.return_chest_cost(chest_table_id)
			log.Error("Player %v open friend %v chest %v table_id %v not equal to %v", this.Id, friend_id, chest_id, ct_id, chest_table_id)
			return -1
		}
		result = this.open_chest_result(chest_table_id)
		if result == nil {
			this.return_chest_cost(chest_table_id)
			return -1
		}
	}

	result.BuildingId = chest_id
	result.FriendId = friend_id
	this.Send(uint16(msg_client_message.S2COpenMapChest_ProtoID), result)

	this.TaskUpdate(tables.TASK_COMPLETE_TYPE_OPEN_FRIEND_TREATURE_BOX, false, 0, 1)

	return 1
}

func C2SFriendSearchHandler(p *Player, msg_data []byte) int32 {
	var req msg_client_message.C2SFriendSearch
	err := proto.Unmarshal(msg_data, &req)
	if err != nil {
		log.Error("unmarshal msg failed %v", err.Error())
		return -1
	}

	return p.search_friend(req.GetKey())
}

func C2SAddFriendByIdHandler(p *Player, msg_data []byte) int32 {
	var req msg_client_message.C2SAddFriendByPId
	err := proto.Unmarshal(msg_data, &req)
	if err != nil {
		log.Error("unmarshal msg failed %v", err.Error())
		return -1
	}

	tgt_pid := req.GetPlayerId()
	if p.Id == tgt_pid {
		log.Error("C2SAddFriendByPIdHandler add self !")
		return -1
	}

	return p.add_friend_by_id(tgt_pid)
}

func C2SAddFriendAgreeHandler(p *Player, msg_data []byte) int32 {
	var req msg_client_message.C2SAgreeFriend
	err := proto.Unmarshal(msg_data, &req)
	if err != nil {
		log.Error("unmarshal msg failed %v", err.Error())
		return -1
	}

	return p.agree_add_friend(req.GetPlayerId())
}

func C2SRefuseAddFriendHandler(p *Player, msg_data []byte) int32 {
	var req msg_client_message.C2SRefuseFriend
	err := proto.Unmarshal(msg_data, &req)
	if err != nil {
		log.Error("unmarshal msg failed %v", err.Error())
		return -1
	}

	return p.refuse_add_friend(req.GetPlayerId())
}

func C2SFriendRemoveHandler(p *Player, msg_data []byte) int32 {
	var req msg_client_message.C2SRemoveFriend
	err := proto.Unmarshal(msg_data, &req)
	if err != nil {
		log.Error("unmarshal msg failed %v", err.Error())
		return -1
	}

	return p.remove_friend(req.GetPlayerId())
}

func C2SGetFriendListHandler(p *Player, msg_data []byte) int32 {
	var req msg_client_message.C2SGetFriendList
	err := proto.Unmarshal(msg_data, &req)
	if err != nil {
		log.Error("unmarshal msg failed %v", err.Error())
		return -1
	}

	return p.get_friend_list()
}

func C2SGiveFriendPointsHandler(p *Player, msg_data []byte) int32 {
	var req msg_client_message.C2SGiveFriendPoints
	err := proto.Unmarshal(msg_data, &req)
	if err != nil {
		log.Error("unmarshal msg failed %v", err.Error())
		return -1
	}

	return p.give_friend_points(req.GetFriendId())
}

func C2SGetFriendPointsHandler(p *Player, msg_data []byte) int32 {
	var req msg_client_message.C2SGetFriendPoints
	err := proto.Unmarshal(msg_data, &req)
	if err != nil {
		log.Error("unmarshal msg failed %v", err.Error())
		return -1
	}

	return p.get_friend_points(req.GetFriendId())
}

func C2SFriendChatHandler(p *Player, msg_data []byte) int32 {
	var req msg_client_message.C2SFriendChat
	err := proto.Unmarshal(msg_data, &req)
	if err != nil {
		log.Error("unmarshal msg failed %v", err.Error())
		return -1
	}

	return p.friend_chat(req.GetPlayerId(), req.GetContent())
}

func C2SFriendGetUnreadMessageNumHandler(p *Player, msg_data []byte) int32 {
	var req msg_client_message.C2SFriendGetUnreadMessageNum
	err := proto.Unmarshal(msg_data, &req)
	if err != nil {
		log.Error("unmarshal msg failed %v", err.Error())
		return -1
	}
	return p.friend_get_unread_message_num(req.GetFriendIds())
}

func C2SFriendPullUnreadMessageHandler(p *Player, msg_data []byte) int32 {
	var req msg_client_message.C2SFriendPullUnreadMessage
	err := proto.Unmarshal(msg_data, &req)
	if err != nil {
		log.Error("unmarshal msg failed %v", err.Error())
		return -1
	}
	return p.friend_pull_unread_message(req.GetFriendId())
}

func C2SFriendConfirmUnreadMessageHandler(p *Player, msg_data []byte) int32 {
	var req msg_client_message.C2SFriendConfirmUnreadMessage
	err := proto.Unmarshal(msg_data, &req)
	if err != nil {
		log.Error("unmarshal msg failed %v", err.Error())
		return -1
	}
	return p.friend_confirm_unread_message(req.GetFriendId(), req.GetMessageNum())
}

func C2SGetOnlineFriendsHandler(p *Player, msg_data []byte) int32 {
	var req msg_client_message.C2SGetOnlineFriends
	err := proto.Unmarshal(msg_data, &req)
	if err != nil {
		log.Error("unmarshal msg failed %v", err.Error())
		return -1
	}

	return 0
}

func C2SOpenFriendChestHandler(p *Player, msg_data []byte) int32 {
	var req msg_client_message.C2SOpenFriendChest
	err := proto.Unmarshal(msg_data, &req)
	if err != nil {
		log.Error("unmarshal msg failed %v", err.Error())
		return -1
	}

	return p.open_friend_chest(req.GetFriendid(), req.GetBuildingId())
}

// ------------------------------------------------------
