package csv_readers

import (
	"encoding/csv"
	"io/ioutil"
	"log"
	"strconv"
	"strings"
)

type Goldhand struct {
	Level int32
	GoldReward1 int32
	GemCost1 int32
	GoldReward2 int32
	GemCost2 int32
	GoldReward3 int32
	GemCost3 int32
	RefreshCD int32
}

type GoldhandMgr struct {
	id2items map[int32]*Goldhand
	items_array []*Goldhand
}

func (this *GoldhandMgr) Read(file_path_name string) bool {
	if file_path_name == "" {
		file_path_name = "../game_csv/goldhand.csv"
	}
	cs, err := ioutil.ReadFile(file_path_name)
	if err != nil {
		log.Printf("GoldhandMgr.Read err: %v", err.Error())
		return false
	}

 	r := csv.NewReader(strings.NewReader(string(cs)))
	ss, _ := r.ReadAll()
   	sz := len(ss)
	this.id2items = make(map[int32]*Goldhand)
    for i := int32(1); i < int32(sz); i++ {
		//if i < 5 {
		//	continue
		//}
		var v Goldhand
		var intv, id int
		// Level
		intv, err = strconv.Atoi(ss[i][0])
		if err != nil {
			log.Printf("table Goldhand convert column Level value %v with row %v err %v", ss[i][0], 0, err.Error())
			return false
		}
		v.Level = int32(intv)
		id = intv
		// GoldReward1
		intv, err = strconv.Atoi(ss[i][1])
		if err != nil {
			log.Printf("table Goldhand convert column GoldReward1 value %v with row %v err %v", ss[i][1], 1, err.Error())
			return false
		}
		v.GoldReward1 = int32(intv)
		// GemCost1
		intv, err = strconv.Atoi(ss[i][2])
		if err != nil {
			log.Printf("table Goldhand convert column GemCost1 value %v with row %v err %v", ss[i][2], 2, err.Error())
			return false
		}
		v.GemCost1 = int32(intv)
		// GoldReward2
		intv, err = strconv.Atoi(ss[i][3])
		if err != nil {
			log.Printf("table Goldhand convert column GoldReward2 value %v with row %v err %v", ss[i][3], 3, err.Error())
			return false
		}
		v.GoldReward2 = int32(intv)
		// GemCost2
		intv, err = strconv.Atoi(ss[i][4])
		if err != nil {
			log.Printf("table Goldhand convert column GemCost2 value %v with row %v err %v", ss[i][4], 4, err.Error())
			return false
		}
		v.GemCost2 = int32(intv)
		// GoldReward3
		intv, err = strconv.Atoi(ss[i][5])
		if err != nil {
			log.Printf("table Goldhand convert column GoldReward3 value %v with row %v err %v", ss[i][5], 5, err.Error())
			return false
		}
		v.GoldReward3 = int32(intv)
		// GemCost3
		intv, err = strconv.Atoi(ss[i][6])
		if err != nil {
			log.Printf("table Goldhand convert column GemCost3 value %v with row %v err %v", ss[i][6], 6, err.Error())
			return false
		}
		v.GemCost3 = int32(intv)
		// RefreshCD
		intv, err = strconv.Atoi(ss[i][7])
		if err != nil {
			log.Printf("table Goldhand convert column RefreshCD value %v with row %v err %v", ss[i][7], 7, err.Error())
			return false
		}
		v.RefreshCD = int32(intv)
		if id <= 0 {
			continue
		}
		this.id2items[int32(id)] = &v
		this.items_array = append(this.items_array, &v)
   	}
	return true
}

func (this *GoldhandMgr) Get(id int32) *Goldhand {
	return this.id2items[id]
}

func (this *GoldhandMgr) GetByIndex(idx int32) *Goldhand {
	if int(idx) >= len(this.items_array) {
		return nil
	}
	return this.items_array[idx]
}

func (this *GoldhandMgr) GetNum() int32 {
	return int32(len(this.items_array))
}

