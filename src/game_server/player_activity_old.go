package main

import (
	"mm_server/libs/log"
	"mm_server/libs/timer"
	"mm_server/proto/gen_go/client_message"
	"time"

	"mm_server/src/tables"

	"github.com/golang/protobuf/proto"
)

const (
	DEFAULT_PLAYER_MSG_ACT_UPDATE = 0
	DEFAULT_PLAYER_MSG_ACT_REMOVE = 1

	DEFAULT_PLAYER_MSG_ACT_ARRAY_LEN  = 10
	DEFAULT_PLAYER_MSG_ACT_ARRAY_STEP = 5

	PLAYER_ACTIVITY_TYPE_FIRST_PAY      = 1 // 首冲类型
	PLAYER_ACTIVITY_TYPE_DAY_REWARD     = 2 // 每日奖励
	PLAYER_ACTIVITY_TYPE_LVL_REWARD     = 3 // 等级奖励
	PLAYER_ACTIVITY_TYPE_VIP_CARD       = 4 // 月卡奖励
	PLAYER_ACTIVITY_TYPE_SUM_DAY_REWARD = 5 //累计奖励

	PLAYER_ACTIVITY_STATE_NORMAL   = 0 // 初始状态
	PLAYER_ACTIVITY_STATE_FINISHED = 1 // 可以领奖状态
	PLAYER_ACTIVITY_STATE_REWARDED = 2 // 可以领奖状态

	PLAYER_ACTIVITY_START_NOLIMIT      = 0 // 无限限制
	PLAYER_ACTIVITY_START_P_CREATE_DAY = 1 // 从账号创建开始计算天
	PLAYER_ACTIVITY_START_P_CREATE_SEC = 2 // 从账号创建开始计算秒
	PLAYER_ACTIVITY_START_DATE         = 3 // 年夜日时分秒
	PLAYER_ACTIVITY_START_S_CREATE_DAY = 4 // 从服务器创建开始计算天
	PLAYER_ACTIVITY_START_S_CREATE_SEC = 5 // 从服务器创建开始计算秒
	PLAYER_ACTIVITY_START_WEEK_DAY     = 6 // 周几
	PLAYER_ACTIVITY_START_MONTH_DAY    = 7 // 每月几号

	PLAYER_ACTIVITY_END_NOLIMIT      = 0 // 无限限制
	PLAYER_ACTIVITY_END_P_CREATE_DAY = 1 // 从账号创建开始计算天
	PLAYER_ACTIVITY_END_P_CREATE_SEC = 2 // 从账号创建开始计算秒
	PLAYER_ACTIVITY_END_DATE         = 3 // 年夜日时分秒
	PLAYER_ACTIVITY_END_S_CREATE_DAY = 4 // 从服务器创建开始计算天
	PLAYER_ACTIVITY_END_S_CREATE_SEC = 5 // 从服务器创建开始计算秒
	PLAYER_ACTIVITY_END_WEEK_DAY     = 6 // 周几
	PLAYER_ACTIVITY_END_MONTH_DAY    = 7 // 每月几号

	PLAYER_ACTIVITY_REWARD_WAY_DIRECT = 1 // 直接发奖励
	PLAYER_ACTIVITY_REWARD_WAY_MAIL   = 2 // 邮件发奖励
)

func (this *dbPlayerActivityColumn) IfHaveAct(act_id int32) bool {
	this.m_row.m_lock.UnSafeRLock("dbPlayerActivityColumn.IfHaveAct")
	defer this.m_row.m_lock.UnSafeRUnlock()

	if nil == this.m_data[act_id] {
		return false
	}

	return true
}

func (this *dbPlayerActivityColumn) FillAllClientMsg(vip_left_day int32) (ret_msg *msg_client_message.S2CActivityInfosUpdate) {

	this.m_row.m_lock.UnSafeLock("dbPlayerActivityColumn.GetAll")
	defer this.m_row.m_lock.UnSafeUnlock()

	tmp_len := int32(len(this.m_data))
	if tmp_len < 1 {
		return nil
	}

	ret_msg = &msg_client_message.S2CActivityInfosUpdate{}
	ret_msg.Activityinfos = make([]*msg_client_message.ActivityInfo, 0, tmp_len)
	var tmp_info *msg_client_message.ActivityInfo
	var task_cfg *tables.XmlActivityOldItem
	//cur_unix_day := timer.GetDayFrom1970WithCfg(0)
	for _, v := range this.m_data {
		log.Info("dbPlayerActivityColumn 处理 活动 [%d] [%v] %v", v.CfgId, this.m_data, &this.m_data)
		task_cfg = activity_old_table_mgr.Map[v.CfgId]
		if nil == task_cfg {
			log.Error("dbPlayerActivityColumn 找不到配置[%d]", v.CfgId)
			continue
		}

		tmp_info = &msg_client_message.ActivityInfo{}
		tmp_info.CfgId = v.CfgId
		tmp_info.States = v.States
		tmp_info.Vals = v.Vals

		ret_msg.Activityinfos = append(ret_msg.Activityinfos, tmp_info)
	}

	return
}

func (this *dbPlayerActivityColumn) GetVals0(id int32) int32 {
	this.m_row.m_lock.UnSafeRLock("dbPlayerActivityColumn.GetVals0")
	defer this.m_row.m_lock.UnSafeRUnlock()
	d := this.m_data[id]
	if d == nil {
		log.Error("GetVals0 not exist %v %v", this.m_row.GetPlayerId(), id)
		return 0
	}

	if len(d.Vals) < 1 {
		return 0
	}

	return d.Vals[0]
}

