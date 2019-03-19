package csv_readers

import (
	"encoding/csv"
	"io/ioutil"
	"log"
	"strconv"
	"strings"
)

type Accelcost struct {
	AccelTimes int32
	Cost int32
}

type AccelcostMgr struct {
	id2items map[int32]*Accelcost
	items_array []*Accelcost
}

func (this *AccelcostMgr) Read(file_path_name string) bool {
	if file_path_name == "" {
		file_path_name = "../game_csv/accelcost.csv"
	}
	cs, err := ioutil.ReadFile(file_path_name)
	if err != nil {
		log.Printf("AccelcostMgr.Read err: %v", err.Error())
		return false
	}

 	r := csv.NewReader(strings.NewReader(string(cs)))
	ss, _ := r.ReadAll()
   	sz := len(ss)
	this.id2items = make(map[int32]*Accelcost)
    for i := int32(1); i < int32(sz); i++ {
		//if i < 5 {
		//	continue
		//}
		var v Accelcost
		var intv, id int
		// AccelTimes
		intv, err = strconv.Atoi(ss[i][0])
		if err != nil {
			log.Printf("table Accelcost convert column AccelTimes value %v with row %v err %v", ss[i][0], 0, err.Error())
			return false
		}
		v.AccelTimes = int32(intv)
		id = intv
		// Cost
		intv, err = strconv.Atoi(ss[i][1])
		if err != nil {
			log.Printf("table Accelcost convert column Cost value %v with row %v err %v", ss[i][1], 1, err.Error())
			return false
		}
		v.Cost = int32(intv)
		if id <= 0 {
			continue
		}
		this.id2items[int32(id)] = &v
		this.items_array = append(this.items_array, &v)
   	}
	return true
}

func (this *AccelcostMgr) Get(id int32) *Accelcost {
	return this.id2items[id]
}

func (this *AccelcostMgr) GetByIndex(idx int32) *Accelcost {
	if int(idx) >= len(this.items_array) {
		return nil
	}
	return this.items_array[idx]
}

func (this *AccelcostMgr) GetNum() int32 {
	return int32(len(this.items_array))
}

