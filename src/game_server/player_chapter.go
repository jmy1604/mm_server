package main

import (
	"mm_server/libs/log"
	"mm_server/libs/timer"
	"mm_server/proto/gen_go/client_message"
	"time"

	"github.com/golang/protobuf/proto"
)

const (
	PLAYER_CHAPTER_UNLOCK_TYPE_TIME      = 0 // 时间解锁
	PLAYER_CHAPTER_UNLOCK_TYPE_STAR      = 1 // 星星解锁
	PLAYER_CHAPTER_UNLOCK_TYPE_DIAMOND   = 2 // 钻石解锁
	PLAYER_CHAPTER_UNLOCK_TYPE_REQFRIEND = 3 // 请求好友
)

func (this *Player) ChkSendNewUnlockStage() {
	if this.new_unlock_chapter_id > 0 {
		this.new_unlock_chapter_id = 0
		res2cli := &msg_client_message.S2CChapterUnlock{}
		res2cli.ChapterId = this.new_unlock_chapter_id
		res2cli.MaxUnlockStageId = this.db.Info.GetMaxUnlockStage()

		this.Send(uint16(msg_client_message.S2CChapterUnlock_ProtoID), res2cli)
	}
}

func (this *Player) ChkDayHelpUnlockNum(bsend bool) {
	cur_unix_day := timer.GetDayFrom1970WithCfg(0)
	if cur_unix_day != this.db.Info.GetDayHelpUnlockUpDay() {
		this.db.Info.SetDayHelpUnlockUpDay(cur_unix_day)
		this.db.Info.SetDayHelpUnlockCount(0)
	}

	if bsend {
		res2cli := &msg_client_message.S2CRetDayHelpUnlockCount{}
		res2cli.HelpOtherNum = global_config.MaxHelpUnlockNum - this.db.Info.GetDayHelpUnlockCount()

		this.Send(uint16(msg_client_message.S2CRetDayHelpUnlockCount_ProtoID), res2cli)
	}

}

func (this *Player) HelpUnlock(pid, chapter_id int32) int32 {
	this.ChkDayHelpUnlockNum(false)
	if this.db.Info.GetDayHelpUnlockCount() >= global_config.MaxHelpUnlockNum {
		return int32(msg_client_message.E_ERR_CHAPTER_HELP_UNLOCK_LESS_NUM)
	}

	/*req2co := &msg_server_message.ChapterUnlockAgree{}
	req2co.AgreePlayerId = proto.Int32(this.Id)
	req2co.ChapterId = proto.Int32(chapter_id)
	req2co.AgreePlayerName = proto.String(this.db.GetName())
	req2co.ReqPlayerId = proto.Int32(pid)

	center_conn.Send(req2co)*/

	this.ChkDayHelpUnlockNum(true)

	return 0
}

// -------------------------------------------------------------------------

func reg_player_chapter_msg() {
	msg_handler_mgr.SetPlayerMsgHandler(uint16(msg_client_message.C2SChapterUnlock_ProtoID), C2SChapterUnlockHandler)
	msg_handler_mgr.SetPlayerMsgHandler(uint16(msg_client_message.C2SGetCurHelpReqPIds_ProtoID), C2SGetCurHelpReqPIdsHandler)

	//center_conn.SetMessageHandler(msg_server_message.ID_ChapterUnlockHelp, C2HChapterUnlockHelpHandler)
	//center_conn.SetMessageHandler(msg_client_message.ID_C2SAgreeMailHelpReq, C2HChapterUnlockAgreeHandler)
}