func (this *dbPlayerActivityColumn) GetValsEnd(id int32) int32 {
	this.m_row.m_lock.UnSafeRLock("dbPlayerActivityColumn.GetValsEnd")
	defer this.m_row.m_lock.UnSafeRUnlock()
	d := this.m_data[id]
	if d == nil {
		log.Error("GetValsEnd not exist %v %v", this.m_row.GetPlayerId(), id)
		return 0
	}

	tmp_len := len(d.Vals)
	if tmp_len < 1 {
		return 0
	}

	return d.Vals[tmp_len-1]
}

func (this *dbPlayerActivityColumn) IfValsHave(id, v int32) bool {
	this.m_row.m_lock.UnSafeRLock("dbPlayerActivityColumn.IfValsHave")
	defer this.m_row.m_lock.UnSafeRUnlock()
	d := this.m_data[id]
	if d == nil {
		log.Error("IfStatesHave not exist %v %v", this.m_row.GetPlayerId(), id)
		return false
	}

	for _, val := range d.Vals {
		if val == v {
			return true
		}
	}

	return false
}

func (this *dbPlayerActivityColumn) SetVals0(id int32, v int32) {
	this.m_row.m_lock.UnSafeLock("dbPlayerActivityColumn.SetVals0")
	defer this.m_row.m_lock.UnSafeUnlock()
	d := this.m_data[id]
	if d == nil {
		log.Error("SetVals0 not exist %v %v", this.m_row.GetPlayerId(), id)
		return
	}

	if len(d.Vals) < 1 {
		d.Vals = make([]int32, 1)
	}

	d.Vals[0] = v

	this.m_changed = true
	return
}

func (this *dbPlayerActivityColumn) AddValsVal(id int32, v int32) {
	this.m_row.m_lock.UnSafeLock("dbPlayerActivityColumn.AddValsVal")
	defer this.m_row.m_lock.UnSafeUnlock()
	d := this.m_data[id]
	if d == nil {
		log.Error("AddValsVal not exist %v %v", this.m_row.GetPlayerId(), id)
		return
	}

	tmp_len := int32(len(d.Vals))
	new_vals := make([]int32, tmp_len+1)
	for idx, val := range d.Vals {
		new_vals[idx] = val
	}

	new_vals[tmp_len] = v
	d.Vals = new_vals

	this.m_changed = true
	return
}

func (this *dbPlayerActivityColumn) RemoveValsVal(id int32, v int32) {
	this.m_row.m_lock.UnSafeLock("dbPlayerActivityColumn.RemoveValsVal")
	defer this.m_row.m_lock.UnSafeUnlock()
	d := this.m_data[id]
	if d == nil {
		log.Error("AddValsVal not exist %v %v", this.m_row.GetPlayerId(), id)
		return
	}

	tmp_len := int32(len(d.Vals))
	new_vals := make([]int32, 0, tmp_len)
	for _, val := range d.Vals {
		if val != v {
			new_vals = append(new_vals, val)
		} else {
			this.m_changed = true
		}
	}

	if this.m_changed {
		d.Vals = new_vals
	}

	return
}

func (this *dbPlayerActivityColumn) ClearVals(id int32) {
	this.m_row.m_lock.UnSafeLock("dbPlayerActivityColumn.ClearVals")
	defer this.m_row.m_lock.UnSafeUnlock()
	d := this.m_data[id]
	if d == nil {
		log.Error("ClearVals not exist %v %v", this.m_row.GetPlayerId(), id)
		return
	}

	d.Vals = nil

	this.m_changed = true
	return
}

func (this *dbPlayerActivityColumn) GetStates0(id int32) int32 {
	this.m_row.m_lock.UnSafeRLock("dbPlayerActivityColumn.GetStates0")
	defer this.m_row.m_lock.UnSafeRUnlock()
	d := this.m_data[id]
	if d == nil {
		log.Error("GetStates0 not exist %v %v", this.m_row.GetPlayerId(), id)
		return 0
	}

	if len(d.States) < 1 {
		return 0
	}

	return d.States[0]
}

func (this *dbPlayerActivityColumn) GetStates1(id int32) int32 {
	this.m_row.m_lock.UnSafeRLock("dbPlayerActivityColumn.GetStates1")
	defer this.m_row.m_lock.UnSafeRUnlock()
	d := this.m_data[id]
	if d == nil {
		log.Error("GetStates0 not exist %v %v", this.m_row.GetPlayerId(), id)
		return 0
	}

	if len(d.States) < 2 {
		return 0
	}

	return d.States[1]
}

func (this *dbPlayerActivityColumn) GetStates2(id int32) int32 {
	this.m_row.m_lock.UnSafeRLock("dbPlayerActivityColumn.GetStates2")
	defer this.m_row.m_lock.UnSafeRUnlock()
	d := this.m_data[id]
	if d == nil {
		log.Error("GetStates0 not exist %v %v", this.m_row.GetPlayerId(), id)
		return 0
	}

	if len(d.States) < 3 {
		return 0
	}

	return d.States[2]
}

func (this *dbPlayerActivityColumn) IfStatesHave(id, v int32) bool {
	this.m_row.m_lock.UnSafeRLock("dbPlayerActivityColumn.GetStates0")
	defer this.m_row.m_lock.UnSafeRUnlock()
	d := this.m_data[id]
	if d == nil {
		log.Error("IfStatesHave not exist %v %v", this.m_row.GetPlayerId(), id)
		return false
	}

	for _, val := range d.States {
		if val == v {
			return true
		}
	}

	return false
}

