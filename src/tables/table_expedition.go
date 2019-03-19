package tables

import (
	"encoding/xml"
	"io/ioutil"
	"math/rand"
	"mm_server/libs/log"
	"mm_server/src/server_config"
)

const (
	PLAYER_EXPEDITION_CON_CAT_LVL   = 1 // 猫咪等级条件
	PLAYER_EXPEDITION_CON_CAT_QUA   = 2 // 猫咪品阶条件
	PLAYER_EXPEDITION_CON_CAT_STAR  = 3 // 猫咪星阶条件
	PLAYER_EXPEDITION_CON_CAT_COLOR = 4 // 猫咪颜色条件
	PLAYER_EXPEDITION_CON_CAT_NUM   = 5 // 猫咪数量条件
)

const (
	PLAYER_EXPEDITION_TYPE_NORMAL    = 1
	PLAYER_EXPEDITION_TYPE_TIMELIMIT = 2

	PLAYER_EXPEDITION_STATE_INIT    = 0 // 探险任务初始状态
	PLAYER_EXPEDITION_STATE_DOING   = 1 // 探险任务探险中状态
	PLAYER_EXPEDITION_STATE_SUCCEED = 2 // 探险任务成功状态
	PLAYER_EXPEDITION_STATE_FAILED  = 3 // 探险任务失败状态

	PLAYER_EXPEDITION_RESULT_FAILED  = 0 // 探险任务结果失败
	PLAYER_EXPEDITION_RESULT_SUCCEED = 1 // 探险任务结果成功
)

type ExpeditionCondition struct {
	Con_Type   int32   // 条件类型
	Con_Val    int32   // 条件值
	Ext_val    int32   // 附加值
	Ext_vals   []int32 // 附加值
	Con_Weight int32   // 条件权重
}

type XmlExpeditionItem struct {
	Id              int32  `xml:"Id,attr"`
	CostTime        int32  `xml:"SearchTime,attr"`
	RandWeight      int32  `xml:"SearchWeight,attr"`
	TaskType        int32  `xml:"Type,attr"`
	LimitTimeSec    int32  `xml:"ValidTime,attr"`
	LvlScopeStr     string `xml:"SearchLv,attr"`
	LvlMin          int32
	LvlMax          int32
	FixRewardsStr   string `xml:"Reward,attr"`
	FixRewards      []int32
	FixRewardsNum   int32
	SucceedBaseRate int32 `xml:"SearchChance,attr"`
	SearchEventId   int32 `xml:"SearchEventId,attr"`
	EventBaseRate   int32 `xml:"SearchEventChance,attr"`

	NeedConditionNum  int32  `xml:"RequireNum,attr"`
	Lv                int32  `xml:"Lv,attr"`
	LvWeight          int32  `xml:"LvWeight,attr"`
	Quality           int32  `xml:"Quality,attr"`
	QualityWeight     int32  `xml:"QualityWeight,attr"`
	Star              int32  `xml:"Star,attr"`
	StarWeight        int32  `xml:"StarWeight,attr"`
	ColorNumWeightStr string `xml:"ColorNumWeight,attr"`
	ColorWeight       int32  `xml:"ColorWeight,attr"`
	ColorNum          int32  `xml:"ColorNum,attr"`
	CatNum            int32  `xml:"CatNum,attr"`
	CatNumWeight      int32  `xml:"NumWeight,attr"`
	BuyBackStr        string `xml:"BuyBack,attr"`
	BuyBackCosts      []int32
	Single            int32 `xml:"Single,attr"`

	TotalConWeight    int32 // 总权重
	Conditions        []*ExpeditionCondition
	TotalConditionNum int32 // 条件数目
}

type XmlExpeditionItemConfig struct {
	Items []XmlExpeditionItem `xml:"item"`
}

