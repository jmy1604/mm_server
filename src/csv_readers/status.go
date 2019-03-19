package csv_readers

import (
	"encoding/csv"
	"io/ioutil"
	"log"
	"strconv"
	"strings"
)

type Status struct {
	BuffID int32
	Effect string
	Type int32
	ResistCountMax int32
	MutexType int32
	ResistMutexType string
	CancelMutexType string
	ResistMutexID string
	CancelMutexID string
}

type StatusMgr struct {
	id2items map[int32]*Status
	items_array []*Status
}

func (this *StatusMgr) Read(file_path_name string) bool {
	if file_path_name == "" {
		file_path_name = "../game_csv/status.csv"
	}
	cs, err := ioutil.ReadFile(file_path_name)
	if err != nil {
		log.Printf("StatusMgr.Read err: %v", err.Error())
		return false
	}

 	r := csv.NewReader(strings.NewReader(string(cs)))
	ss, _ := r.ReadAll()
   	sz := len(ss)
	this.id2items = make(map[int32]*Status)
    for i := int32(1); i < int32(sz); i++ {
		//if i < 5 {
		//	continue
		//}
		var v Status
		var intv, id int
		// BuffID
		intv, err = strconv.Atoi(ss[i][0])
		if err != nil {
			log.Printf("table Status convert column BuffID value %v with row %v err %v", ss[i][0], 0, err.Error())
			return false
		}
		v.BuffID = int32(intv)
		id = intv
		// Effect
		v.Effect = ss[i][1]
		// Type
		intv, err = strconv.Atoi(ss[i][2])
		if err != nil {
			log.Printf("table Status convert column Type value %v with row %v err %v", ss[i][2], 2, err.Error())
			return false
		}
		v.Type = int32(intv)
		// ResistCountMax
		intv, err = strconv.Atoi(ss[i][3])
		if err != nil {
			log.Printf("table Status convert column ResistCountMax value %v with row %v err %v", ss[i][3], 3, err.Error())
			return false
		}
		v.ResistCountMax = int32(intv)
		// MutexType
		intv, err = strconv.Atoi(ss[i][4])
		if err != nil {
			log.Printf("table Status convert column MutexType value %v with row %v err %v", ss[i][4], 4, err.Error())
			return false
		}
		v.MutexType = int32(intv)
		// ResistMutexType
		v.ResistMutexType = ss[i][5]
		// CancelMutexType
		v.CancelMutexType = ss[i][6]
		// ResistMutexID
		v.ResistMutexID = ss[i][7]
		// CancelMutexID
		v.CancelMutexID = ss[i][8]
		if id <= 0 {
			continue
		}
		this.id2items[int32(id)] = &v
		this.items_array = append(this.items_array, &v)
   	}
	return true
}

func (this *StatusMgr) Get(id int32) *Status {
	return this.id2items[id]
}

func (this *StatusMgr) GetByIndex(idx int32) *Status {
	if int(idx) >= len(this.items_array) {
		return nil
	}
	return this.items_array[idx]
}

func (this *StatusMgr) GetNum() int32 {
	return int32(len(this.items_array))
}