func (this *dbPlayerActivityColumn) SetStates0(id int32, v int32) {
	this.m_row.m_lock.UnSafeLock("dbPlayerActivityColumn.SetStates0")
	defer this.m_row.m_lock.UnSafeUnlock()
	d := this.m_data[id]
	if d == nil {
		log.Error("not exist %v %v", this.m_row.GetPlayerId(), id)
		return
	}

	if len(d.States) < 1 {
		d.States = make([]int32, 1)
	}

	d.States[0] = v

	this.m_changed = true
	return
}

func (this *dbPlayerActivityColumn) IncbyStates0(id int32, v int32) {
	this.m_row.m_lock.UnSafeLock("dbPlayerActivityColumn.IncbyStates0")
	defer this.m_row.m_lock.UnSafeUnlock()
	d := this.m_data[id]
	if d == nil {
		log.Error("not exist %v %v", this.m_row.GetPlayerId(), id)
		return
	}

	if len(d.States) < 1 {
		d.States = make([]int32, 1)
	}

	d.States[0] += v

	this.m_changed = true
	return
}

func (this *dbPlayerActivityColumn) SetStates1(id int32, v int32) {
	this.m_row.m_lock.UnSafeLock("dbPlayerActivityColumn.SetStates1")
	defer this.m_row.m_lock.UnSafeUnlock()
	d := this.m_data[id]
	if d == nil {
		log.Error("not exist %v %v", this.m_row.GetPlayerId(), id)
		return
	}

	cur_len := int32(len(d.States))
	if cur_len < 2 {
		new_states := make([]int32, 2)
		for idx := int32(0); idx < cur_len; idx++ {
			new_states[idx] = d.States[idx]
		}

		d.States = new_states
	}

	d.States[1] = v

	this.m_changed = true
	return
}

func (this *dbPlayerActivityColumn) SetStates2(id int32, v int32) {
	this.m_row.m_lock.UnSafeLock("dbPlayerActivityColumn.SetStates2")
	defer this.m_row.m_lock.UnSafeUnlock()
	d := this.m_data[id]
	if d == nil {
		log.Error("not exist %v %v", this.m_row.GetPlayerId(), id)
		return
	}

	cur_len := int32(len(d.States))
	if cur_len < 3 {
		new_states := make([]int32, 3)
		for idx := int32(0); idx < cur_len; idx++ {
			new_states[idx] = d.States[idx]
		}

		d.States = new_states
	}

	d.States[2] = v

	this.m_changed = true
	return
}

func (this *dbPlayerActivityColumn) AddStateVal(id, v int32, bunique bool) {
	this.m_row.m_lock.UnSafeLock("dbPlayerActivityColumn.AddState")
	defer this.m_row.m_lock.UnSafeUnlock()
	d := this.m_data[id]
	if d == nil {
		log.Error("AddState not exist %v %v", this.m_row.GetPlayerId(), id)
		return
	}

	tmp_len := len(d.States)
	new_states := make([]int32, tmp_len+1)
	for idx := 0; idx < tmp_len; idx++ {
		if bunique && d.States[idx] == v {
			return
		}

		new_states[idx] = d.States[idx]
	}

	new_states[tmp_len] = v

	d.States = new_states

	this.m_changed = true
	return
}

// ---------------------------------------------------------------------------

func (this *Player) GetActStartCheckSec(act_cfg *tables.XmlActivityOldItem) int32 {
	ret_unix := int32(time.Now().Unix())

	switch act_cfg.StartTimeType {

	case PLAYER_ACTIVITY_START_P_CREATE_DAY:
		fallthrough
	case PLAYER_ACTIVITY_START_S_CREATE_DAY:
		{
			ret_unix = timer.GetDayFrom1970WithCfg(0) * 24 * 3600
		}
	}

	return ret_unix
}

func (this *Player) GetActEndCheckSec(act_cfg *tables.XmlActivityOldItem) int32 {
	ret_unix := int32(time.Now().Unix())

	switch act_cfg.EndTimeType {

	case PLAYER_ACTIVITY_END_P_CREATE_DAY:
		fallthrough
	case PLAYER_ACTIVITY_END_S_CREATE_DAY:
		{
			ret_unix = timer.GetDayFrom1970WithCfg(0) * 24 * 3600
		}
	}

	return ret_unix
}

func (this *Player) GetActStartSec(act_cfg *tables.XmlActivityOldItem) int32 {
	switch act_cfg.StartTimeType {
	case PLAYER_ACTIVITY_START_NOLIMIT:
		{
			return 0
		}
	case PLAYER_ACTIVITY_START_P_CREATE_DAY:
		{
			create_unix_day := timer.GetDayFrom1970WithCfgAndSec(0, this.db.Info.GetCreateUnix())
			return (create_unix_day + act_cfg.StartTimeParams[0] - 1) * 24 * 3600
		}
	case PLAYER_ACTIVITY_START_P_CREATE_SEC:
		{
			return this.db.Info.GetCreateUnix() + act_cfg.StartTimeParams[0]
		}
	case PLAYER_ACTIVITY_START_DATE:
		{
			return int32(act_cfg.StartTime.Unix())
		}
	case PLAYER_ACTIVITY_START_S_CREATE_DAY:
		{
			create_unix_day := timer.GetDayFrom1970WithCfgAndSec(0 /*hall_server.server_info_row.GetCreateUnix()*/, 0)
			return (create_unix_day + act_cfg.StartTimeParams[0] - 1) * 24 * 3600
		}
	case PLAYER_ACTIVITY_START_S_CREATE_SEC:
		{
			return /*hall_server.server_info_row.GetCreateUnix()*/ 0 + act_cfg.StartTimeParams[0]
		}
	}

	return 0
}

