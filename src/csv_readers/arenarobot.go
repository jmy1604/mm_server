package csv_readers

import (
	"encoding/csv"
	"io/ioutil"
	"log"
	"strconv"
	"strings"
)

type Arenarobot struct {
	RobotID int32
	RobotLevel int32
	RobotHead int32
	RobotName string
	RobotCardList string
	RobotScore string
	IsExpenditonRobot int32
}

type ArenarobotMgr struct {
	id2items map[int32]*Arenarobot
	items_array []*Arenarobot
}

func (this *ArenarobotMgr) Read(file_path_name string) bool {
	if file_path_name == "" {
		file_path_name = "../game_csv/arenarobot.csv"
	}
	cs, err := ioutil.ReadFile(file_path_name)
	if err != nil {
		log.Printf("ArenarobotMgr.Read err: %v", err.Error())
		return false
	}

 	r := csv.NewReader(strings.NewReader(string(cs)))
	ss, _ := r.ReadAll()
   	sz := len(ss)
	this.id2items = make(map[int32]*Arenarobot)
    for i := int32(1); i < int32(sz); i++ {
		//if i < 5 {
		//	continue
		//}
		var v Arenarobot
		var intv, id int
		// RobotID
		intv, err = strconv.Atoi(ss[i][0])
		if err != nil {
			log.Printf("table Arenarobot convert column RobotID value %v with row %v err %v", ss[i][0], 0, err.Error())
			return false
		}
		v.RobotID = int32(intv)
		id = intv
		// RobotLevel
		intv, err = strconv.Atoi(ss[i][1])
		if err != nil {
			log.Printf("table Arenarobot convert column RobotLevel value %v with row %v err %v", ss[i][1], 1, err.Error())
			return false
		}
		v.RobotLevel = int32(intv)
		// RobotHead
		intv, err = strconv.Atoi(ss[i][2])
		if err != nil {
			log.Printf("table Arenarobot convert column RobotHead value %v with row %v err %v", ss[i][2], 2, err.Error())
			return false
		}
		v.RobotHead = int32(intv)
		// RobotName
		v.RobotName = ss[i][3]
		// RobotCardList
		v.RobotCardList = ss[i][4]
		// RobotScore
		v.RobotScore = ss[i][5]
		// IsExpenditonRobot
		intv, err = strconv.Atoi(ss[i][6])
		if err != nil {
			log.Printf("table Arenarobot convert column IsExpenditonRobot value %v with row %v err %v", ss[i][6], 6, err.Error())
			return false
		}
		v.IsExpenditonRobot = int32(intv)
		if id <= 0 {
			continue
		}
		this.id2items[int32(id)] = &v
		this.items_array = append(this.items_array, &v)
   	}
	return true
}

func (this *ArenarobotMgr) Get(id int32) *Arenarobot {
	return this.id2items[id]
}

func (this *ArenarobotMgr) GetByIndex(idx int32) *Arenarobot {
	if int(idx) >= len(this.items_array) {
		return nil
	}
	return this.items_array[idx]
}

func (this *ArenarobotMgr) GetNum() int32 {
	return int32(len(this.items_array))
}

