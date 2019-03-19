package csv_readers

import (
	"encoding/csv"
	"io/ioutil"
	"log"
	"strconv"
	"strings"
)

type Guildboss struct {
	BossIndex int32
	StageID int32
	BattleReward string
	RankReward1Cond string
	RankReward1 string
	RankReward2Cond string
	RankReward2 string
	RankReward3Cond string
	RankReward3 string
	RankReward4Cond string
	RankReward4 string
	RankReward5Cond string
	RankReward5 string
}

type GuildbossMgr struct {
	id2items map[int32]*Guildboss
	items_array []*Guildboss
}

func (this *GuildbossMgr) Read(file_path_name string) bool {
	if file_path_name == "" {
		file_path_name = "../game_csv/guildboss.csv"
	}
	cs, err := ioutil.ReadFile(file_path_name)
	if err != nil {
		log.Printf("GuildbossMgr.Read err: %v", err.Error())
		return false
	}

 	r := csv.NewReader(strings.NewReader(string(cs)))
	ss, _ := r.ReadAll()
   	sz := len(ss)
	this.id2items = make(map[int32]*Guildboss)
    for i := int32(1); i < int32(sz); i++ {
		//if i < 5 {
		//	continue
		//}
		var v Guildboss
		var intv, id int
		// BossIndex
		intv, err = strconv.Atoi(ss[i][0])
		if err != nil {
			log.Printf("table Guildboss convert column BossIndex value %v with row %v err %v", ss[i][0], 0, err.Error())
			return false
		}
		v.BossIndex = int32(intv)
		id = intv
		// StageID
		intv, err = strconv.Atoi(ss[i][1])
		if err != nil {
			log.Printf("table Guildboss convert column StageID value %v with row %v err %v", ss[i][1], 1, err.Error())
			return false
		}
		v.StageID = int32(intv)
		// BattleReward
		v.BattleReward = ss[i][2]
		// RankReward1Cond
		v.RankReward1Cond = ss[i][3]
		// RankReward1
		v.RankReward1 = ss[i][4]
		// RankReward2Cond
		v.RankReward2Cond = ss[i][5]
		// RankReward2
		v.RankReward2 = ss[i][6]
		// RankReward3Cond
		v.RankReward3Cond = ss[i][7]
		// RankReward3
		v.RankReward3 = ss[i][8]
		// RankReward4Cond
		v.RankReward4Cond = ss[i][9]
		// RankReward4
		v.RankReward4 = ss[i][10]
		// RankReward5Cond
		v.RankReward5Cond = ss[i][11]
		// RankReward5
		v.RankReward5 = ss[i][12]
		if id <= 0 {
			continue
		}
		this.id2items[int32(id)] = &v
		this.items_array = append(this.items_array, &v)
   	}
	return true
}

func (this *GuildbossMgr) Get(id int32) *Guildboss {
	return this.id2items[id]
}

func (this *GuildbossMgr) GetByIndex(idx int32) *Guildboss {
	if int(idx) >= len(this.items_array) {
		return nil
	}
	return this.items_array[idx]
}

func (this *GuildbossMgr) GetNum() int32 {
	return int32(len(this.items_array))
}