func (this *Player) GetActEndSec(act_cfg *tables.XmlActivityOldItem) int32 {
	switch act_cfg.EndTimeType {
	case PLAYER_ACTIVITY_END_NOLIMIT:
		{
			return 0
		}
	case PLAYER_ACTIVITY_END_P_CREATE_DAY:
		{
			create_unix_day := timer.GetDayFrom1970WithCfgAndSec(0, this.db.Info.GetCreateUnix())
			return (create_unix_day + act_cfg.EndTimeParams[0] - 1) * 24 * 3600
		}
	case PLAYER_ACTIVITY_END_P_CREATE_SEC:
		{
			return this.db.Info.GetCreateUnix() + act_cfg.EndTimeParams[0]
		}
	case PLAYER_ACTIVITY_END_DATE:
		{
			return int32(act_cfg.EndTime.Unix())
		}
	case PLAYER_ACTIVITY_END_S_CREATE_DAY:
		{
			create_unix_day := timer.GetDayFrom1970WithCfgAndSec(0 /*hall_server.server_info_row.GetCreateUnix()*/, 0)
			return (create_unix_day + act_cfg.EndTimeParams[0] - 1) * 24 * 3600
		}
	case PLAYER_ACTIVITY_END_S_CREATE_SEC:
		{
			return /*hall_server.server_info_row.GetCreateUnix()*/ 0 + act_cfg.EndTimeParams[0]
		}
	}

	return 0
}

func (this *Player) IfActOpen(act_cfg *tables.XmlActivityOldItem) bool {
	start_sec := this.GetActStartSec(act_cfg)
	start_chk_sec := this.GetActStartCheckSec(act_cfg)
	if start_sec > 0 && start_chk_sec < start_sec {
		return false
	}

	end_sec := this.GetActEndSec(act_cfg)
	end_chk_sec := this.GetActEndCheckSec(act_cfg)
	if end_sec > 0 && end_chk_sec > end_sec {
		return false
	}

	return true
}

// ---------------------------------------------------------------------------

func (this *Player) ChkSendActUpdate() {
	tmp_msg := this.PopPlayerActMsg()
	if nil != tmp_msg {
		this.Send(uint16(msg_client_message.S2CActivityInfosUpdate_ProtoID), tmp_msg)
	}
}

func (this *Player) AddMonthCard(day_count int32) {
	this.db.Info.SetVipCardEndDay(timer.GetDayFrom1970WithCfg(0) + day_count)
	return
}

func (this *Player) AddPlayerActMsg(msg *msg_client_message.ActivityInfo) {
	if nil == msg {
		log.Error("Player AddPlayerActMsg msg nil !")
		return
	}

	this.msg_acts_lock.Lock()
	defer this.msg_acts_lock.Unlock()

	if this.cur_msg_acts_len >= this.max_msg_acts_len {
		new_max := this.max_msg_acts_len + DEFAULT_PLAYER_MSG_ACT_ARRAY_STEP
		new_msgs := make([]*msg_client_message.ActivityInfo, 0, new_max)
		for idx := int32(0); idx < this.max_msg_acts_len; idx++ {
			new_msgs = append(new_msgs, this.msg_acts[idx])
		}

		this.msg_acts = new_msgs
		this.max_msg_acts_len = new_max
	}

	this.msg_acts = append(this.msg_acts, msg)
	this.cur_msg_acts_len++
}

func (this *Player) PopPlayerActMsg() *msg_client_message.S2CActivityInfosUpdate {
	this.msg_acts_lock.Lock()
	defer this.msg_acts_lock.Unlock()

	if this.cur_msg_acts_len > 0 {
		ret_msg := &msg_client_message.S2CActivityInfosUpdate{}
		ret_msg.Activityinfos = make([]*msg_client_message.ActivityInfo, 0, this.cur_msg_acts_len)
		for idx := int32(0); idx < this.cur_msg_acts_len; idx++ {
			ret_msg.Activityinfos = append(ret_msg.Activityinfos, this.msg_acts[idx])
		}

		this.cur_msg_acts_len = 0
		return ret_msg
	}

	return nil
}