func C2SChapterUnlockHandler(p *Player, msg_data []byte) int32 {
	var req msg_client_message.C2SChapterUnlock
	err := proto.Unmarshal(msg_data, &req)
	if err != nil {
		log.Error("unmarshal msg failed %v", err.Error())
		return -1
	}

	chapter_id := req.GetChapterId()
	chapter_cfg := chapter_table_mgr.Map[chapter_id]
	if nil == chapter_cfg {
		log.Error("C2SChapterUnlockHandler failed to find chapter_cfg[%d] !", chapter_id)
		return int32(msg_client_message.E_ERR_CHAPTER_UNLOCK_NO_UNLOCK_CFG)
	}

	cur_max_chapter_id := p.db.Info.GetMaxChapter()
	cur_max_chapter_cfg := chapter_table_mgr.Map[cur_max_chapter_id]
	if nil == cur_max_chapter_cfg {
		log.Error("C2SChapterUnlockHandler failed to find cur_max_chapter_cfg[%d] !", cur_max_chapter_id)
		return int32(msg_client_message.E_ERR_CHAPTER_UNLOCK_NO_CUR_UNLOCK_CFG)
	}

	if chapter_id != cur_max_chapter_id+1 {
		log.Error("C2SChapterUnlockHandler not next chapter id [%d] [%d]", cur_max_chapter_id, chapter_id)
		return int32(msg_client_message.E_ERR_CHAPTER_UNLOCK_WRONG_CHAPTER_ID)
	}

	if nil == p.db.Stages.Get(cur_max_chapter_cfg.MaxStageId) {
		log.Error("C2SChapterUnlockHandler not finished cur chapter max stage [%d]", cur_max_chapter_cfg.MaxStageId)
		return int32(msg_client_message.E_ERR_CHAPTER_UNLOCK_NEED_PASS_STAGE)
	}

	cur_chapter_cfg := chapter_table_mgr.Map[chapter_id-1]
	if nil == cur_chapter_cfg || nil == p.db.Stages.Get(cur_chapter_cfg.MaxStageId) {
		return int32(msg_client_message.E_ERR_CHAPTER_UNLOCK_NO_CUR_UNLOCK_CFG)
	}

	b_new := false
	if chapter_id != p.db.ChapterUnLock.GetChapterId() {
		p.db.ChapterUnLock.SetNewUnlockChapter(chapter_id)
		b_new = true
	}

	cur_unix := int32(time.Now().Unix())
	switch req.GetUnLockType() {
	case PLAYER_CHAPTER_UNLOCK_TYPE_TIME:
		{

			if b_new || cur_unix-p.db.ChapterUnLock.GetStartUnix() < chapter_cfg.UnlockTime {
				return 0
			}

		}
	case PLAYER_CHAPTER_UNLOCK_TYPE_STAR:
		{
			if p.db.Stages.GetTotalTopStar() < chapter_cfg.UnlockStarNum {
				return int32(msg_client_message.E_ERR_CHAPTER_UNLOCK_LESS_STAR)
			}
		}
	case PLAYER_CHAPTER_UNLOCK_TYPE_DIAMOND:
		{
			needsec := chapter_cfg.UnlockTime - (cur_unix - p.db.ChapterUnLock.GetStartUnix())
			need_diamond := (needsec + global_config.ChapterUnlockSecPerDiamond - 1) / global_config.ChapterUnlockSecPerDiamond
			if p.db.Info.GetDiamond() < need_diamond {
				return int32(msg_client_message.E_ERR_CHAPTER_UNLOCK_LESS_DIAMOND)
			}

			p.SubDiamond(need_diamond, "chapter_unlock", "chapter")
		}
	case PLAYER_CHAPTER_UNLOCK_TYPE_REQFRIEND:
		{
			/*req2c := &msg_server_message.ChapterUnlockHelp{}
			tmp_len := int32(len(req2c.HelpPlayerIds))
			if len(req2c.HelpPlayerIds) <= 0 {
				return int32(msg_client_message.E_ERR_CHAPTER_UNLOCK_NO_FRIEND_IDS)
			}
			req2c.HelpPlayerIds = make([]int32, 0, tmp_len)
			req2c.ChapterId = proto.Int32(chapter_id)
			req2c.ReqPlayerId = proto.Int32(p.Id)
			req2c.ReqPlayerName = proto.String(p.db.GetName())

			cur_req_pids := p.db.ChapterUnLock.GetPlayerIds()
			new_len := tmp_len + int32(len(cur_req_pids))
			new_pids := make([]int32, 0, new_len)

			p.db.ChapterUnLock.SetPlayerIds(new_pids)

			for _, val := range cur_req_pids {
				cur_req_pids = append(cur_req_pids, val)
			}

			bfind := false
			for _, val := range req.GetFriendIds() {
				bfind = false
				for _, old_pid := range cur_req_pids {
					if old_pid == val {
						bfind = true
						break
					}
				}

				if !bfind {
					cur_req_pids = append(cur_req_pids, val)
					req2c.HelpPlayerIds = append(req2c.HelpPlayerIds, val)
				}
			}

			if len(req2c.HelpPlayerIds) <= 0 {
				return 0
			}

			center_conn.Send(req2c)*/

			return 0
		}
	}

	p.db.ChapterUnLock.Reset()
	p.db.Info.SetMaxUnlockStage(chapter_cfg.MaxStageId)
	p.db.Info.SetMaxChapter(chapter_id)
	res2cli := &msg_client_message.S2CChapterUnlock{}
	res2cli.ChapterId = chapter_id
	res2cli.MaxUnlockStageId = chapter_cfg.MaxStageId

	p.Send(uint16(msg_client_message.S2CChapterUnlock_ProtoID), res2cli)

	return 1
}