type XmlExpeditionEventItem struct {
	SearchEventID     int32  `xml:"Id,attr"`
	ClientId          int32  `xml:"ClientId,attr"`
	SearchEventWeight int32  `xml:"SearchEventWeight,attr"`
	DropIdStr         string `xml:"DropId,attr"`
	DropIds           []int32
}

type XmlExpeditionEventConfig struct {
	Items []XmlExpeditionEventItem `xml:"item"`
}

type ExpeditionEvent struct {
	Array       []*XmlExpeditionEventItem
	Count       int32
	TotalWeight int32
}

type ExpeditionTableMgr struct {
	Map        map[int32]*XmlExpeditionItem
	Array      []*XmlExpeditionItem
	TotalCount int32

	Id2Event map[int32]*ExpeditionEvent
	EventMap map[int32]*XmlExpeditionEventItem
}

func (this *ExpeditionTableMgr) Init(table_file, event_file string) bool {
	if !this.LoadExpediton(table_file) {
		return false
	}

	if !this.LoadEvent(event_file) {
		return false
	}

	log.Info("当前所有事件")
	for id, event := range this.Id2Event {
		log.Info("	时间[%d] [%v]", id, *event)
	}

	return true
}

func (this *ExpeditionTableMgr) LoadExpediton(table_file string) bool {
	if table_file == "" {
		table_file = "SearchTask.xml"
	}
	file_path := server_config.GetGameDataPathFile(table_file)
	data, err := ioutil.ReadFile(file_path)
	if nil != err {
		log.Error("ExpeditionTableMgr Init read file err[%s] !", err.Error())
		return false
	}

	tmp_cfg := &XmlExpeditionItemConfig{}
	err = xml.Unmarshal(data, tmp_cfg)
	if nil != err {
		log.Error("ExpeditionTableMgr Init xml Unmarshal failed error [%s] !", err.Error())
		return false
	}

	this.Map = make(map[int32]*XmlExpeditionItem)
	this.Array = make([]*XmlExpeditionItem, 0, len(tmp_cfg.Items))
	tmp_len := int32(len(tmp_cfg.Items))
	var tmp_item *XmlExpeditionItem
	var tmp_con *ExpeditionCondition
	for idx := int32(0); idx < tmp_len; idx++ {
		tmp_item = &tmp_cfg.Items[idx]
		tmp_item.FixRewards = parse_xml_str_arr(tmp_item.FixRewardsStr, ",")
		tmp_item.FixRewardsNum = int32(len(tmp_item.FixRewards)) / 2
		if len(tmp_item.FixRewards)%2 != 0 {
			log.Error("ExpeditionTableMgr init FixRewardsStr[%s] error !", tmp_item.FixRewardsStr)
			return false
		}

		tmp_item.TotalConditionNum = 5
		tmp_item.Conditions = make([]*ExpeditionCondition, 0, tmp_item.TotalConditionNum)

		// 任务等级条件
		tmp_scope := parse_xml_str_arr(tmp_item.LvlScopeStr, ",")
		tmp_item.LvlMin = tmp_scope[0]
		tmp_item.LvlMax = tmp_scope[1]

		// 猫咪等级条件
		tmp_con = &ExpeditionCondition{}
		tmp_con.Con_Type = PLAYER_EXPEDITION_CON_CAT_LVL
		tmp_con.Con_Val = tmp_item.Lv
		tmp_con.Con_Weight = tmp_item.LvWeight
		tmp_item.TotalConWeight += tmp_item.LvWeight
		tmp_item.Conditions = append(tmp_item.Conditions, tmp_con)

		// 猫咪品阶条件
		tmp_con = &ExpeditionCondition{}
		tmp_con.Con_Type = PLAYER_EXPEDITION_CON_CAT_QUA
		tmp_con.Con_Val = tmp_item.Quality
		tmp_con.Con_Weight = tmp_item.QualityWeight
		tmp_item.TotalConWeight += tmp_item.QualityWeight
		tmp_item.Conditions = append(tmp_item.Conditions, tmp_con)

		// 猫咪星阶条件
		tmp_con = &ExpeditionCondition{}
		tmp_con.Con_Type = PLAYER_EXPEDITION_CON_CAT_STAR
		tmp_con.Con_Val = tmp_item.Star
		tmp_con.Con_Weight = tmp_item.StarWeight
		tmp_item.TotalConWeight += tmp_item.StarWeight
		tmp_item.Conditions = append(tmp_item.Conditions, tmp_con)

		// 猫咪颜色条件
		tmp_con = &ExpeditionCondition{}
		tmp_con.Con_Type = PLAYER_EXPEDITION_CON_CAT_COLOR
		tmp_con.Con_Val = tmp_item.ColorNum
		tmp_con.Con_Weight = tmp_item.ColorWeight
		tmp_con.Ext_vals = parse_xml_str_arr(tmp_item.ColorNumWeightStr, ",")
		for _, val := range tmp_con.Ext_vals {
			tmp_con.Ext_val += val
		}
		tmp_item.TotalConWeight += tmp_item.ColorWeight
		tmp_item.Conditions = append(tmp_item.Conditions, tmp_con)

		// 购买结果消耗
		tmp_item.BuyBackCosts = parse_xml_str_arr(tmp_item.BuyBackStr, ",")
		if 0 != len(tmp_item.BuyBackCosts)%2 {
			log.Error("CfgExpeditionMgr BuyBackStr[%s] error !", tmp_item.BuyBackStr)
			return false
		}

		log.Info("任务%d时间限制%d", tmp_item.Id, tmp_item.LimitTimeSec)

		this.Array = append(this.Array, tmp_item)
		this.Map[tmp_item.Id] = tmp_item
		this.TotalCount++
	}

	//log.Info("CfgExpeditionMgr total count %d info %v", this.TotalCount, this.Map)

	return true
}

