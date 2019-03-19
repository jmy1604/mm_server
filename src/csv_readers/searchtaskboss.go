package csv_readers

import (
	"encoding/csv"
	"io/ioutil"
	"log"
	"strconv"
	"strings"
)

type Searchtaskboss struct {
	SearchTaskBossGroup int32
	Weight int32
	Stageid int32
}

type SearchtaskbossMgr struct {
	id2items map[int32]*Searchtaskboss
	items_array []*Searchtaskboss
}

func (this *SearchtaskbossMgr) Read(file_path_name string) bool {
	if file_path_name == "" {
		file_path_name = "../game_csv/searchtaskboss.csv"
	}
	cs, err := ioutil.ReadFile(file_path_name)
	if err != nil {
		log.Printf("SearchtaskbossMgr.Read err: %v", err.Error())
		return false
	}

 	r := csv.NewReader(strings.NewReader(string(cs)))
	ss, _ := r.ReadAll()
   	sz := len(ss)
	this.id2items = make(map[int32]*Searchtaskboss)
    for i := int32(1); i < int32(sz); i++ {
		//if i < 5 {
		//	continue
		//}
		var v Searchtaskboss
		var intv, id int
		// SearchTaskBossGroup
		intv, err = strconv.Atoi(ss[i][0])
		if err != nil {
			log.Printf("table Searchtaskboss convert column SearchTaskBossGroup value %v with row %v err %v", ss[i][0], 0, err.Error())
			return false
		}
		v.SearchTaskBossGroup = int32(intv)
		id = intv
		// Weight
		intv, err = strconv.Atoi(ss[i][1])
		if err != nil {
			log.Printf("table Searchtaskboss convert column Weight value %v with row %v err %v", ss[i][1], 1, err.Error())
			return false
		}
		v.Weight = int32(intv)
		// Stageid
		intv, err = strconv.Atoi(ss[i][2])
		if err != nil {
			log.Printf("table Searchtaskboss convert column Stageid value %v with row %v err %v", ss[i][2], 2, err.Error())
			return false
		}
		v.Stageid = int32(intv)
		if id <= 0 {
			continue
		}
		this.id2items[int32(id)] = &v
		this.items_array = append(this.items_array, &v)
   	}
	return true
}

func (this *SearchtaskbossMgr) Get(id int32) *Searchtaskboss {
	return this.id2items[id]
}

func (this *SearchtaskbossMgr) GetByIndex(idx int32) *Searchtaskboss {
	if int(idx) >= len(this.items_array) {
		return nil
	}
	return this.items_array[idx]
}

func (this *SearchtaskbossMgr) GetNum() int32 {
	return int32(len(this.items_array))
}

