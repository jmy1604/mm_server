package csv_readers

import (
	"encoding/csv"
	"io/ioutil"
	"log"
	"strconv"
	"strings"
)

type Talent struct {
	TalentID int32
	TalentBaseID int32
	MaxLevel int32
	Level int32
	CanLearn int32
	UpgradeCost string
	PreSkillCond int32
	PreSkillLevCond int32
	PageLabel int32
	TeamSpeedBonus int32
	TalentEffectCond string
	TalentAttr string
	TalentSkillList string
}

type TalentMgr struct {
	id2items map[int32]*Talent
	items_array []*Talent
}

func (this *TalentMgr) Read(file_path_name string) bool {
	if file_path_name == "" {
		file_path_name = "../game_csv/talent.csv"
	}
	cs, err := ioutil.ReadFile(file_path_name)
	if err != nil {
		log.Printf("TalentMgr.Read err: %v", err.Error())
		return false
	}

 	r := csv.NewReader(strings.NewReader(string(cs)))
	ss, _ := r.ReadAll()
   	sz := len(ss)
	this.id2items = make(map[int32]*Talent)
    for i := int32(1); i < int32(sz); i++ {
		//if i < 5 {
		//	continue
		//}
		var v Talent
		var intv, id int
		// TalentID
		intv, err = strconv.Atoi(ss[i][0])
		if err != nil {
			log.Printf("table Talent convert column TalentID value %v with row %v err %v", ss[i][0], 0, err.Error())
			return false
		}
		v.TalentID = int32(intv)
		id = intv
		// TalentBaseID
		intv, err = strconv.Atoi(ss[i][1])
		if err != nil {
			log.Printf("table Talent convert column TalentBaseID value %v with row %v err %v", ss[i][1], 1, err.Error())
			return false
		}
		v.TalentBaseID = int32(intv)
		// MaxLevel
		intv, err = strconv.Atoi(ss[i][2])
		if err != nil {
			log.Printf("table Talent convert column MaxLevel value %v with row %v err %v", ss[i][2], 2, err.Error())
			return false
		}
		v.MaxLevel = int32(intv)
		// Level
		intv, err = strconv.Atoi(ss[i][3])
		if err != nil {
			log.Printf("table Talent convert column Level value %v with row %v err %v", ss[i][3], 3, err.Error())
			return false
		}
		v.Level = int32(intv)
		// CanLearn
		intv, err = strconv.Atoi(ss[i][4])
		if err != nil {
			log.Printf("table Talent convert column CanLearn value %v with row %v err %v", ss[i][4], 4, err.Error())
			return false
		}
		v.CanLearn = int32(intv)
		// UpgradeCost
		v.UpgradeCost = ss[i][5]
		// PreSkillCond
		intv, err = strconv.Atoi(ss[i][6])
		if err != nil {
			log.Printf("table Talent convert column PreSkillCond value %v with row %v err %v", ss[i][6], 6, err.Error())
			return false
		}
		v.PreSkillCond = int32(intv)
		// PreSkillLevCond
		intv, err = strconv.Atoi(ss[i][7])
		if err != nil {
			log.Printf("table Talent convert column PreSkillLevCond value %v with row %v err %v", ss[i][7], 7, err.Error())
			return false
		}
		v.PreSkillLevCond = int32(intv)
		// PageLabel
		intv, err = strconv.Atoi(ss[i][8])
		if err != nil {
			log.Printf("table Talent convert column PageLabel value %v with row %v err %v", ss[i][8], 8, err.Error())
			return false
		}
		v.PageLabel = int32(intv)
		// TeamSpeedBonus
		intv, err = strconv.Atoi(ss[i][9])
		if err != nil {
			log.Printf("table Talent convert column TeamSpeedBonus value %v with row %v err %v", ss[i][9], 9, err.Error())
			return false
		}
		v.TeamSpeedBonus = int32(intv)
		// TalentEffectCond
		v.TalentEffectCond = ss[i][10]
		// TalentAttr
		v.TalentAttr = ss[i][11]
		// TalentSkillList
		v.TalentSkillList = ss[i][12]
		if id <= 0 {
			continue
		}
		this.id2items[int32(id)] = &v
		this.items_array = append(this.items_array, &v)
   	}
	return true
}

func (this *TalentMgr) Get(id int32) *Talent {
	return this.id2items[id]
}

func (this *TalentMgr) GetByIndex(idx int32) *Talent {
	if int(idx) >= len(this.items_array) {
		return nil
	}
	return this.items_array[idx]
}

func (this *TalentMgr) GetNum() int32 {
	return int32(len(this.items_array))
}

