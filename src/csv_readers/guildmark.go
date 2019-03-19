package csv_readers

import (
	"encoding/csv"
	"io/ioutil"
	"log"
	"strconv"
	"strings"
)

type Guildmark struct {
	ID int32
}

type GuildmarkMgr struct {
	id2items map[int32]*Guildmark
	items_array []*Guildmark
}

func (this *GuildmarkMgr) Read(file_path_name string) bool {
	if file_path_name == "" {
		file_path_name = "../game_csv/guildmark.csv"
	}
	cs, err := ioutil.ReadFile(file_path_name)
	if err != nil {
		log.Printf("GuildmarkMgr.Read err: %v", err.Error())
		return false
	}

 	r := csv.NewReader(strings.NewReader(string(cs)))
	ss, _ := r.ReadAll()
   	sz := len(ss)
	this.id2items = make(map[int32]*Guildmark)
    for i := int32(1); i < int32(sz); i++ {
		//if i < 5 {
		//	continue
		//}
		var v Guildmark
		var intv, id int
		// ID
		intv, err = strconv.Atoi(ss[i][0])
		if err != nil {
			log.Printf("table Guildmark convert column ID value %v with row %v err %v", ss[i][0], 0, err.Error())
			return false
		}
		v.ID = int32(intv)
		id = intv
		if id <= 0 {
			continue
		}
		this.id2items[int32(id)] = &v
		this.items_array = append(this.items_array, &v)
   	}
	return true
}

func (this *GuildmarkMgr) Get(id int32) *Guildmark {
	return this.id2items[id]
}

func (this *GuildmarkMgr) GetByIndex(idx int32) *Guildmark {
	if int(idx) >= len(this.items_array) {
		return nil
	}
	return this.items_array[idx]
}

func (this *GuildmarkMgr) GetNum() int32 {
	return int32(len(this.items_array))
}

