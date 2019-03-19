package tables

import (
	"encoding/xml"
	"io/ioutil"
	"mm_server/libs/log"
	"mm_server/src/server_config"
)

// 任务类型
const (
	TASK_TYPE_DAILY   = 1 // 日常任务
	TASK_TYPE_ACHIEVE = 2 // 成就任务
)

// 任务完成类型
const (
	// 每日任务和成就任务(参数 EventParam CompleteNum)
	TASK_COMPLETE_TYPE_ALL_DAILY          = 101 // (EventParam 0, CompleteNum 0) 完成所有每日任务
	TASK_COMPLETE_TYPE_PASS_NUM           = 102 // (EventParam 0, CompleteNum N) 挑战N次消除关卡
	TASK_COMPLETE_TYPE_WEEK_MATCH_NUM     = 103 // (EventParam 0, CompleteNum N) 挑战周赛N次
	TASK_COMPLETE_TYPE_EXPLORE_NUM        = 104 // (EventParam 0, CompleteNum N) 完成N个探索任务
	TASK_COMPLETE_TYPE_MAKING_FORMULA_NUM = 105 // (EventParam 0, CompleteNum N) 完成N个装饰物打造
	TASK_COMPLETE_TYPE_CAT_FEED           = 106 // (EventParam 0, CompleteNum N) 喂食N只猫
	TASK_COMPLETE_TYPE_GET_EXP_BY_FOSTER  = 107 // (EventParam 0, CompleteNum N) 收取N次寄养经验
	TASK_COMPLETE_TYPE_GIVE_FRIEND_POINT  = 108 // (EventParam 0, CompleteNum N) 赠送N次友情点
	TASK_COMPLETE_TYPE_VISIT_FRIEND_NUM   = 109 // (EventParam 0, CompleteNum N) 拜访N次好友家园

	TASK_COMPLETE_TYPE_PASS_CHAPTER                = 201 // (EventParam A, CompleteNum B) 通关A章节B次
	TASK_COMPLETE_TYPE_COLLECT_STAR_NUM            = 202 // (EventParam A, CompleteNum 0) 收集A颗消除星星B次
	TASK_COMPLETE_TYPE_EXPLORE_TASK_NUM            = 203 // (EventParam 0, CompleteNum B) 探索任务成功B次
	TASK_COMPLETE_TYPE_MAKING_FORMULA_BUILDING_NUM = 204 // (EventParam A, CompleteNum B) 完成A阶饰物打造B次
	TASK_COMPLETE_TYPE_COLLECT_SSR                 = 205 // (EventParam A, CompleteNum B) 收集SSR B只
	TASK_COMPLETE_TYPE_CAT_LEVEL_UP                = 206 // (EventParam A, CompleteNum B) B只猫升到A级
	TASK_COMPLETE_TYPE_CAT_UP_STAR                 = 207 // (EventParam A, CompleteNum B) B只猫升到A星
	TASK_COMPLETE_TYPE_CAT_UP_SKILL_LEVEL          = 208 // (EventParam A, CompleteNum B) B只猫技能升到A级
	TASK_COMPLETE_TYPE_CHARM_VALUE                 = 209 // (EventParam A, CompleteNum B) 魅力值A达到B次
	TASK_COMPLETE_TYPE_WON_PRAISE                  = 210 // (EventParam A, CompleteNum B) 获得A个赞B次
	TASK_COMPLETE_TYPE_OPEN_FRIEND_TREATURE_BOX    = 211 // (EventParam 0, CompleteNum B) 开启B次好友宝箱
)

type TaskReward struct {
	ItemId int32
	Num    int32
}

type XmlTaskItem struct {
	Id          int32  `xml:"Id,attr"`
	Type        int32  `xml:"Type,attr"`
	EventId     int32  `xml:"EventId,attr"`
	EventParam  int32  `xml:"EventParam,attr"`
	CompleteNum int32  `xml:"CompleteNum,attr"`
	Prev        int32  `xml:"Prev,attr"`
	Next        int32  `xml:"Next,attr"`
	MinLevel    int32  `xml:"MinLevel,attr"`
	MaxLevel    int32  `xml:"Maxlevel,attr"`
	Exp         int32  `xml:"Exp,attr"`
	RewardStr   string `xml:"Reward,attr"`
	Rewards     []int32
}

type XmlTaskTable struct {
	Items []XmlTaskItem `xml:"item"`
}

type FinishTypeTasks struct {
	count int32
	array []*XmlTaskItem
}

func (this *FinishTypeTasks) GetCount() int32 {
	return this.count
}

func (this *FinishTypeTasks) GetArray() []*XmlTaskItem {
	return this.array
}

type TaskTableMgr struct {
	task_map          map[int32]*XmlTaskItem     // 任务map
	task_array        []*XmlTaskItem             // 任务数组
	task_array_len    int32                      // 数组长度
	finish_tasks      map[int32]*FinishTypeTasks // 按完成条件组织任务数据
	daily_task_map    map[int32]*XmlTaskItem     // 日常任务MAP
	daily_task_array  []*XmlTaskItem             // 日常任务数组
	all_daily_task    *XmlTaskItem               // 所有日常任务
	achieve_tasks_map map[int32]*XmlTaskItem     // 成就任务MAP
	achieve_tasks     []*XmlTaskItem             // 初始成就任务
}

