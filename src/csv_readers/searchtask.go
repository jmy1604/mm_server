package csv_readers

import (
	"encoding/csv"
	"io/ioutil"
	"log"
	"strconv"
	"strings"
)

type Searchtask struct {
	Id int32
	Type int32
	TaskWeight int32
	TaskStar int32
	CardStarCond int32
	CardCampNumCond int32
	CardCampCond string
	CardTypeNumCond int32
	CardTypeCond string
	CardNum int32
	SearchTime int32
	AccelCost int32
	ConstReward string
	RandomReward int32
	BonusStageLevelCond int32
	BonusStageChance int32
	BonusStageListID int32
	TaskHeroNameList string
	TaskNameList string
	TaskRoleList string
}

type SearchtaskMgr struct {
	id2items map[int32]*Searchtask
	items_array []*Searchtask
}

func (this *SearchtaskMgr) Read(file_path_name string) bool {
	if file_path_name == "" {
		file_path_name = "../game_csv/searchtask.csv"
	}
	cs, err := ioutil.ReadFile(file_path_name)
	if err != nil {
		log.Printf("SearchtaskMgr.Read err: %v", err.Error())
		return false
	}

 	r := csv.NewReader(strings.NewReader(string(cs)))
	ss, _ := r.ReadAll()
   	sz := len(ss)
	this.id2items = make(map[int32]*Searchtask)
    for i := int32(1); i < int32(sz); i++ {
		//if i < 5 {
		//	continue
		//}
		var v Searchtask
		var intv, id int
		// Id
		intv, err = strconv.Atoi(ss[i][0])
		if err != nil {
			log.Printf("table Searchtask convert column Id value %v with row %v err %v", ss[i][0], 0, err.Error())
			return false
		}
		v.Id = int32(intv)
		id = intv
		// Type
		intv, err = strconv.Atoi(ss[i][1])
		if err != nil {
			log.Printf("table Searchtask convert column Type value %v with row %v err %v", ss[i][1], 1, err.Error())
			return false
		}
		v.Type = int32(intv)
		// TaskWeight
		intv, err = strconv.Atoi(ss[i][2])
		if err != nil {
			log.Printf("table Searchtask convert column TaskWeight value %v with row %v err %v", ss[i][2], 2, err.Error())
			return false
		}
		v.TaskWeight = int32(intv)
		// TaskStar
		intv, err = strconv.Atoi(ss[i][3])
		if err != nil {
			log.Printf("table Searchtask convert column TaskStar value %v with row %v err %v", ss[i][3], 3, err.Error())
			return false
		}
		v.TaskStar = int32(intv)
		// CardStarCond
		intv, err = strconv.Atoi(ss[i][4])
		if err != nil {
			log.Printf("table Searchtask convert column CardStarCond value %v with row %v err %v", ss[i][4], 4, err.Error())
			return false
		}
		v.CardStarCond = int32(intv)
		// CardCampNumCond
		intv, err = strconv.Atoi(ss[i][5])
		if err != nil {
			log.Printf("table Searchtask convert column CardCampNumCond value %v with row %v err %v", ss[i][5], 5, err.Error())
			return false
		}
		v.CardCampNumCond = int32(intv)
		// CardCampCond
		v.CardCampCond = ss[i][6]
		// CardTypeNumCond
		intv, err = strconv.Atoi(ss[i][7])
		if err != nil {
			log.Printf("table Searchtask convert column CardTypeNumCond value %v with row %v err %v", ss[i][7], 7, err.Error())
			return false
		}
		v.CardTypeNumCond = int32(intv)
		// CardTypeCond
		v.CardTypeCond = ss[i][8]
		// CardNum
		intv, err = strconv.Atoi(ss[i][9])
		if err != nil {
			log.Printf("table Searchtask convert column CardNum value %v with row %v err %v", ss[i][9], 9, err.Error())
			return false
		}
		v.CardNum = int32(intv)
		// SearchTime
		intv, err = strconv.Atoi(ss[i][10])
		if err != nil {
			log.Printf("table Searchtask convert column SearchTime value %v with row %v err %v", ss[i][10], 10, err.Error())
			return false
		}
		v.SearchTime = int32(intv)
		// AccelCost
		intv, err = strconv.Atoi(ss[i][11])
		if err != nil {
			log.Printf("table Searchtask convert column AccelCost value %v with row %v err %v", ss[i][11], 11, err.Error())
			return false
		}
		v.AccelCost = int32(intv)
		// ConstReward
		v.ConstReward = ss[i][12]
		// RandomReward
		intv, err = strconv.Atoi(ss[i][13])
		if err != nil {
			log.Printf("table Searchtask convert column RandomReward value %v with row %v err %v", ss[i][13], 13, err.Error())
			return false
		}
		v.RandomReward = int32(intv)
		// BonusStageLevelCond
		intv, err = strconv.Atoi(ss[i][14])
		if err != nil {
			log.Printf("table Searchtask convert column BonusStageLevelCond value %v with row %v err %v", ss[i][14], 14, err.Error())
			return false
		}
		v.BonusStageLevelCond = int32(intv)
		// BonusStageChance
		intv, err = strconv.Atoi(ss[i][15])
		if err != nil {
			log.Printf("table Searchtask convert column BonusStageChance value %v with row %v err %v", ss[i][15], 15, err.Error())
			return false
		}
		v.BonusStageChance = int32(intv)
		// BonusStageListID
		intv, err = strconv.Atoi(ss[i][16])
		if err != nil {
			log.Printf("table Searchtask convert column BonusStageListID value %v with row %v err %v", ss[i][16], 16, err.Error())
			return false
		}
		v.BonusStageListID = int32(intv)
		// TaskHeroNameList
		v.TaskHeroNameList = ss[i][17]
		// TaskNameList
		v.TaskNameList = ss[i][18]
		// TaskRoleList
		v.TaskRoleList = ss[i][19]
		if id <= 0 {
			continue
		}
		this.id2items[int32(id)] = &v
		this.items_array = append(this.items_array, &v)
   	}
	return true
}

func (this *SearchtaskMgr) Get(id int32) *Searchtask {
	return this.id2items[id]
}

func (this *SearchtaskMgr) GetByIndex(idx int32) *Searchtask {
	if int(idx) >= len(this.items_array) {
		return nil
	}
	return this.items_array[idx]
}

func (this *SearchtaskMgr) GetNum() int32 {
	return int32(len(this.items_array))
}

