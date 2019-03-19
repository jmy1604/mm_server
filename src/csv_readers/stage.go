package csv_readers

import (
	"encoding/csv"
	"io/ioutil"
	"log"
	"strconv"
	"strings"
)

type Stage struct {
	StageID int32
	MaxWaves int32
	MonsterList string
	MaxRound int32
	TimeUpWin int32
	PlayerCardMax int32
	FriendSupportMax int32
	NpcSupportList string
	StageTeamSkillList string
	RewardList string
}

type StageMgr struct {
	id2items map[int32]*Stage
	items_array []*Stage
}

func (this *StageMgr) Read(file_path_name string) bool {
	if file_path_name == "" {
		file_path_name = "../game_csv/stage.csv"
	}
	cs, err := ioutil.ReadFile(file_path_name)
	if err != nil {
		log.Printf("StageMgr.Read err: %v", err.Error())
		return false
	}

 	r := csv.NewReader(strings.NewReader(string(cs)))
	ss, _ := r.ReadAll()
   	sz := len(ss)
	this.id2items = make(map[int32]*Stage)
    for i := int32(1); i < int32(sz); i++ {
		//if i < 5 {
		//	continue
		//}
		var v Stage
		var intv, id int
		// StageID
		intv, err = strconv.Atoi(ss[i][0])
		if err != nil {
			log.Printf("table Stage convert column StageID value %v with row %v err %v", ss[i][0], 0, err.Error())
			return false
		}
		v.StageID = int32(intv)
		id = intv
		// MaxWaves
		intv, err = strconv.Atoi(ss[i][1])
		if err != nil {
			log.Printf("table Stage convert column MaxWaves value %v with row %v err %v", ss[i][1], 1, err.Error())
			return false
		}
		v.MaxWaves = int32(intv)
		// MonsterList
		v.MonsterList = ss[i][2]
		// MaxRound
		intv, err = strconv.Atoi(ss[i][3])
		if err != nil {
			log.Printf("table Stage convert column MaxRound value %v with row %v err %v", ss[i][3], 3, err.Error())
			return false
		}
		v.MaxRound = int32(intv)
		// TimeUpWin
		intv, err = strconv.Atoi(ss[i][4])
		if err != nil {
			log.Printf("table Stage convert column TimeUpWin value %v with row %v err %v", ss[i][4], 4, err.Error())
			return false
		}
		v.TimeUpWin = int32(intv)
		// PlayerCardMax
		intv, err = strconv.Atoi(ss[i][5])
		if err != nil {
			log.Printf("table Stage convert column PlayerCardMax value %v with row %v err %v", ss[i][5], 5, err.Error())
			return false
		}
		v.PlayerCardMax = int32(intv)
		// FriendSupportMax
		intv, err = strconv.Atoi(ss[i][6])
		if err != nil {
			log.Printf("table Stage convert column FriendSupportMax value %v with row %v err %v", ss[i][6], 6, err.Error())
			return false
		}
		v.FriendSupportMax = int32(intv)
		// NpcSupportList
		v.NpcSupportList = ss[i][7]
		// StageTeamSkillList
		v.StageTeamSkillList = ss[i][8]
		// RewardList
		v.RewardList = ss[i][9]
		if id <= 0 {
			continue
		}
		this.id2items[int32(id)] = &v
		this.items_array = append(this.items_array, &v)
   	}
	return true
}

func (this *StageMgr) Get(id int32) *Stage {
	return this.id2items[id]
}

func (this *StageMgr) GetByIndex(idx int32) *Stage {
	if int(idx) >= len(this.items_array) {
		return nil
	}
	return this.items_array[idx]
}

func (this *StageMgr) GetNum() int32 {
	return int32(len(this.items_array))
}

