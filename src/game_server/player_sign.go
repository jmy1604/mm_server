package main

import (
	//"mm_server/libs/log"
	"mm_server/proto/gen_go/client_message"
	"time"
	//"github.com/golang/protobuf/proto"
)

func (this *Player) ChkPlayerSignInfo() {
	cur_month := int32(time.Now().Month())
	if cur_month != this.db.SignInfo.GetCurSignSumMonth() {
		this.db.SignInfo.SetCurSignSumMonth(cur_month)
		this.db.SignInfo.SetCurSignSum(0)
		this.db.SignInfo.SetRewardSignSum(nil)
	}
}

func (this *Player) SyncPlayerSignInfo() {
	this.ChkPlayerSignInfo()
	res2cli := &msg_client_message.S2CSyncSignInfo{}
	this.db.SignInfo.FillSyncMsg(res2cli)
	this.Send(uint16(msg_client_message.S2CSyncSignInfo_ProtoID), res2cli)
}

// --------------------------------------------

/*func reg_player_sign_msg() {
	hall_server.SetMessageHandler(msg_client_message.ID_C2SDaySign, C2SDaySignHandler)
	hall_server.SetMessageHandler(msg_client_message.ID_C2SGetDaySignSumReward, C2SGetDaySignSumRewardHandler)
}

func C2SDaySignHandler(c *socket.TcpConn, msg proto.Message) {
	req := msg.(*msg_client_message.C2SDaySign)
	if nil == c || nil == req {
		log.Error("C2SDaySignHandler c or req nil [%v]", nil == req)
		return
	}

	p := player_mgr.GetPlayerById(int32(c.T))
	if nil == p {
		log.Error("C2SDaySignHandler not login [%d]", c.T)
		return
	}

	p.ChkPlayerSignInfo()

	cur_sign_key := int32(time.Now().Year()*10000 + int(time.Now().Month())*100 + time.Now().Day())
	sign_cfg := cfg_day_sign_mgr.Map[cur_sign_key]
	if nil == sign_cfg {
		log.Error("Failed to find Day sign cfg [%d]", cur_sign_key)
		return
	}

	cur_signs := p.db.SignInfo.GetCurSignDays()
	for _, val := range cur_signs {
		if val == cur_sign_key {
			log.Error("C2SDaySignHandler already signed !!")
			return
		}
	}

	tmp_len := int32(len(cur_signs))
	new_signs := make([]int32, tmp_len+1)
	for idx, val := range cur_signs {
		new_signs[idx] = val
	}

	new_signs[tmp_len] = cur_sign_key
	p.db.SignInfo.SetCurSignDays(new_signs)
	p.db.SignInfo.IncbyCurSignSum(1)

	res2cli := &msg_client_message.S2CDaySign{}
	res2cli.SignDay = proto.Int32(cur_sign_key)

	if sign_cfg.GoldCount > 0 {
		res2cli.CurCoin = proto.Int32(p.AddCoin(sign_cfg.GoldCount, "DaySign", "DaySign"))
	}

	if sign_cfg.GemCount > 0 {
		res2cli.CurDiamond = proto.Int32(p.AddDiamond(sign_cfg.GemCount, "DaySign", "DaySign"))
	}

	if sign_cfg.ChestID > 0 {
		res2cli.ChestOpen = p.OpenChest(sign_cfg.ChestID, "Sign_Reward", "DaySign", 0, -1, true)
	}

	p.Send(res2cli)

	return
}

func C2SGetDaySignSumRewardHandler(c *socket.TcpConn, msg proto.Message) {
	req := msg.(*msg_client_message.C2SGetDaySignSumReward)
	if nil == c || nil == req {
		log.Error("C2SGetDaySignSumRewardHandler c or req nil [%v]", nil == req)
		return
	}

	p := player_mgr.GetPlayerById(int32(c.T))
	if nil == p {
		log.Error("C2SGetDaySignSumRewardHandler not login[%d]", c.T)
		return
	}

	p.ChkPlayerSignInfo()
	sign_sum := req.GetSumNum()

	if sign_sum > p.db.SignInfo.GetCurSignSum() {
		log.Error("C2SGetDaySignSumRewardHandler sign_sum not enough !")
		return
	}

	cur_sum_signs := p.db.SignInfo.GetRewardSignSum()
	for _, val := range cur_sum_signs {
		if val == sign_sum {
			return
		}
	}

	reward := cfg_day_sign_mgr.SunNum2Reward[sign_sum]
	if nil == reward {
		log.Error("C2SGetDaySignSumRewardHandler can not find sign_sum[%d] for cfg !", sign_sum)
		return
	}

	tmp_len := int32(len(cur_sum_signs)) + 1
	new_sum_signs := make([]int32, tmp_len)
	for tmp_idx, val := range cur_sum_signs {
		new_sum_signs[tmp_idx] = val
	}

	new_sum_signs[tmp_len-1] = sign_sum
	p.db.SignInfo.SetRewardSignSum(new_sum_signs)
	p.db.SignInfo.SetCurSignSumMonth(int32(time.Now().Month()))

	res2cli := &msg_client_message.S2CRetDaySignSumReward{}
	res2cli.Rewards = p.OpenChest(reward.ChestId, "Sign_sum_Reward", "DaySign", 0, -1, true)
	res2cli.SumNum = proto.Int32(sign_sum)

	p.Send(res2cli)

	return
}
*/