func C2SGetCurHelpReqPIdsHandler(p *Player, msg_data []byte) int32 {
	res2cli := &msg_client_message.S2CRetCurHelpReqPIds{}
	res2cli.PIds = p.db.ChapterUnLock.GetPlayerIds()

	p.Send(uint16(msg_client_message.S2CRetCurHelpReqPIds_ProtoID), res2cli)
	return 1
}

// ============================================================================

/*
func C2HChapterUnlockHelpHandler(c *CenterConnection, msg proto.Message) {
	req := msg.(*msg_server_message.ChapterUnlockHelp)
	if nil == req || nil == c {
		log.Error("C2HChapterUnlockHelpHandler c or res nil !")
		return
	}

	var p *Player
	pids := req.GetHelpPlayerIds()
	for _, pid := range pids {
		p = player_mgr.GetPlayerById(pid)
		if nil == p {
			continue
		}

		p.SendHelpMail(req.GetReqPlayerId(), req.GetReqPlayerName(), req.GetChapterId())
	}

	return
}

func C2HChapterUnlockAgreeHandler(c *CenterConnection, msg proto.Message) {
	req := msg.(*msg_server_message.ChapterUnlockAgree)
	if nil == req || nil == c {
		log.Error("C2HChapterUnlockHelpHandler c or res nil !")
		return
	}

	p := player_mgr.GetPlayerById(req.GetReqPlayerId())
	if nil == p {
		log.Error("ChapterUnlockAgreeHandler failed to find player[%d] !", req.GetReqPlayerId())
		return
	}

	chapter_id := req.GetChapterId()
	if chapter_id != p.db.ChapterUnLock.GetChapterId() {
		log.Error("ChapterUnlockAgreeHandler not maptch %d %d", chapter_id, p.db.ChapterUnLock.GetChapterId())
		return
	}

	agree_pid := req.GetAgreePlayerId()
	cur_agree_pids := p.db.ChapterUnLock.GetCurHelpIds()

	for _, pid := range cur_agree_pids {
		if pid == agree_pid {
			log.Error("ChapterUnlockAgreeHandler [%d] already in agree_pids [%v]", cur_agree_pids, pid)
			return
		}
	}

	tmp_len := int32(len(cur_agree_pids)) + 1
	log.Info("当前帮助好友数目")
	if tmp_len >= global_config_mgr.GetGlobalConfig().ChapterUnlockNeedFriendNum {
		chapter_cfg := cfg_chapter_mgr.Map[chapter_id]
		if nil == chapter_cfg {
			log.Error("ChapterUnlockAgreeHandler failed to find chapter cfg [%d] !", chapter_cfg.ChapterId)
			return
		}

		p.db.ChapterUnLock.Reset()
		p.db.Info.SetMaxUnlockStage(chapter_cfg.MaxStageId)
		p.new_unlock_chapter_id = chapter_id
		return
	}

	new_help_ids := make([]int32, 0, tmp_len)
	for _, pid := range cur_agree_pids {
		new_help_ids = append(new_help_ids, pid)
	}

	new_help_ids = append(new_help_ids, agree_pid)
	p.db.ChapterUnLock.SetCurHelpIds(new_help_ids)

	p.ChkDayHelpUnlockNum(true)

	return
}
*/