func (this *Player) ChkUpdatePlayerActivity() {

	cur_unix_day := timer.GetDayFrom1970WithCfg(0)
	cur_month := int32(time.Now().Month())
	cur_month_day := int32(time.Now().Day())
	for _, task_cfg := range activity_old_table_mgr.Array {
		if !this.IfActOpen(task_cfg) {
			continue
		}

		switch task_cfg.ActivityType {
		case PLAYER_ACTIVITY_TYPE_DAY_REWARD:
			{
				if !this.db.Activitys.IfHaveAct(task_cfg.CfgId) {
					v := &dbPlayerActivityData{}
					v.CfgId = task_cfg.CfgId
					v.States = make([]int32, 3)
					v.States[0] = PLAYER_ACTIVITY_STATE_FINISHED
					v.States[1] = int32(cur_month)
					v.States[2] = cur_month_day
					this.db.Activitys.Add(v)
				} else {
					if this.db.Activitys.GetStates1(task_cfg.CfgId) != int32(cur_month) {
						this.db.Activitys.SetStates1(task_cfg.CfgId, cur_month)
						this.db.Activitys.ClearVals(task_cfg.CfgId)
					}
					if this.db.Activitys.GetStates2(task_cfg.CfgId) != cur_month_day {
						this.db.Activitys.SetStates0(task_cfg.CfgId, PLAYER_ACTIVITY_STATE_FINISHED)
						this.db.Activitys.SetStates2(task_cfg.CfgId, cur_month_day)
					}
				}
			}
		case PLAYER_ACTIVITY_TYPE_VIP_CARD:
			{
				if cur_unix_day > this.db.Info.GetVipCardEndDay() {
					if this.db.Activitys.IfHaveAct(task_cfg.CfgId) {
						this.db.Activitys.Remove(task_cfg.CfgId)
					}
				} else {
					if !this.db.Activitys.IfHaveAct(task_cfg.CfgId) {
						v := &dbPlayerActivityData{}
						v.CfgId = task_cfg.CfgId
						v.Vals = make([]int32, 1)
						v.Vals[0] = cur_unix_day
						v.States = make([]int32, 1)
						v.States[0] = PLAYER_ACTIVITY_STATE_FINISHED
						this.db.Activitys.Add(v)
					} else {
						if this.db.Activitys.GetVals0(task_cfg.CfgId) != cur_unix_day {
							this.db.Activitys.SetStates0(task_cfg.CfgId, PLAYER_ACTIVITY_STATE_FINISHED)
							this.db.Activitys.SetVals0(task_cfg.CfgId, cur_unix_day)
						}
					}
				}
			}
		case PLAYER_ACTIVITY_TYPE_LVL_REWARD:
			{
				log.Info("检测到等级奖励活动[%d]", task_cfg.CfgId)
				if !this.db.Activitys.IfHaveAct(task_cfg.CfgId) {
					log.Info("添加等级奖励活动[%d]", task_cfg.CfgId)
					v := &dbPlayerActivityData{}
					v.CfgId = task_cfg.CfgId
					this.db.Activitys.Add(v)
				}

				log.Info("all %v", this.db.Activitys.GetAll())
			}
		}
	}

	log.Info("all1 %v", this.db.Activitys.GetAll())
}

func (this *Player) OnActivityValAdd(task_type, val int32) {
	var act_db *dbPlayerActivityData
	cur_unix_day := timer.GetDayFrom1970WithCfg(0)
	for _, val := range activity_old_table_mgr.Array {
		if task_type != val.ActivityType {
			continue
		}

		if !this.IfActOpen(val) {
			continue
		}

		switch task_type {
		case PLAYER_ACTIVITY_TYPE_FIRST_PAY:
			{
				act_db = this.db.Activitys.Get(val.CfgId)
				if nil == act_db {
					act_db = &dbPlayerActivityData{}
					act_db.CfgId = val.CfgId
					act_db.States = make([]int32, 1)
					act_db.States[0] = PLAYER_FIRST_PAY_ACT
					this.db.Activitys.Add(act_db)
					log.Info("OnActivityValAdd PLAYER_ACTIVITY_TYPE_FIRST_PAY %v", *val)
				}
			}
		case PLAYER_ACTIVITY_TYPE_DAY_REWARD:
			{
				act_db = this.db.Activitys.Get(val.CfgId)
				if nil == act_db {
					act_db = &dbPlayerActivityData{}
					act_db.CfgId = val.CfgId
					act_db.States = make([]int32, 1)
					act_db.States[0] = PLAYER_ACTIVITY_STATE_FINISHED
					act_db.Vals = make([]int32, 1)
					this.db.Activitys.Add(act_db)
					log.Info("OnActivityValAdd PLAYER_ACTIVITY_TYPE_FIRST_PAY %v", *val)
				} else {
					if cur_unix_day != this.db.Activitys.GetVals0(val.CfgId) {
						this.db.Activitys.SetVals0(val.CfgId, cur_unix_day)
						this.db.Activitys.SetStates0(val.CfgId, PLAYER_ACTIVITY_STATE_FINISHED)
					}
				}
			}
		case PLAYER_ACTIVITY_TYPE_SUM_DAY_REWARD:
			{
				act_db = this.db.Activitys.Get(val.CfgId)
				if nil == act_db {
					act_db = &dbPlayerActivityData{}
					act_db.CfgId = val.CfgId
					act_db.States = make([]int32, 1)
					act_db.States[0] = 1
					this.db.Activitys.Add(act_db)
					log.Info("OnActivityValAdd PLAYER_ACTIVITY_TYPE_FIRST_PAY %v", *val)
				} else {
					this.db.Activitys.IncbyStates0(val.CfgId, 1)
				}
			}
		}
	}
}

func (this *Player) OnActivityValSet(task_type, val int32) {
	var act_db *dbPlayerActivityData
	for _, val := range activity_old_table_mgr.Array {
		if task_type != val.ActivityType {
			continue
		}

		if !this.IfActOpen(val) {
			continue
		}

		switch task_type {
		case PLAYER_ACTIVITY_TYPE_FIRST_PAY:
			{
				act_db = this.db.Activitys.Get(val.CfgId)
				if nil == act_db {
					act_db = &dbPlayerActivityData{}
					act_db.CfgId = val.CfgId
					act_db.States = make([]int32, 1)
					act_db.States[0] = PLAYER_FIRST_PAY_ACT
					this.db.Activitys.Add(act_db)

					act_msg := &msg_client_message.ActivityInfo{}
					act_msg.CfgId = val.CfgId
					act_msg.States = act_db.States
					this.AddPlayerActMsg(act_msg)
				}
			}
		}
	}
}

