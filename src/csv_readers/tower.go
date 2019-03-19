package csv_readers

import (
	"encoding/csv"
	"io/ioutil"
	"log"
	"strconv"
	"strings"
)

type Tower struct {
	TowerID int32
	StageID int32
	UnlockTower int32
}

type TowerMgr struct {
	id2items map[int32]*Tower
	items_array []*Tower
}

func (this *TowerMgr) Read(file_path_name string) bool {
	if file_path_name == "" {
		file_path_name = "../game_csv/tower.csv"
	}
	cs, err := ioutil.ReadFile(file_path_name)
	if err != nil {
		log.Printf("TowerMgr.Read err: %v", err.Error())
		return false
	}

 	r := csv.NewReader(strings.NewReader(string(cs)))
	ss, _ := r.ReadAll()
   	sz := len(ss)
	this.id2items = make(map[int32]*Tower)
    for i := int32(1); i < int32(sz); i++ {
		//if i < 5 {
		//	continue
		//}
		var v Tower
		var intv, id int
		// TowerID
		intv, err = strconv.Atoi(ss[i][0])
		if err != nil {
			log.Printf("table Tower convert column TowerID value %v with row %v err %v", ss[i][0], 0, err.Error())
			return false
		}
		v.TowerID = int32(intv)
		id = intv
		// StageID
		intv, err = strconv.Atoi(ss[i][1])
		if err != nil {
			log.Printf("table Tower convert column StageID value %v with row %v err %v", ss[i][1], 1, err.Error())
			return false
		}
		v.StageID = int32(intv)
		// UnlockTower
		intv, err = strconv.Atoi(ss[i][2])
		if err != nil {
			log.Printf("table Tower convert column UnlockTower value %v with row %v err %v", ss[i][2], 2, err.Error())
			return false
		}
		v.UnlockTower = int32(intv)
		if id <= 0 {
			continue
		}
		this.id2items[int32(id)] = &v
		this.items_array = append(this.items_array, &v)
   	}
	return true
}

func (this *TowerMgr) Get(id int32) *Tower {
	return this.id2items[id]
}

func (this *TowerMgr) GetByIndex(idx int32) *Tower {
	if int(idx) >= len(this.items_array) {
		return nil
	}
	return this.items_array[idx]
}

func (this *TowerMgr) GetNum() int32 {
	return int32(len(this.items_array))
}

