package csv_readers

import (
	"encoding/csv"
	"io/ioutil"
	"log"
	"strconv"
	"strings"
)

type Expedition struct {
	ID int32
	StageType int32
	PlayerCardMax int32
	EnemyBattlePower int32
	GoldBase int32
	GoldRate int32
	TokenBase int32
	TokenRate int32
	PurifyPoint int32
}

type ExpeditionMgr struct {
	id2items map[int32]*Expedition
	items_array []*Expedition
}

func (this *ExpeditionMgr) Read(file_path_name string) bool {
	if file_path_name == "" {
		file_path_name = "../game_csv/expedition.csv"
	}
	cs, err := ioutil.ReadFile(file_path_name)
	if err != nil {
		log.Printf("ExpeditionMgr.Read err: %v", err.Error())
		return false
	}

 	r := csv.NewReader(strings.NewReader(string(cs)))
	ss, _ := r.ReadAll()
   	sz := len(ss)
	this.id2items = make(map[int32]*Expedition)
    for i := int32(1); i < int32(sz); i++ {
		//if i < 5 {
		//	continue
		//}
		var v Expedition
		var intv, id int
		// ID
		intv, err = strconv.Atoi(ss[i][0])
		if err != nil {
			log.Printf("table Expedition convert column ID value %v with row %v err %v", ss[i][0], 0, err.Error())
			return false
		}
		v.ID = int32(intv)
		id = intv
		// StageType
		intv, err = strconv.Atoi(ss[i][1])
		if err != nil {
			log.Printf("table Expedition convert column StageType value %v with row %v err %v", ss[i][1], 1, err.Error())
			return false
		}
		v.StageType = int32(intv)
		// PlayerCardMax
		intv, err = strconv.Atoi(ss[i][2])
		if err != nil {
			log.Printf("table Expedition convert column PlayerCardMax value %v with row %v err %v", ss[i][2], 2, err.Error())
			return false
		}
		v.PlayerCardMax = int32(intv)
		// EnemyBattlePower
		intv, err = strconv.Atoi(ss[i][3])
		if err != nil {
			log.Printf("table Expedition convert column EnemyBattlePower value %v with row %v err %v", ss[i][3], 3, err.Error())
			return false
		}
		v.EnemyBattlePower = int32(intv)
		// GoldBase
		intv, err = strconv.Atoi(ss[i][4])
		if err != nil {
			log.Printf("table Expedition convert column GoldBase value %v with row %v err %v", ss[i][4], 4, err.Error())
			return false
		}
		v.GoldBase = int32(intv)
		// GoldRate
		intv, err = strconv.Atoi(ss[i][5])
		if err != nil {
			log.Printf("table Expedition convert column GoldRate value %v with row %v err %v", ss[i][5], 5, err.Error())
			return false
		}
		v.GoldRate = int32(intv)
		// TokenBase
		intv, err = strconv.Atoi(ss[i][6])
		if err != nil {
			log.Printf("table Expedition convert column TokenBase value %v with row %v err %v", ss[i][6], 6, err.Error())
			return false
		}
		v.TokenBase = int32(intv)
		// TokenRate
		intv, err = strconv.Atoi(ss[i][7])
		if err != nil {
			log.Printf("table Expedition convert column TokenRate value %v with row %v err %v", ss[i][7], 7, err.Error())
			return false
		}
		v.TokenRate = int32(intv)
		// PurifyPoint
		intv, err = strconv.Atoi(ss[i][8])
		if err != nil {
			log.Printf("table Expedition convert column PurifyPoint value %v with row %v err %v", ss[i][8], 8, err.Error())
			return false
		}
		v.PurifyPoint = int32(intv)
		if id <= 0 {
			continue
		}
		this.id2items[int32(id)] = &v
		this.items_array = append(this.items_array, &v)
   	}
	return true
}

func (this *ExpeditionMgr) Get(id int32) *Expedition {
	return this.id2items[id]
}

func (this *ExpeditionMgr) GetByIndex(idx int32) *Expedition {
	if int(idx) >= len(this.items_array) {
		return nil
	}
	return this.items_array[idx]
}

func (this *ExpeditionMgr) GetNum() int32 {
	return int32(len(this.items_array))
}

