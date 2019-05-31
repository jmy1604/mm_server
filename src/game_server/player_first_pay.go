package main

//"mm_server/libs/log"
//"mm_server/proto/gen_go/client_message"

//"github.com/golang/protobuf/proto"

const (
	PLAYER_FIRST_PAY_NOT_ACT  = 0 // 首充奖励未激活
	PLAYER_FIRST_PAY_ACT      = 1 // 首充奖励未领取
	PLAYER_FIRST_PAY_REWARDED = 2 // 首充奖励已经领取
)

func (this *Player) SyncPlayerFirstPayState() {
	this.OnActivityValSet(PLAYER_ACTIVITY_TYPE_FIRST_PAY, 1)
	/*
		res2cli := &msg_client_message.S2CSyncFirstPayState{}
		res2cli.CurState = proto.Int32(this.db.Info.GetFirstPayState())
		this.Send(res2cli)
	*/
}

// ----------------------------------------------------------------------------

/*func reg_player_first_pay_msg() {
	hall_server.SetMessageHandler(msg_client_message.ID_C2SGetFirstPayReward, C2SGetFirstPayRewardHandler)
}

func C2SGetFirstPayRewardHandler(c *socket.TcpConn, msg proto.Message) {
	req := msg.(*msg_client_message.C2SGetFirstPayReward)
	if nil == c || nil == req {
		log.Error("C2SGetFirstPayRewardHandler c or req nil [%v]", nil == req)
		return
	}

	p := player_mgr.GetPlayerById(int32(c.T))
	if nil == p {
		log.Error("C2SGetFirstPayRewardHandler not login[%d]", c.T)
		return
	}

	cur_state := p.db.Info.GetFirstPayState()
	if PLAYER_FIRST_PAY_ACT != cur_state {
		log.Error("C2SGetFirstPayRewardHandler state[%d] error", cur_state)
		return
	}

	p.db.Info.SetFirstPayState(PLAYER_FIRST_PAY_REWARDED)
	res2cli := &msg_client_message.S2CRetFirstPayReward{}
	res2cli.Rewards = p.OpenChest(global_config_mgr.GetGlobalConfig().FirstPayReward, "first_pay_reward", "first_pay", 0, -1, true)
	p.Send(res2cli)

	return
}*/
