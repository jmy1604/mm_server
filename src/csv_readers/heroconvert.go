package csv_readers

import (
	"encoding/csv"
	"io/ioutil"
	"log"
	"strconv"
	"strings"
)

type Heroconvert struct {
	ConvertGroupID int32
	HeroID int32
	Weight int32
}

type HeroconvertMgr struct {
	id2items map[int32]*Heroconvert
	items_array []*Heroconvert
}

func (this *HeroconvertMgr) Read(file_path_name string) bool {
	if file_path_name == "" {
		file_path_name = "../game_csv/heroconvert.csv"
	}
	cs, err := ioutil.ReadFile(file_path_name)
	if err != nil {
		log.Printf("HeroconvertMgr.Read err: %v", err.Error())
		return false
	}

 	r := csv.NewReader(strings.NewReader(string(cs)))
	ss, _ := r.ReadAll()
   	sz := len(ss)
	this.id2items = make(map[int32]*Heroconvert)
    for i := int32(1); i < int32(sz); i++ {
		//if i < 5 {
		//	continue
		//}
		var v Heroconvert
		var intv, id int
		// ConvertGroupID
		intv, err = strconv.Atoi(ss[i][0])
		if err != nil {
			log.Printf("table Heroconvert convert column ConvertGroupID value %v with row %v err %v", ss[i][0], 0, err.Error())
			return false
		}
		v.ConvertGroupID = int32(intv)
		id = intv
		// HeroID
		intv, err = strconv.Atoi(ss[i][1])
		if err != nil {
			log.Printf("table Heroconvert convert column HeroID value %v with row %v err %v", ss[i][1], 1, err.Error())
			return false
		}
		v.HeroID = int32(intv)
		// Weight
		intv, err = strconv.Atoi(ss[i][2])
		if err != nil {
			log.Printf("table Heroconvert convert column Weight value %v with row %v err %v", ss[i][2], 2, err.Error())
			return false
		}
		v.Weight = int32(intv)
		if id <= 0 {
			continue
		}
		this.id2items[int32(id)] = &v
		this.items_array = append(this.items_array, &v)
   	}
	return true
}

func (this *HeroconvertMgr) Get(id int32) *Heroconvert {
	return this.id2items[id]
}

func (this *HeroconvertMgr) GetByIndex(idx int32) *Heroconvert {
	if int(idx) >= len(this.items_array) {
		return nil
	}
	return this.items_array[idx]
}

func (this *HeroconvertMgr) GetNum() int32 {
	return int32(len(this.items_array))
}

