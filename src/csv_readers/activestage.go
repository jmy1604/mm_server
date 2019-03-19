package csv_readers

import (
	"encoding/csv"
	"io/ioutil"
	"log"
	"strconv"
	"strings"
)

type Activestage struct {
	ID int32
	Type int32
	PlayerLevelCond int32
	StageID int32
	PlayerLevelSuggestion int32
}

type ActivestageMgr struct {
	id2items map[int32]*Activestage
	items_array []*Activestage
}

func (this *ActivestageMgr) Read(file_path_name string) bool {
	if file_path_name == "" {
		file_path_name = "../game_csv/activestage.csv"
	}
	cs, err := ioutil.ReadFile(file_path_name)
	if err != nil {
		log.Printf("ActivestageMgr.Read err: %v", err.Error())
		return false
	}

 	r := csv.NewReader(strings.NewReader(string(cs)))
	ss, _ := r.ReadAll()
   	sz := len(ss)
	this.id2items = make(map[int32]*Activestage)
    for i := int32(1); i < int32(sz); i++ {
		//if i < 5 {
		//	continue
		//}
		var v Activestage
		var intv, id int
		// ID
		intv, err = strconv.Atoi(ss[i][0])
		if err != nil {
			log.Printf("table Activestage convert column ID value %v with row %v err %v", ss[i][0], 0, err.Error())
			return false
		}
		v.ID = int32(intv)
		id = intv
		// Type
		intv, err = strconv.Atoi(ss[i][1])
		if err != nil {
			log.Printf("table Activestage convert column Type value %v with row %v err %v", ss[i][1], 1, err.Error())
			return false
		}
		v.Type = int32(intv)
		// PlayerLevelCond
		intv, err = strconv.Atoi(ss[i][2])
		if err != nil {
			log.Printf("table Activestage convert column PlayerLevelCond value %v with row %v err %v", ss[i][2], 2, err.Error())
			return false
		}
		v.PlayerLevelCond = int32(intv)
		// StageID
		intv, err = strconv.Atoi(ss[i][3])
		if err != nil {
			log.Printf("table Activestage convert column StageID value %v with row %v err %v", ss[i][3], 3, err.Error())
			return false
		}
		v.StageID = int32(intv)
		// PlayerLevelSuggestion
		intv, err = strconv.Atoi(ss[i][4])
		if err != nil {
			log.Printf("table Activestage convert column PlayerLevelSuggestion value %v with row %v err %v", ss[i][4], 4, err.Error())
			return false
		}
		v.PlayerLevelSuggestion = int32(intv)
		if id <= 0 {
			continue
		}
		this.id2items[int32(id)] = &v
		this.items_array = append(this.items_array, &v)
   	}
	return true
}

func (this *ActivestageMgr) Get(id int32) *Activestage {
	return this.id2items[id]
}

func (this *ActivestageMgr) GetByIndex(idx int32) *Activestage {
	if int(idx) >= len(this.items_array) {
		return nil
	}
	return this.items_array[idx]
}

func (this *ActivestageMgr) GetNum() int32 {
	return int32(len(this.items_array))
}

