package csv_readers

import (
	"encoding/csv"
	"io/ioutil"
	"log"
	"strconv"
	"strings"
)

type Drop struct {
	DropGroupID int32
	DropItemID int32
	Weight int32
	Min int32
	Max int32
}

type DropMgr struct {
	id2items map[int32]*Drop
	items_array []*Drop
}

func (this *DropMgr) Read(file_path_name string) bool {
	if file_path_name == "" {
		file_path_name = "../game_csv/drop.csv"
	}
	cs, err := ioutil.ReadFile(file_path_name)
	if err != nil {
		log.Printf("DropMgr.Read err: %v", err.Error())
		return false
	}

 	r := csv.NewReader(strings.NewReader(string(cs)))
	ss, _ := r.ReadAll()
   	sz := len(ss)
	this.id2items = make(map[int32]*Drop)
    for i := int32(1); i < int32(sz); i++ {
		//if i < 5 {
		//	continue
		//}
		var v Drop
		var intv, id int
		// DropGroupID
		intv, err = strconv.Atoi(ss[i][0])
		if err != nil {
			log.Printf("table Drop convert column DropGroupID value %v with row %v err %v", ss[i][0], 0, err.Error())
			return false
		}
		v.DropGroupID = int32(intv)
		id = intv
		// DropItemID
		intv, err = strconv.Atoi(ss[i][1])
		if err != nil {
			log.Printf("table Drop convert column DropItemID value %v with row %v err %v", ss[i][1], 1, err.Error())
			return false
		}
		v.DropItemID = int32(intv)
		// Weight
		intv, err = strconv.Atoi(ss[i][2])
		if err != nil {
			log.Printf("table Drop convert column Weight value %v with row %v err %v", ss[i][2], 2, err.Error())
			return false
		}
		v.Weight = int32(intv)
		// Min
		intv, err = strconv.Atoi(ss[i][3])
		if err != nil {
			log.Printf("table Drop convert column Min value %v with row %v err %v", ss[i][3], 3, err.Error())
			return false
		}
		v.Min = int32(intv)
		// Max
		intv, err = strconv.Atoi(ss[i][4])
		if err != nil {
			log.Printf("table Drop convert column Max value %v with row %v err %v", ss[i][4], 4, err.Error())
			return false
		}
		v.Max = int32(intv)
		if id <= 0 {
			continue
		}
		this.id2items[int32(id)] = &v
		this.items_array = append(this.items_array, &v)
   	}
	return true
}

func (this *DropMgr) Get(id int32) *Drop {
	return this.id2items[id]
}

func (this *DropMgr) GetByIndex(idx int32) *Drop {
	if int(idx) >= len(this.items_array) {
		return nil
	}
	return this.items_array[idx]
}

func (this *DropMgr) GetNum() int32 {
	return int32(len(this.items_array))
}