func (this *TaskTableMgr) Init(table_file string) bool {
	if table_file == "" {
		table_file = "mission.xml"
	}
	file_path := server_config.GetGameDataPathFile(table_file)
	content, err := ioutil.ReadFile(file_path)
	if nil != err {
		log.Error("TaskTableMgr LoadTask read file error !")
		return false
	}

	tmp_cfg := &XmlTaskTable{}
	err = xml.Unmarshal(content, tmp_cfg)
	if nil != err {
		log.Error("TaskTableMgr LoadTask unmarshal failed(%s)", err.Error())
		return false
	}

	tmp_len := int32(len(tmp_cfg.Items))

	this.task_array = make([]*XmlTaskItem, 0, tmp_len)
	this.task_map = make(map[int32]*XmlTaskItem)
	this.finish_tasks = make(map[int32]*FinishTypeTasks)
	this.daily_task_map = make(map[int32]*XmlTaskItem)
	this.achieve_tasks_map = make(map[int32]*XmlTaskItem)

	var tmp_item *XmlTaskItem
	for idx := int32(0); idx < tmp_len; idx++ {
		tmp_item = &tmp_cfg.Items[idx]

		rewards := parse_xml_str_arr2(tmp_item.RewardStr, ",")
		if rewards != nil && len(rewards)%2 != 0 {
			log.Error("@@@@@@ Task[%v] Reward[%v] invalid", tmp_item.Id, tmp_item.RewardStr)
			return false
		}

		tmp_item.Rewards = rewards
		if tmp_item.EventId != TASK_COMPLETE_TYPE_ALL_DAILY && tmp_item.CompleteNum <= 0 {
			tmp_item.CompleteNum = 1
		}

		this.task_map[tmp_item.Id] = tmp_item
		this.task_array = append(this.task_array, tmp_item)
		if nil == this.finish_tasks[tmp_item.EventId] {
			this.finish_tasks[tmp_item.EventId] = &FinishTypeTasks{}
		}
		this.finish_tasks[tmp_item.EventId].count++
		if tmp_item.Type == TASK_TYPE_DAILY {
			this.daily_task_map[tmp_item.Id] = tmp_item
			this.daily_task_array = append(this.daily_task_array, tmp_item)
			if tmp_item.EventId == TASK_COMPLETE_TYPE_ALL_DAILY {
				this.all_daily_task = tmp_item
			}
		} else if tmp_item.Type == TASK_TYPE_ACHIEVE {
			this.achieve_tasks = append(this.achieve_tasks, tmp_item)
			this.achieve_tasks_map[tmp_item.Id] = tmp_item
		}
	}
	for idx := int32(0); idx < tmp_len; idx++ {
		tmp_item = &tmp_cfg.Items[idx]
		if nil == this.finish_tasks[tmp_item.EventId].array {
			this.finish_tasks[tmp_item.EventId].array = make([]*XmlTaskItem, 0, this.finish_tasks[tmp_item.EventId].count)
		}
		this.finish_tasks[tmp_item.EventId].array = append(this.finish_tasks[tmp_item.EventId].array, tmp_item)
	}

	this.task_array_len = int32(len(this.task_array))

	// 所有日常任务CompleteNum处理
	if this.all_daily_task != nil {
		for _, d := range this.daily_task_map {
			if d.EventId != TASK_COMPLETE_TYPE_ALL_DAILY {
				this.all_daily_task.CompleteNum += 1
			}
		}
	}

	log.Info("TaskTableMgr Loaded Task table")

	return true

	return true
}

func (this *TaskTableMgr) GetTaskMap() map[int32]*XmlTaskItem {
	return this.task_map
}

func (this *TaskTableMgr) GetTask(task_id int32) *XmlTaskItem {
	if this.task_map == nil {
		return nil
	}
	return this.task_map[task_id]
}

func (this *TaskTableMgr) GetWholeDailyTask() *XmlTaskItem {
	return this.all_daily_task
}

func (this *TaskTableMgr) GetFinishTasks() map[int32]*FinishTypeTasks {
	return this.finish_tasks
}

func (this *TaskTableMgr) GetDailyTasks() map[int32]*XmlTaskItem {
	return this.daily_task_map
}

func (this *TaskTableMgr) GetAchieveTasks() []*XmlTaskItem {
	return this.achieve_tasks
}
func (this *TaskTableMgr) GetTasks(task_type int32) (tasks []*XmlTaskItem) {
	if task_type == TASK_TYPE_DAILY {
		tasks = this.daily_task_array
	} else if task_type == TASK_TYPE_ACHIEVE {
		tasks = this.achieve_tasks
	}
	return
}

func (this *TaskTableMgr) IsDaily(task_id int32) bool {
	if this.daily_task_map != nil {
		if this.daily_task_map[task_id] != nil {
			return true
		}
	}
	return false
}

func (this *TaskTableMgr) IsAchieve(task_id int32) bool {
	if this.achieve_tasks_map != nil {
		if this.achieve_tasks_map[task_id] != nil {
			return true
		}
	}
	return false
}
