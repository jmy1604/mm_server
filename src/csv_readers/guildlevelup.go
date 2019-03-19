package csv_readers

import (
	"encoding/csv"
	"io/ioutil"
	"log"
	"strconv"
	"strings"
)

type Guildlevelup struct {
	Level int32
	Exp int32
	Members int32
}

type GuildlevelupMgr struct {
	id2items map[int32]*Guildlevelup
	items_array []*Guildlevelup
}

func (this *GuildlevelupMgr) Read(file_path_name string) bool {
	if file_path_name == "" {
		file_path_name = "../game_csv/guildlevelup.csv"
	}
	cs, err := ioutil.ReadFile(file_path_name)
	if err != nil {
		log.Printf("GuildlevelupMgr.Read err: %v", err.Error())
		return false
	}

 	r := csv.NewReader(strings.NewReader(string(cs)))
	ss, _ := r.ReadAll()
   	sz := len(ss)
	this.id2items = make(map[int32]*Guildlevelup)
    for i := int32(1); i < int32(sz); i++ {
		//if i < 5 {
		//	continue
		//}
		var v Guildlevelup
		var intv, id int
		// Level
		intv, err = strconv.Atoi(ss[i][0])
		if err != nil {
			log.Printf("table Guildlevelup convert column Level value %v with row %v err %v", ss[i][0], 0, err.Error())
			return false
		}
		v.Level = int32(intv)
		id = intv
		// Exp
		intv, err = strconv.Atoi(ss[i][1])
		if err != nil {
			log.Printf("table Guildlevelup convert column Exp value %v with row %v err %v", ss[i][1], 1, err.Error())
			return false
		}
		v.Exp = int32(intv)
		// Members
		intv, err = strconv.Atoi(ss[i][2])
		if err != nil {
			log.Printf("table Guildlevelup convert column Members value %v with row %v err %v", ss[i][2], 2, err.Error())
			return false
		}
		v.Members = int32(intv)
		if id <= 0 {
			continue
		}
		this.id2items[int32(id)] = &v
		this.items_array = append(this.items_array, &v)
   	}
	return true
}

func (this *GuildlevelupMgr) Get(id int32) *Guildlevelup {
	return this.id2items[id]
}

func (this *GuildlevelupMgr) GetByIndex(idx int32) *Guildlevelup {
	if int(idx) >= len(this.items_array) {
		return nil
	}
	return this.items_array[idx]
}

func (this *GuildlevelupMgr) GetNum() int32 {
	return int32(len(this.items_array))
}

