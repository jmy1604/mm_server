package csv_readers

import (
	"encoding/csv"
	"io/ioutil"
	"log"
	"strconv"
	"strings"
)

type Mainactive struct {
	MainActiveID int32
	ActiveType int32
	EventID int32
	StartTime string
	EndTime string
	SubActiveList string
	RewardMailID int32
}

type MainactiveMgr struct {
	id2items map[int32]*Mainactive
	items_array []*Mainactive
}

func (this *MainactiveMgr) Read(file_path_name string) bool {
	if file_path_name == "" {
		file_path_name = "../game_csv/mainactive.csv"
	}
	cs, err := ioutil.ReadFile(file_path_name)
	if err != nil {
		log.Printf("MainactiveMgr.Read err: %v", err.Error())
		return false
	}

 	r := csv.NewReader(strings.NewReader(string(cs)))
	ss, _ := r.ReadAll()
   	sz := len(ss)
	this.id2items = make(map[int32]*Mainactive)
    for i := int32(1); i < int32(sz); i++ {
		//if i < 5 {
		//	continue
		//}
		var v Mainactive
		var intv, id int
		// MainActiveID
		intv, err = strconv.Atoi(ss[i][0])
		if err != nil {
			log.Printf("table Mainactive convert column MainActiveID value %v with row %v err %v", ss[i][0], 0, err.Error())
			return false
		}
		v.MainActiveID = int32(intv)
		id = intv
		// ActiveType
		intv, err = strconv.Atoi(ss[i][1])
		if err != nil {
			log.Printf("table Mainactive convert column ActiveType value %v with row %v err %v", ss[i][1], 1, err.Error())
			return false
		}
		v.ActiveType = int32(intv)
		// EventID
		intv, err = strconv.Atoi(ss[i][2])
		if err != nil {
			log.Printf("table Mainactive convert column EventID value %v with row %v err %v", ss[i][2], 2, err.Error())
			return false
		}
		v.EventID = int32(intv)
		// StartTime
		v.StartTime = ss[i][3]
		// EndTime
		v.EndTime = ss[i][4]
		// SubActiveList
		v.SubActiveList = ss[i][5]
		// RewardMailID
		intv, err = strconv.Atoi(ss[i][6])
		if err != nil {
			log.Printf("table Mainactive convert column RewardMailID value %v with row %v err %v", ss[i][6], 6, err.Error())
			return false
		}
		v.RewardMailID = int32(intv)
		if id <= 0 {
			continue
		}
		this.id2items[int32(id)] = &v
		this.items_array = append(this.items_array, &v)
   	}
	return true
}

func (this *MainactiveMgr) Get(id int32) *Mainactive {
	return this.id2items[id]
}

func (this *MainactiveMgr) GetByIndex(idx int32) *Mainactive {
	if int(idx) >= len(this.items_array) {
		return nil
	}
	return this.items_array[idx]
}

func (this *MainactiveMgr) GetNum() int32 {
	return int32(len(this.items_array))
}