func (this *Player) GetActReward(act_cfg *tables.XmlActivityOldItem, extras []int32) int32 {

	if nil == act_cfg {
		log.Error("Player GetActReward act_cfg nil !")
		return -1
	}

	act_id := act_cfg.CfgId
	if !this.IfActOpen(act_cfg) {
		return int32(msg_client_message.E_ERR_ACTIVITY_NOT_OPEN)
	}

	act_db := this.db.Activitys.Get(act_id)

	rewards := act_cfg.Rewards

	log.Info("Player GetActReward [%v]", *act_cfg)

	cur_month_day := int32(time.Now().Day())

	switch act_cfg.ActivityType {
	case PLAYER_ACTIVITY_TYPE_FIRST_PAY:
		{
			if nil == act_db || len(act_db.States) < 1 {
				return int32(msg_client_message.E_ERR_ACTIVITY_NOT_FINISHED)
			}

			if PLAYER_FIRST_PAY_NOT_ACT == act_db.States[0] {
				return int32(msg_client_message.E_ERR_ACTIVITY_NOT_FINISHED)
			}

			if PLAYER_FIRST_PAY_REWARDED == act_db.States[0] {
				return int32(msg_client_message.E_ERR_ACTIVITY_HAVE_REWARDED)
			}

			this.db.Activitys.SetStates0(act_id, PLAYER_ACTIVITY_STATE_REWARDED)
		}
	case PLAYER_ACTIVITY_TYPE_DAY_REWARD:
		{
			if nil == act_db || len(act_db.States) < 1 {
				return int32(msg_client_message.E_ERR_ACTIVITY_NOT_FINISHED)
			}

			if 0 == act_db.States[0] {
				return int32(msg_client_message.E_ERR_ACTIVITY_NOT_FINISHED)
			}

			if 2 == act_db.States[0] {
				return int32(msg_client_message.E_ERR_ACTIVITY_HAVE_REWARDED)
			}

			day_r_cfg := activity_old_table_mgr.Day2Reward[cur_month_day]

			if nil == day_r_cfg {
				log.Info("没有着地第%d天的奖励！", cur_month_day)
				return int32(msg_client_message.E_ERR_ACTIVITY_DAY_REWARD_NO_CFG)
			}

			this.db.Activitys.SetStates0(act_id, PLAYER_ACTIVITY_STATE_REWARDED)
			this.db.Activitys.AddValsVal(act_id, cur_month_day)
			this.OnActivityValAdd(PLAYER_ACTIVITY_TYPE_SUM_DAY_REWARD, 1)
			this.GetSumDayRewardUntilNow()

			rewards = day_r_cfg.Rewards
		}
	case PLAYER_ACTIVITY_TYPE_LVL_REWARD:
		{
			if nil == act_db {
				return int32(msg_client_message.E_ERR_ACTIVITY_NOT_FINISHED)
			}

			if len(extras) < 1 {
				return int32(msg_client_message.E_ERR_ACTIVITY_GET_REWARD_REQ_ERROR)
			}

			lvl := extras[0]
			if lvl > this.db.GetLevel() {
				return int32(msg_client_message.E_ERR_ACTIVITY_LVL_REWARD_LESS_LVL)
			}
			lvl_r_cfg := activity_old_table_mgr.Lvl2Reward[lvl]
			if nil == lvl_r_cfg {
				log.Info("未找到等级[%d]的奖励", lvl)
				return int32(msg_client_message.E_ERR_ACTIVITY_LVL_REWARD_NO_CFG)
			}

			log.Info("等级[%v]奖励[%v]", lvl_r_cfg.Lvl, lvl_r_cfg.Rewards)

			if this.db.Activitys.IfStatesHave(act_id, lvl) {
				return int32(msg_client_message.E_ERR_ACTIVITY_HAVE_REWARDED)
			}

			this.db.Activitys.AddStateVal(act_id, lvl, true)

			rewards = lvl_r_cfg.Rewards
		}
	case PLAYER_ACTIVITY_TYPE_VIP_CARD:
		{
			if nil == act_db || len(act_db.States) < 1 {
				return int32(msg_client_message.E_ERR_ACTIVITY_NOT_FINISHED)
			}

			if len(extras) < 1 {
				return int32(msg_client_message.E_ERR_ACTIVITY_GET_REWARD_REQ_ERROR)
			}

			cur_unix_day := timer.GetDayFrom1970WithCfg(0)
			if cur_unix_day > this.db.Info.GetVipCardEndDay() {
				return int32(msg_client_message.E_ERR_ACTIVITY_VIPCARD_NOT_OPEN)
			}

			if len(act_db.Vals) >= 1 && act_db.Vals[0] == cur_unix_day {
				return int32(msg_client_message.E_ERR_ACTIVITY_HAVE_REWARDED)
			}

			new_vals := make([]int32, 1)
			new_vals[0] = cur_unix_day
			new_states := make([]int32, 1)
			new_states[0] = PLAYER_ACTIVITY_STATE_REWARDED
			this.db.Activitys.SetVals(act_id, new_vals)
		}
	case PLAYER_ACTIVITY_TYPE_SUM_DAY_REWARD:
		{
			if nil == act_db {
				return int32(msg_client_message.E_ERR_ACTIVITY_NOT_FINISHED)
			}

			if len(extras) < 1 {
				return int32(msg_client_message.E_ERR_ACTIVITY_GET_REWARD_REQ_ERROR)
			}

			sum_day := extras[0]
			sumd_r_cfg := activity_old_table_mgr.SumDay2Reward[sum_day]
			if nil == sumd_r_cfg {
				return int32(msg_client_message.E_ERR_ACTIVITY_SUM_DAYREWARD_NO_CFG)
			}

			if this.db.Activitys.IfValsHave(act_id, sum_day) {
				return int32(msg_client_message.E_ERR_ACTIVITY_HAVE_REWARDED)
			}

			this.db.Activitys.AddValsVal(act_id, sum_day)

			rewards = sumd_r_cfg.Rewards
		}
	}

	reward_count := int32(len(rewards))
	if reward_count < 2 {
		return int32(msg_client_message.E_ERR_ACTIVITY_NO_REWARDED)
	}

	switch act_cfg.RewardWay {
	case PLAYER_ACTIVITY_REWARD_WAY_MAIL:
		{
			//this.SendRewardMail(act_cfg.MallTitle, act_cfg.MallDescription, rewards, false)
		}
	case PLAYER_ACTIVITY_REWARD_WAY_DIRECT:
		{
			res2cil := &msg_client_message.S2CRetActivityReward{}
			res2cil.ActivityCfgId = act_id
			res2cil.Rewards = make([]*msg_client_message.IdNum, 0, reward_count/2)

			var tmp_idnum *msg_client_message.IdNum
			for idx := int32(0); idx+1 < reward_count; idx += 2 {
				tmp_idnum = &msg_client_message.IdNum{}
				tmp_idnum.Id = rewards[idx]
				tmp_idnum.Num = rewards[idx+1]

				this.AddObj(tmp_idnum.Id, tmp_idnum.Num, "get_activity_reward", "Activity", false)
				res2cil.Rewards = append(res2cil.Rewards, tmp_idnum)
			}
			this.SendItemsUpdate()
			this.SendCatsUpdate()
			this.SendDepotBuildingUpdate()

			this.Send(uint16(msg_client_message.S2CRetActivityReward_ProtoID), res2cil)
		}
	}

	act_db = this.db.Activitys.Get(act_cfg.CfgId)
	if nil != act_db {
		res2cli := &msg_client_message.S2CActivityInfosUpdate{}
		res2cli.Activityinfos = make([]*msg_client_message.ActivityInfo, 1)
		tmp_actinfo := &msg_client_message.ActivityInfo{}
		tmp_actinfo.CfgId = act_cfg.CfgId
		tmp_actinfo.States = act_db.States
		tmp_actinfo.Vals = act_db.Vals
		res2cli.Activityinfos[0] = tmp_actinfo

		this.Send(uint16(msg_client_message.S2CActivityInfosUpdate_ProtoID), res2cli)
	}

	return 1

}

