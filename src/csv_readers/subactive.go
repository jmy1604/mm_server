package csv_readers

import (
	"encoding/csv"
	"io/ioutil"
	"log"
	"strconv"
	"strings"
)

type Subactive struct {
	SubActiveID int32
	BundleID string
	Param1 int32
	Param2 int32
	Param3 int32
	Param4 int32
	EventCount int32
	Reward string
}

type SubactiveMgr struct {
	id2items map[int32]*Subactive
	items_array []*Subactive
}

func (this *SubactiveMgr) Read(file_path_name string) bool {
	if file_path_name == "" {
		file_path_name = "../game_csv/subactive.csv"
	}
	cs, err := ioutil.ReadFile(file_path_name)
	if err != nil {
		log.Printf("SubactiveMgr.Read err: %v", err.Error())
		return false
	}

 	r := csv.NewReader(strings.NewReader(string(cs)))
	ss, _ := r.ReadAll()
   	sz := len(ss)
	this.id2items = make(map[int32]*Subactive)
    for i := int32(1); i < int32(sz); i++ {
		//if i < 5 {
		//	continue
		//}
		var v Subactive
		var intv, id int
		// SubActiveID
		intv, err = strconv.Atoi(ss[i][0])
		if err != nil {
			log.Printf("table Subactive convert column SubActiveID value %v with row %v err %v", ss[i][0], 0, err.Error())
			return false
		}
		v.SubActiveID = int32(intv)
		id = intv
		// BundleID
		v.BundleID = ss[i][1]
		// Param1
		intv, err = strconv.Atoi(ss[i][2])
		if err != nil {
			log.Printf("table Subactive convert column Param1 value %v with row %v err %v", ss[i][2], 2, err.Error())
			return false
		}
		v.Param1 = int32(intv)
		// Param2
		intv, err = strconv.Atoi(ss[i][3])
		if err != nil {
			log.Printf("table Subactive convert column Param2 value %v with row %v err %v", ss[i][3], 3, err.Error())
			return false
		}
		v.Param2 = int32(intv)
		// Param3
		intv, err = strconv.Atoi(ss[i][4])
		if err != nil {
			log.Printf("table Subactive convert column Param3 value %v with row %v err %v", ss[i][4], 4, err.Error())
			return false
		}
		v.Param3 = int32(intv)
		// Param4
		intv, err = strconv.Atoi(ss[i][5])
		if err != nil {
			log.Printf("table Subactive convert column Param4 value %v with row %v err %v", ss[i][5], 5, err.Error())
			return false
		}
		v.Param4 = int32(intv)
		// EventCount
		intv, err = strconv.Atoi(ss[i][6])
		if err != nil {
			log.Printf("table Subactive convert column EventCount value %v with row %v err %v", ss[i][6], 6, err.Error())
			return false
		}
		v.EventCount = int32(intv)
		// Reward
		v.Reward = ss[i][7]
		if id <= 0 {
			continue
		}
		this.id2items[int32(id)] = &v
		this.items_array = append(this.items_array, &v)
   	}
	return true
}

func (this *SubactiveMgr) Get(id int32) *Subactive {
	return this.id2items[id]
}

func (this *SubactiveMgr) GetByIndex(idx int32) *Subactive {
	if int(idx) >= len(this.items_array) {
		return nil
	}
	return this.items_array[idx]
}

func (this *SubactiveMgr) GetNum() int32 {
	return int32(len(this.items_array))
}

