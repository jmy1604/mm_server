package csv_readers

import (
	"encoding/csv"
	"io/ioutil"
	"log"
	"strconv"
	"strings"
)

type Systemunlock struct {
	ServerID string
	Level int32
}

type SystemunlockMgr struct {
	id2items map[int32]*Systemunlock
	items_array []*Systemunlock
}

func (this *SystemunlockMgr) Read(file_path_name string) bool {
	if file_path_name == "" {
		file_path_name = "../game_csv/systemunlock.csv"
	}
	cs, err := ioutil.ReadFile(file_path_name)
	if err != nil {
		log.Printf("SystemunlockMgr.Read err: %v", err.Error())
		return false
	}

 	r := csv.NewReader(strings.NewReader(string(cs)))
	ss, _ := r.ReadAll()
   	sz := len(ss)
	this.id2items = make(map[int32]*Systemunlock)
    for i := int32(1); i < int32(sz); i++ {
		//if i < 5 {
		//	continue
		//}
		var v Systemunlock
		var intv, id int
		// ServerID
		v.ServerID = ss[i][0]
		// Level
		intv, err = strconv.Atoi(ss[i][1])
		if err != nil {
			log.Printf("table Systemunlock convert column Level value %v with row %v err %v", ss[i][1], 1, err.Error())
			return false
		}
		v.Level = int32(intv)
		if id <= 0 {
			continue
		}
		this.id2items[int32(id)] = &v
		this.items_array = append(this.items_array, &v)
   	}
	return true
}

func (this *SystemunlockMgr) Get(id int32) *Systemunlock {
	return this.id2items[id]
}

func (this *SystemunlockMgr) GetByIndex(idx int32) *Systemunlock {
	if int(idx) >= len(this.items_array) {
		return nil
	}
	return this.items_array[idx]
}

func (this *SystemunlockMgr) GetNum() int32 {
	return int32(len(this.items_array))
}