func (this *Player) ChkRewardAct(task_type int32) {
	for _, val := range activity_old_table_mgr.Array {
		if task_type != val.ActivityType {
			continue
		}

		this.GetActReward(val, nil)
	}
}

func (this *Player) GetSumDayRewardUntilNow() {
	extras := make([]int32, 1)
	for _, val := range activity_old_table_mgr.Array {
		if val.ActivityType != PLAYER_ACTIVITY_TYPE_SUM_DAY_REWARD {
			continue
		}

		cur_sum_day := this.db.Activitys.GetStates0(val.CfgId)
		if cur_sum_day <= 0 {
			continue
		}

		for day := int32(1); day <= cur_sum_day; day++ {
			extras[0] = day
			log.Info("尝试获取累计天数[%d]的奖励", day)
			this.GetActReward(val, extras)
		}
	}
}

// ----------------------------------------------------------------------------

func C2SGetAllActivityInfosHandler(p *Player, msg_data []byte) int32 {
	p.ChkUpdatePlayerActivity()
	log.Info("all2 %v", p.db.Activitys.GetAll())
	res2cil := p.db.Activitys.FillAllClientMsg(p.db.Info.GetVipCardEndDay() - timer.GetDayFrom1970WithCfg(0))
	if nil != res2cil {
		p.Send(uint16(msg_client_message.S2CActivityInfosUpdate_ProtoID), res2cil)
		return 1
	}

	return 0
}

