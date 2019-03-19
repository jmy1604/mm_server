package csv_readers

import (
	"encoding/csv"
	"io/ioutil"
	"log"
	"strconv"
	"strings"
)

type Mission struct {
	Id int32
	Type int32
	EventId int32
	EventParam int32
	CompleteNum int32
	Prev int32
	Next int32
	Reward string
	Hyperlink int32
}

type MissionMgr struct {
	id2items map[int32]*Mission
	items_array []*Mission
}

func (this *MissionMgr) Read(file_path_name string) bool {
	if file_path_name == "" {
		file_path_name = "../game_csv/mission.csv"
	}
	cs, err := ioutil.ReadFile(file_path_name)
	if err != nil {
		log.Printf("MissionMgr.Read err: %v", err.Error())
		return false
	}

 	r := csv.NewReader(strings.NewReader(string(cs)))
	ss, _ := r.ReadAll()
   	sz := len(ss)
	this.id2items = make(map[int32]*Mission)
    for i := int32(1); i < int32(sz); i++ {
		//if i < 5 {
		//	continue
		//}
		var v Mission
		var intv, id int
		// Id
		intv, err = strconv.Atoi(ss[i][0])
		if err != nil {
			log.Printf("table Mission convert column Id value %v with row %v err %v", ss[i][0], 0, err.Error())
			return false
		}
		v.Id = int32(intv)
		id = intv
		// Type
		intv, err = strconv.Atoi(ss[i][1])
		if err != nil {
			log.Printf("table Mission convert column Type value %v with row %v err %v", ss[i][1], 1, err.Error())
			return false
		}
		v.Type = int32(intv)
		// EventId
		intv, err = strconv.Atoi(ss[i][2])
		if err != nil {
			log.Printf("table Mission convert column EventId value %v with row %v err %v", ss[i][2], 2, err.Error())
			return false
		}
		v.EventId = int32(intv)
		// EventParam
		intv, err = strconv.Atoi(ss[i][3])
		if err != nil {
			log.Printf("table Mission convert column EventParam value %v with row %v err %v", ss[i][3], 3, err.Error())
			return false
		}
		v.EventParam = int32(intv)
		// CompleteNum
		intv, err = strconv.Atoi(ss[i][4])
		if err != nil {
			log.Printf("table Mission convert column CompleteNum value %v with row %v err %v", ss[i][4], 4, err.Error())
			return false
		}
		v.CompleteNum = int32(intv)
		// Prev
		intv, err = strconv.Atoi(ss[i][5])
		if err != nil {
			log.Printf("table Mission convert column Prev value %v with row %v err %v", ss[i][5], 5, err.Error())
			return false
		}
		v.Prev = int32(intv)
		// Next
		intv, err = strconv.Atoi(ss[i][6])
		if err != nil {
			log.Printf("table Mission convert column Next value %v with row %v err %v", ss[i][6], 6, err.Error())
			return false
		}
		v.Next = int32(intv)
		// Reward
		v.Reward = ss[i][7]
		// Hyperlink
		intv, err = strconv.Atoi(ss[i][8])
		if err != nil {
			log.Printf("table Mission convert column Hyperlink value %v with row %v err %v", ss[i][8], 8, err.Error())
			return false
		}
		v.Hyperlink = int32(intv)
		if id <= 0 {
			continue
		}
		this.id2items[int32(id)] = &v
		this.items_array = append(this.items_array, &v)
   	}
	return true
}

func (this *MissionMgr) Get(id int32) *Mission {
	return this.id2items[id]
}

func (this *MissionMgr) GetByIndex(idx int32) *Mission {
	if int(idx) >= len(this.items_array) {
		return nil
	}
	return this.items_array[idx]
}

func (this *MissionMgr) GetNum() int32 {
	return int32(len(this.items_array))
}

