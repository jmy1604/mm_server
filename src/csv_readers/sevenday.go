package csv_readers

import (
	"encoding/csv"
	"io/ioutil"
	"log"
	"strconv"
	"strings"
)

type Sevenday struct {
	TotalIndex int32
	Reward string
}

type SevendayMgr struct {
	id2items map[int32]*Sevenday
	items_array []*Sevenday
}

func (this *SevendayMgr) Read(file_path_name string) bool {
	if file_path_name == "" {
		file_path_name = "../game_csv/sevenday.csv"
	}
	cs, err := ioutil.ReadFile(file_path_name)
	if err != nil {
		log.Printf("SevendayMgr.Read err: %v", err.Error())
		return false
	}

 	r := csv.NewReader(strings.NewReader(string(cs)))
	ss, _ := r.ReadAll()
   	sz := len(ss)
	this.id2items = make(map[int32]*Sevenday)
    for i := int32(1); i < int32(sz); i++ {
		//if i < 5 {
		//	continue
		//}
		var v Sevenday
		var intv, id int
		// TotalIndex
		intv, err = strconv.Atoi(ss[i][0])
		if err != nil {
			log.Printf("table Sevenday convert column TotalIndex value %v with row %v err %v", ss[i][0], 0, err.Error())
			return false
		}
		v.TotalIndex = int32(intv)
		id = intv
		// Reward
		v.Reward = ss[i][1]
		if id <= 0 {
			continue
		}
		this.id2items[int32(id)] = &v
		this.items_array = append(this.items_array, &v)
   	}
	return true
}

func (this *SevendayMgr) Get(id int32) *Sevenday {
	return this.id2items[id]
}

func (this *SevendayMgr) GetByIndex(idx int32) *Sevenday {
	if int(idx) >= len(this.items_array) {
		return nil
	}
	return this.items_array[idx]
}

func (this *SevendayMgr) GetNum() int32 {
	return int32(len(this.items_array))
}