func C2SGetActivityRewardHandler(p *Player, msg_data []byte) int32 {
	p.ChkUpdatePlayerActivity()

	var req msg_client_message.C2SGetActivityReward
	err := proto.Unmarshal(msg_data, &req)
	if err != nil {
		return -1
	}
	act_id := req.GetActivityCfgId()
	act_cfg := activity_old_table_mgr.Map[act_id]
	if nil == act_cfg {
		return int32(msg_client_message.E_ERR_ACTIVITY_NO_CFG)
	}

	/*
		if !p.IfActOpen(act_cfg) {
			return int32(msg_client_message.E_ERR_ACTIVITY_NOT_OPEN)
		}

		act_db := p.db.Activitys.Get(act_id)

		rewards := act_cfg.Rewards

		switch act_cfg.ActivityType {
		case PLAYER_ACTIVITY_TYPE_FIRST_PAY:
			{
				if nil == act_db || len(act_db.States) < 1 {
					return int32(msg_client_message.E_ERR_ACTIVITY_NOT_FINISHED)
				}

				if PLAYER_FIRST_PAY_NOT_ACT == act_db.States[0] {
					return int32(msg_client_message.E_ERR_ACTIVITY_NOT_FINISHED)
				}

				if PLAYER_FIRST_PAY_REWARDED == act_db.States[0] {
					return int32(msg_client_message.E_ERR_ACTIVITY_HAVE_REWARDED)
				}

				p.db.Activitys.SetStates0(act_id, PLAYER_ACTIVITY_STATE_REWARDED)
			}
		case PLAYER_ACTIVITY_TYPE_DAY_REWARD:
			{
				if nil == act_db || len(act_db.States) < 1 {
					return int32(msg_client_message.E_ERR_ACTIVITY_NOT_FINISHED)
				}

				if 0 == act_db.States[0] {
					return int32(msg_client_message.E_ERR_ACTIVITY_NOT_FINISHED)
				}

				if 2 == act_db.States[0] {
					return int32(msg_client_message.E_ERR_ACTIVITY_HAVE_REWARDED)
				}

				p.db.Activitys.SetStates0(act_id, PLAYER_ACTIVITY_STATE_REWARDED)
				//p.db.Activitys.AddValsVal(act_id, timer.GetDayFrom1970WithCfg(0))
				p.OnActivityValAdd(PLAYER_ACTIVITY_TYPE_SUM_DAY_REWARD, 1)
				p.ChkRewardAct(PLAYER_ACTIVITY_TYPE_SUM_DAY_REWARD)

			}
		case PLAYER_ACTIVITY_TYPE_LVL_REWARD:
			{
				if nil == act_db {
					return int32(msg_client_message.E_ERR_ACTIVITY_NOT_FINISHED)
				}

				extras := req.GetExtraParams()
				if len(extras) < 1 {
					return int32(msg_client_message.E_ERR_ACTIVITY_GET_REWARD_REQ_ERROR)
				}

				lvl := extras[0]
				if lvl > p.db.Info.GetLvl() {
					return int32(msg_client_message.E_ERR_ACTIVITY_LVL_REWARD_LESS_LVL)
				}
				lvl_r_cfg := cfg_activity_mgr.Lvl2Reward[lvl]
				if nil == lvl_r_cfg {
					log.Info("未找到等级[%d]的奖励", lvl)
					return int32(msg_client_message.E_ERR_ACTIVITY_LVL_REWARD_NO_CFG)
				}

				log.Info("等级[%v]奖励[%v]", lvl_r_cfg.Lvl, lvl_r_cfg.Rewards)

				if p.db.Activitys.IfStatesHave(act_id, lvl) {
					return int32(msg_client_message.E_ERR_ACTIVITY_HAVE_REWARDED)
				}

				p.db.Activitys.AddStateVal(act_id, lvl, true)

				rewards = lvl_r_cfg.Rewards
			}
		case PLAYER_ACTIVITY_TYPE_VIP_CARD:
			{
				if nil == act_db || len(act_db.States) < 1 {
					return int32(msg_client_message.E_ERR_ACTIVITY_NOT_FINISHED)
				}

				extras := req.GetExtraParams()
				if len(extras) < 1 {
					return int32(msg_client_message.E_ERR_ACTIVITY_GET_REWARD_REQ_ERROR)
				}

				day := extras[0]
				day_r_cfg := cfg_activity_mgr.Day2Reward[day]

				if nil == day_r_cfg {
					return int32(msg_client_message.E_ERR_ACTIVITY_DAY_REWARD_NO_CFG)
				}

				cur_unix_day := timer.GetDayFrom1970WithCfg(0)
				if cur_unix_day > p.db.Info.GetVipCardEndDay() {
					return int32(msg_client_message.E_ERR_ACTIVITY_VIPCARD_NOT_OPEN)
				}

				if len(act_db.Vals) >= 1 && act_db.Vals[0] == cur_unix_day {
					return int32(msg_client_message.E_ERR_ACTIVITY_HAVE_REWARDED)
				}

				new_vals := make([]int32, 1)
				new_vals[0] = cur_unix_day
				new_states := make([]int32, 1)
				new_states[0] = PLAYER_ACTIVITY_STATE_REWARDED
				p.db.Activitys.SetVals(act_id, new_vals)
			}
		case PLAYER_ACTIVITY_TYPE_SUM_DAY_REWARD:
			{
				if nil == act_db {
					return int32(msg_client_message.E_ERR_ACTIVITY_NOT_FINISHED)
				}

				extras := req.GetExtraParams()
				if len(extras) < 1 {
					return int32(msg_client_message.E_ERR_ACTIVITY_GET_REWARD_REQ_ERROR)
				}

				sum_day := extras[0]
				sumd_r_cfg := act_cfg.SumDayReward.SumDay2Reward[sum_day]
				if nil == sumd_r_cfg {
					return int32(msg_client_message.E_ERR_ACTIVITY_SUM_DAYREWARD_NO_CFG)
				}

				if p.db.Activitys.IfStatesHave(act_id, sum_day) {
					return int32(msg_client_message.E_ERR_ACTIVITY_HAVE_REWARDED)
				}

				p.db.Activitys.AddStateVal(act_id, sum_day, true)

				rewards = sumd_r_cfg.Rewards
			}
		}

		reward_count := int32(len(rewards))
		if reward_count < 2 {
			return int32(msg_client_message.E_ERR_ACTIVITY_NO_REWARDED)
		}

		switch act_cfg.RewardWay {
		case PLAYER_ACTIVITY_REWARD_WAY_MAIL:
			{
				p.SendRewardMail(act_cfg.MallTitle, act_cfg.MallDescription, rewards, false)
			}
		case PLAYER_ACTIVITY_REWARD_WAY_DIRECT:
			{
				res2cil := &msg_client_message.S2CRetActivityReward{}
				res2cil.ActivityCfgId = proto.Int32(act_id)
				res2cil.Rewards = make([]*msg_client_message.IdNum, 0, reward_count/2)

				var tmp_idnum *msg_client_message.IdNum
				for idx := int32(0); idx+1 < reward_count; idx += 2 {
					tmp_idnum = &msg_client_message.IdNum{}
					tmp_idnum.Id = proto.Int32(rewards[idx])
					tmp_idnum.Num = proto.Int32(rewards[idx+1])

					p.AddObj(*tmp_idnum.Id, *tmp_idnum.Num, "get_activity_reward", "Activity", true)
					res2cil.Rewards = append(res2cil.Rewards, tmp_idnum)
				}

				p.Send(res2cil)
			}
		}
	*/

	return p.GetActReward(act_cfg, req.GetExtraParams())
}