func (this *ExpeditionTableMgr) LoadEvent(event_file string) bool {
	if event_file == "" {
		event_file = "SearchEvent.xml"
	}
	file_path := server_config.GetGameDataPathFile(event_file)
	data, err := ioutil.ReadFile(file_path)
	if nil != err {
		log.Error("ExpeditionTableMgr LoadEvent read file error(%s) !", err.Error())
		return false
	}

	tmp_cfg := &XmlExpeditionEventConfig{}
	err = xml.Unmarshal(data, tmp_cfg)
	if nil != err {
		log.Error("ExpeditionTableMgr loadEvent unmarshal failed (%s) !", err.Error())
		return false
	}

	tmp_len := int32(len(tmp_cfg.Items))

	this.Id2Event = make(map[int32]*ExpeditionEvent)
	this.EventMap = make(map[int32]*XmlExpeditionEventItem)
	var tmp_item *XmlExpeditionEventItem
	for idx := int32(0); idx < tmp_len; idx++ {
		tmp_item = &tmp_cfg.Items[idx]
		if nil == tmp_item || tmp_item.SearchEventWeight <= 0 {
			continue
		}

		if nil == this.Id2Event[tmp_item.SearchEventID] {
			this.Id2Event[tmp_item.SearchEventID] = &ExpeditionEvent{}
		}

		this.Id2Event[tmp_item.SearchEventID].Count++
		this.Id2Event[tmp_item.SearchEventID].TotalWeight += tmp_item.SearchEventWeight
		this.EventMap[tmp_item.SearchEventID] = tmp_item
	}

	for idx := int32(0); idx < tmp_len; idx++ {
		tmp_item = &tmp_cfg.Items[idx]
		if nil == tmp_item || tmp_item.SearchEventWeight <= 0 {
			continue
		}

		if nil == this.Id2Event[tmp_item.SearchEventID].Array {
			this.Id2Event[tmp_item.SearchEventID].Array = make([]*XmlExpeditionEventItem, 0, this.Id2Event[tmp_item.SearchEventID].Count)
		}

		tmp_item.DropIds = parse_xml_str_arr(tmp_item.DropIdStr, ",")
		if len(tmp_item.DropIds)%3 != 0 {
			log.Error("ExpeditionTableMgr LoadEvent dropid[%s] error !", tmp_item.DropIdStr)
			return false
		}

		this.Id2Event[tmp_item.SearchEventID].Array = append(this.Id2Event[tmp_item.SearchEventID].Array, tmp_item)

	}

	/*
		for event_id, val := range this.Id2Event {
			log.Info("===============================event[%d,%d,%d]", event_id, val.Count, val.TotalWeight)
			for _, tmp_val := range val.Array {
				log.Info("===task val %v !", *tmp_val)
			}
		}
	*/

	return true
}

