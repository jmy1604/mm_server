package csv_readers

import (
	"encoding/csv"
	"io/ioutil"
	"log"
	"strconv"
	"strings"
)

type Friendboss struct {
	ID int32
	LevelMin int32
	LevelMax int32
	SearchBossChance int32
	SearchItemDropID int32
	BossStageID int32
	ChallengeDropID int32
	RewardLastHit string
	RewardOwner string
}

type FriendbossMgr struct {
	id2items map[int32]*Friendboss
	items_array []*Friendboss
}

func (this *FriendbossMgr) Read(file_path_name string) bool {
	if file_path_name == "" {
		file_path_name = "../game_csv/friendboss.csv"
	}
	cs, err := ioutil.ReadFile(file_path_name)
	if err != nil {
		log.Printf("FriendbossMgr.Read err: %v", err.Error())
		return false
	}

 	r := csv.NewReader(strings.NewReader(string(cs)))
	ss, _ := r.ReadAll()
   	sz := len(ss)
	this.id2items = make(map[int32]*Friendboss)
    for i := int32(1); i < int32(sz); i++ {
		//if i < 5 {
		//	continue
		//}
		var v Friendboss
		var intv, id int
		// ID
		intv, err = strconv.Atoi(ss[i][0])
		if err != nil {
			log.Printf("table Friendboss convert column ID value %v with row %v err %v", ss[i][0], 0, err.Error())
			return false
		}
		v.ID = int32(intv)
		id = intv
		// LevelMin
		intv, err = strconv.Atoi(ss[i][1])
		if err != nil {
			log.Printf("table Friendboss convert column LevelMin value %v with row %v err %v", ss[i][1], 1, err.Error())
			return false
		}
		v.LevelMin = int32(intv)
		// LevelMax
		intv, err = strconv.Atoi(ss[i][2])
		if err != nil {
			log.Printf("table Friendboss convert column LevelMax value %v with row %v err %v", ss[i][2], 2, err.Error())
			return false
		}
		v.LevelMax = int32(intv)
		// SearchBossChance
		intv, err = strconv.Atoi(ss[i][3])
		if err != nil {
			log.Printf("table Friendboss convert column SearchBossChance value %v with row %v err %v", ss[i][3], 3, err.Error())
			return false
		}
		v.SearchBossChance = int32(intv)
		// SearchItemDropID
		intv, err = strconv.Atoi(ss[i][4])
		if err != nil {
			log.Printf("table Friendboss convert column SearchItemDropID value %v with row %v err %v", ss[i][4], 4, err.Error())
			return false
		}
		v.SearchItemDropID = int32(intv)
		// BossStageID
		intv, err = strconv.Atoi(ss[i][5])
		if err != nil {
			log.Printf("table Friendboss convert column BossStageID value %v with row %v err %v", ss[i][5], 5, err.Error())
			return false
		}
		v.BossStageID = int32(intv)
		// ChallengeDropID
		intv, err = strconv.Atoi(ss[i][6])
		if err != nil {
			log.Printf("table Friendboss convert column ChallengeDropID value %v with row %v err %v", ss[i][6], 6, err.Error())
			return false
		}
		v.ChallengeDropID = int32(intv)
		// RewardLastHit
		v.RewardLastHit = ss[i][7]
		// RewardOwner
		v.RewardOwner = ss[i][8]
		if id <= 0 {
			continue
		}
		this.id2items[int32(id)] = &v
		this.items_array = append(this.items_array, &v)
   	}
	return true
}

func (this *FriendbossMgr) Get(id int32) *Friendboss {
	return this.id2items[id]
}

func (this *FriendbossMgr) GetByIndex(idx int32) *Friendboss {
	if int(idx) >= len(this.items_array) {
		return nil
	}
	return this.items_array[idx]
}

func (this *FriendbossMgr) GetNum() int32 {
	return int32(len(this.items_array))
}

