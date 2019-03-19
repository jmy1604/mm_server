package csv_readers

import (
	"encoding/csv"
	"io/ioutil"
	"log"
	"strconv"
	"strings"
)

type Levelup struct {
	Level int32
	PlayerLevelUpExp int32
	CardLevelUpRes string
	CardDecomposeRes string
}

type LevelupMgr struct {
	id2items map[int32]*Levelup
	items_array []*Levelup
}

func (this *LevelupMgr) Read(file_path_name string) bool {
	if file_path_name == "" {
		file_path_name = "../game_csv/levelup.csv"
	}
	cs, err := ioutil.ReadFile(file_path_name)
	if err != nil {
		log.Printf("LevelupMgr.Read err: %v", err.Error())
		return false
	}

 	r := csv.NewReader(strings.NewReader(string(cs)))
	ss, _ := r.ReadAll()
   	sz := len(ss)
	this.id2items = make(map[int32]*Levelup)
    for i := int32(1); i < int32(sz); i++ {
		//if i < 5 {
		//	continue
		//}
		var v Levelup
		var intv, id int
		// Level
		intv, err = strconv.Atoi(ss[i][0])
		if err != nil {
			log.Printf("table Levelup convert column Level value %v with row %v err %v", ss[i][0], 0, err.Error())
			return false
		}
		v.Level = int32(intv)
		id = intv
		// PlayerLevelUpExp
		intv, err = strconv.Atoi(ss[i][1])
		if err != nil {
			log.Printf("table Levelup convert column PlayerLevelUpExp value %v with row %v err %v", ss[i][1], 1, err.Error())
			return false
		}
		v.PlayerLevelUpExp = int32(intv)
		// CardLevelUpRes
		v.CardLevelUpRes = ss[i][2]
		// CardDecomposeRes
		v.CardDecomposeRes = ss[i][3]
		if id <= 0 {
			continue
		}
		this.id2items[int32(id)] = &v
		this.items_array = append(this.items_array, &v)
   	}
	return true
}

func (this *LevelupMgr) Get(id int32) *Levelup {
	return this.id2items[id]
}

func (this *LevelupMgr) GetByIndex(idx int32) *Levelup {
	if int(idx) >= len(this.items_array) {
		return nil
	}
	return this.items_array[idx]
}

func (this *LevelupMgr) GetNum() int32 {
	return int32(len(this.items_array))
}

