package csv_readers

import (
	"encoding/csv"
	"io/ioutil"
	"log"
	"strconv"
	"strings"
)

type Arenarankingbonus struct {
	Index int32
	RankingMin int32
	RankingMax int32
	DayRewardList string
	SeasonRewardList string
}

type ArenarankingbonusMgr struct {
	id2items map[int32]*Arenarankingbonus
	items_array []*Arenarankingbonus
}

func (this *ArenarankingbonusMgr) Read(file_path_name string) bool {
	if file_path_name == "" {
		file_path_name = "../game_csv/arenarankingbonus.csv"
	}
	cs, err := ioutil.ReadFile(file_path_name)
	if err != nil {
		log.Printf("ArenarankingbonusMgr.Read err: %v", err.Error())
		return false
	}

 	r := csv.NewReader(strings.NewReader(string(cs)))
	ss, _ := r.ReadAll()
   	sz := len(ss)
	this.id2items = make(map[int32]*Arenarankingbonus)
    for i := int32(1); i < int32(sz); i++ {
		//if i < 5 {
		//	continue
		//}
		var v Arenarankingbonus
		var intv, id int
		// Index
		intv, err = strconv.Atoi(ss[i][0])
		if err != nil {
			log.Printf("table Arenarankingbonus convert column Index value %v with row %v err %v", ss[i][0], 0, err.Error())
			return false
		}
		v.Index = int32(intv)
		id = intv
		// RankingMin
		intv, err = strconv.Atoi(ss[i][1])
		if err != nil {
			log.Printf("table Arenarankingbonus convert column RankingMin value %v with row %v err %v", ss[i][1], 1, err.Error())
			return false
		}
		v.RankingMin = int32(intv)
		// RankingMax
		intv, err = strconv.Atoi(ss[i][2])
		if err != nil {
			log.Printf("table Arenarankingbonus convert column RankingMax value %v with row %v err %v", ss[i][2], 2, err.Error())
			return false
		}
		v.RankingMax = int32(intv)
		// DayRewardList
		v.DayRewardList = ss[i][3]
		// SeasonRewardList
		v.SeasonRewardList = ss[i][4]
		if id <= 0 {
			continue
		}
		this.id2items[int32(id)] = &v
		this.items_array = append(this.items_array, &v)
   	}
	return true
}

func (this *ArenarankingbonusMgr) Get(id int32) *Arenarankingbonus {
	return this.id2items[id]
}

func (this *ArenarankingbonusMgr) GetByIndex(idx int32) *Arenarankingbonus {
	if int(idx) >= len(this.items_array) {
		return nil
	}
	return this.items_array[idx]
}

func (this *ArenarankingbonusMgr) GetNum() int32 {
	return int32(len(this.items_array))
}

