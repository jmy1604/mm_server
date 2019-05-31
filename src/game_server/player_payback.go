package main

import (
	//"mm_server/libs/log"
	//"mm_server/proto/gen_go/client_message"
	"sync"
	//"time"
	//"github.com/golang/protobuf/proto"
)

type PayBackMgr struct {
	id2payback_lock *sync.RWMutex
	//id2payback      map[int32]*msg_server_message.PayBackAdd
}

var payback_mgr PayBackMgr

func (this *PayBackMgr) Init() bool {
	this.id2payback_lock = &sync.RWMutex{}
	//this.id2payback = make(map[int32]*msg_server_message.PayBackAdd)

	return true
}

/*func (this *PayBackMgr) AddPayBack(msg *msg_server_message.PayBackAdd) {
	this.id2payback_lock.Lock()
	defer this.id2payback_lock.Unlock()

	pbid := msg.GetPayBackId()
	cur_payback := this.id2payback[pbid]
	if nil != cur_payback {
		log.Error("PayBackMgr AddPayBack paybackid[%d] already be used !", pbid)
		return
	}

	this.id2payback[pbid] = msg

	return
}

func (this *PayBackMgr) RemovePayBack(pb_id int32) {
	this.id2payback_lock.Lock()
	defer this.id2payback_lock.Unlock()

	if nil != this.id2payback[pb_id] {
		delete(this.id2payback, pb_id)
	}

	return
}

func (this *PayBackMgr) OnPlayerLogin(p *Player) {
	if nil == p {
		log.Error("PayBackMgr OnPlayerLogin p nil")
		return
	}

	this.id2payback_lock.RLock()
	defer this.id2payback_lock.RUnlock()

	var new_mail *dbPlayerMailData
	var res2cli *msg_client_message.S2CMailList
	var tmp_info *msg_client_message.MailInfo
	for pbid, pb := range this.id2payback {
		if nil == pb {
			continue
		}

		if nil != p.db.PayBacks.Get(pbid) {
			continue
		}

		new_mail = &dbPlayerMailData{}
		new_mail.MailId = p.db.Mails.GetAviMailId()
		new_mail.MailType = int8(PLAYER_MAIL_TYPE_NORMAL)
		new_mail.MailTitle = pb.GetMailTitle()
		new_mail.SenderId = PLAYER_MAIL_SENDER_ID_SYSTEM
		new_mail.SenderName = PLAYER_MAIL_SENDER_NAME_SYSYTEM
		new_mail.Content = pb.GetMailContent()
		new_mail.ObjIds = pb.ObjIds
		new_mail.ObjNums = pb.ObjNums
		new_mail.OverUnix = pb.GetOverUnix()
		new_mail.SendUnix = int32(time.Now().Unix())

		p.db.Mails.Add(new_mail)

		tmp_info = &msg_client_message.MailInfo{}
		tmp_info.MailId = proto.Int32(new_mail.MailId)
		tmp_info.MailType = proto.Int32(PLAYER_MAIL_TYPE_NORMAL)
		tmp_info.Title = proto.String(new_mail.MailTitle)
		tmp_info.SenderId = proto.Int32(new_mail.SenderId)
		tmp_info.SenderName = proto.String(new_mail.SenderName)
		tmp_info.Content = proto.String(new_mail.Content)
		tmp_info.ObjIds = pb.ObjIds
		tmp_info.ObjNums = pb.ObjNums
		tmp_info.SendUnix = proto.Int32(new_mail.SendUnix)
		tmp_info.LeftSec = proto.Int32(pb.GetOverUnix() - int32(time.Now().Unix()))
		res2cli.MailList = make([]*msg_client_message.MailInfo, 1)
		res2cli.MailList[0] = tmp_info

		p.Send(res2cli)
	}
}*/
