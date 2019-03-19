package csv_readers

import (
	"encoding/csv"
	"io/ioutil"
	"log"
	"strconv"
	"strings"
)

type Skill struct {
	ID int32
	Type int32
	SkillAttr string
	IsCancelReport int32
	SkillTriggerType int32
	TriggerCondition1 string
	TriggerCondition2 string
	StunDisableAction int32
	IsDelayLastSkill int32
	TriggerRoundMax int32
	TriggerBattleMax int32
	SkillMelee int32
	SkillEnemy int32
	RangeType int32
	SkillTarget int32
	MaxTarget int32
	CertainHit int32
	Effect1Cond1 string
	Effect1Cond2 string
	Effect1 string
	Effect2Cond1 string
	Effect2Cond2 string
	Effect2 string
	Effect3Cond1 string
	Effect3Cond2 string
	Effect3 string
	ComboSKill int32
}

type SkillMgr struct {
	id2items map[int32]*Skill
	items_array []*Skill
}

func (this *SkillMgr) Read(file_path_name string) bool {
	if file_path_name == "" {
		file_path_name = "../game_csv/skill.csv"
	}
	cs, err := ioutil.ReadFile(file_path_name)
	if err != nil {
		log.Printf("SkillMgr.Read err: %v", err.Error())
		return false
	}

 	r := csv.NewReader(strings.NewReader(string(cs)))
	ss, _ := r.ReadAll()
   	sz := len(ss)
	this.id2items = make(map[int32]*Skill)
    for i := int32(1); i < int32(sz); i++ {
		//if i < 5 {
		//	continue
		//}
		var v Skill
		var intv, id int
		// ID
		intv, err = strconv.Atoi(ss[i][0])
		if err != nil {
			log.Printf("table Skill convert column ID value %v with row %v err %v", ss[i][0], 0, err.Error())
			return false
		}
		v.ID = int32(intv)
		id = intv
		// Type
		intv, err = strconv.Atoi(ss[i][1])
		if err != nil {
			log.Printf("table Skill convert column Type value %v with row %v err %v", ss[i][1], 1, err.Error())
			return false
		}
		v.Type = int32(intv)
		// SkillAttr
		v.SkillAttr = ss[i][2]
		// IsCancelReport
		intv, err = strconv.Atoi(ss[i][3])
		if err != nil {
			log.Printf("table Skill convert column IsCancelReport value %v with row %v err %v", ss[i][3], 3, err.Error())
			return false
		}
		v.IsCancelReport = int32(intv)
		// SkillTriggerType
		intv, err = strconv.Atoi(ss[i][4])
		if err != nil {
			log.Printf("table Skill convert column SkillTriggerType value %v with row %v err %v", ss[i][4], 4, err.Error())
			return false
		}
		v.SkillTriggerType = int32(intv)
		// TriggerCondition1
		v.TriggerCondition1 = ss[i][5]
		// TriggerCondition2
		v.TriggerCondition2 = ss[i][6]
		// StunDisableAction
		intv, err = strconv.Atoi(ss[i][7])
		if err != nil {
			log.Printf("table Skill convert column StunDisableAction value %v with row %v err %v", ss[i][7], 7, err.Error())
			return false
		}
		v.StunDisableAction = int32(intv)
		// IsDelayLastSkill
		intv, err = strconv.Atoi(ss[i][8])
		if err != nil {
			log.Printf("table Skill convert column IsDelayLastSkill value %v with row %v err %v", ss[i][8], 8, err.Error())
			return false
		}
		v.IsDelayLastSkill = int32(intv)
		// TriggerRoundMax
		intv, err = strconv.Atoi(ss[i][9])
		if err != nil {
			log.Printf("table Skill convert column TriggerRoundMax value %v with row %v err %v", ss[i][9], 9, err.Error())
			return false
		}
		v.TriggerRoundMax = int32(intv)
		// TriggerBattleMax
		intv, err = strconv.Atoi(ss[i][10])
		if err != nil {
			log.Printf("table Skill convert column TriggerBattleMax value %v with row %v err %v", ss[i][10], 10, err.Error())
			return false
		}
		v.TriggerBattleMax = int32(intv)
		// SkillMelee
		intv, err = strconv.Atoi(ss[i][11])
		if err != nil {
			log.Printf("table Skill convert column SkillMelee value %v with row %v err %v", ss[i][11], 11, err.Error())
			return false
		}
		v.SkillMelee = int32(intv)
		// SkillEnemy
		intv, err = strconv.Atoi(ss[i][12])
		if err != nil {
			log.Printf("table Skill convert column SkillEnemy value %v with row %v err %v", ss[i][12], 12, err.Error())
			return false
		}
		v.SkillEnemy = int32(intv)
		// RangeType
		intv, err = strconv.Atoi(ss[i][13])
		if err != nil {
			log.Printf("table Skill convert column RangeType value %v with row %v err %v", ss[i][13], 13, err.Error())
			return false
		}
		v.RangeType = int32(intv)
		// SkillTarget
		intv, err = strconv.Atoi(ss[i][14])
		if err != nil {
			log.Printf("table Skill convert column SkillTarget value %v with row %v err %v", ss[i][14], 14, err.Error())
			return false
		}
		v.SkillTarget = int32(intv)
		// MaxTarget
		intv, err = strconv.Atoi(ss[i][15])
		if err != nil {
			log.Printf("table Skill convert column MaxTarget value %v with row %v err %v", ss[i][15], 15, err.Error())
			return false
		}
		v.MaxTarget = int32(intv)
		// CertainHit
		intv, err = strconv.Atoi(ss[i][16])
		if err != nil {
			log.Printf("table Skill convert column CertainHit value %v with row %v err %v", ss[i][16], 16, err.Error())
			return false
		}
		v.CertainHit = int32(intv)
		// Effect1Cond1
		v.Effect1Cond1 = ss[i][17]
		// Effect1Cond2
		v.Effect1Cond2 = ss[i][18]
		// Effect1
		v.Effect1 = ss[i][19]
		// Effect2Cond1
		v.Effect2Cond1 = ss[i][20]
		// Effect2Cond2
		v.Effect2Cond2 = ss[i][21]
		// Effect2
		v.Effect2 = ss[i][22]
		// Effect3Cond1
		v.Effect3Cond1 = ss[i][23]
		// Effect3Cond2
		v.Effect3Cond2 = ss[i][24]
		// Effect3
		v.Effect3 = ss[i][25]
		// ComboSKill
		intv, err = strconv.Atoi(ss[i][26])
		if err != nil {
			log.Printf("table Skill convert column ComboSKill value %v with row %v err %v", ss[i][26], 26, err.Error())
			return false
		}
		v.ComboSKill = int32(intv)
		if id <= 0 {
			continue
		}
		this.id2items[int32(id)] = &v
		this.items_array = append(this.items_array, &v)
   	}
	return true
}

func (this *SkillMgr) Get(id int32) *Skill {
	return this.id2items[id]
}

func (this *SkillMgr) GetByIndex(idx int32) *Skill {
	if int(idx) >= len(this.items_array) {
		return nil
	}
	return this.items_array[idx]
}

func (this *SkillMgr) GetNum() int32 {
	return int32(len(this.items_array))
}