func (this *ExpeditionTableMgr) RandNWithExistIds(cur_ids map[int32]bool, p_lvl, need_n int32) (ret_tasks []*XmlExpeditionItem) {
	if nil == cur_ids || need_n < 1 {
		log.Error("ExpeditionTableMgr RandNWithExistIds param error %v!", need_n)
		return nil
	}

	log.Info("ExpeditionTableMgr RandNWithExistIds %v %d %d", cur_ids, p_lvl, need_n)
	total_weight := int32(0)
	var val *XmlExpeditionItem
	for idx := int32(0); idx < this.TotalCount; idx++ {
		val = this.Array[idx]
		if nil == val || val.LvlMin > p_lvl || val.LvlMax < p_lvl {
			continue
		}

		if 1 == val.Single && cur_ids[val.Id] {
			continue
		}

		log.Info("add to rand lib %d", val.Id)

		total_weight += val.RandWeight
	}

	if total_weight <= 0 {
		log.Error("ExpeditionTableMgr RandNWithExistIds total_weight[%d] <= 0 !", total_weight)
		return nil
	}

	ret_tasks = make([]*XmlExpeditionItem, 0, need_n)

	var rand_val int32
	for i := int32(0); i < need_n; i++ {
		if total_weight <= 0 {
			break
		}
		rand_val = rand.Int31n(total_weight)

		log.Info("=======================rand_val[%d]=======================", rand_val)
		for idx := int32(0); idx < this.TotalCount; idx++ {
			val = this.Array[idx]
			if nil == val || val.LvlMin > p_lvl || val.LvlMax < p_lvl {
				continue
			}

			if 1 == val.Single && (cur_ids[val.Id]) {
				continue
			}

			if rand_val < val.RandWeight {
				ret_tasks = append(ret_tasks, val)
				cur_ids[val.Id] = true
				total_weight -= val.RandWeight
				break
			} else {
				rand_val -= val.RandWeight
			}

			log.Info("consider %d", val.Id)
		}
		log.Info("===========================end===========================")
	}

	return
}

func (this *ExpeditionTableMgr) RandEvent(eventid int32) *XmlExpeditionEventItem {
	event_cfg := this.Id2Event[eventid]
	if nil == event_cfg {
		log.Error("ExpeditionTableMgr RandEvent failed to find cfg [%d]!", eventid)
		log.Info("当前所有事件")
		for id, event := range this.Id2Event {
			log.Info("	时间[%d] [%v]", id, *event)
		}
		return nil
	} else {
		log.Info("EventCfg：[%d] [%d]", event_cfg.TotalWeight, event_cfg.Count)
		for _, val := range event_cfg.Array {
			log.Info(" event :%v", *val)
		}
	}

	rand_val := rand.Int31n(event_cfg.TotalWeight)
	log.Info("本次randval %d", rand_val)
	var tmp_item *XmlExpeditionEventItem
	for idx := int32(0); idx < event_cfg.Count; idx++ {
		tmp_item = event_cfg.Array[idx]
		if rand_val < tmp_item.SearchEventWeight {
			return tmp_item
		} else {
			rand_val -= tmp_item.SearchEventWeight
		}

		log.Info("随机检查 %v %d", *tmp_item, rand_val)
	}

	return nil
}
