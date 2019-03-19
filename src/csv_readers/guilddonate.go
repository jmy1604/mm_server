package csv_readers

import (
	"encoding/csv"
	"io/ioutil"
	"log"
	"strconv"
	"strings"
)

type Guilddonate struct {
	ItemID int32
	RequestNum int32
	DonateRewardItem string
	LimitScore int32
}

type GuilddonateMgr struct {
	id2items map[int32]*Guilddonate
	items_array []*Guilddonate
}

func (this *GuilddonateMgr) Read(file_path_name string) bool {
	if file_path_name == "" {
		file_path_name = "../game_csv/guilddonate.csv"
	}
	cs, err := ioutil.ReadFile(file_path_name)
	if err != nil {
		log.Printf("GuilddonateMgr.Read err: %v", err.Error())
		return false
	}

 	r := csv.NewReader(strings.NewReader(string(cs)))
	ss, _ := r.ReadAll()
   	sz := len(ss)
	this.id2items = make(map[int32]*Guilddonate)
    for i := int32(1); i < int32(sz); i++ {
		//if i < 5 {
		//	continue
		//}
		var v Guilddonate
		var intv, id int
		// ItemID
		intv, err = strconv.Atoi(ss[i][0])
		if err != nil {
			log.Printf("table Guilddonate convert column ItemID value %v with row %v err %v", ss[i][0], 0, err.Error())
			return false
		}
		v.ItemID = int32(intv)
		id = intv
		// RequestNum
		intv, err = strconv.Atoi(ss[i][1])
		if err != nil {
			log.Printf("table Guilddonate convert column RequestNum value %v with row %v err %v", ss[i][1], 1, err.Error())
			return false
		}
		v.RequestNum = int32(intv)
		// DonateRewardItem
		v.DonateRewardItem = ss[i][2]
		// LimitScore
		intv, err = strconv.Atoi(ss[i][3])
		if err != nil {
			log.Printf("table Guilddonate convert column LimitScore value %v with row %v err %v", ss[i][3], 3, err.Error())
			return false
		}
		v.LimitScore = int32(intv)
		if id <= 0 {
			continue
		}
		this.id2items[int32(id)] = &v
		this.items_array = append(this.items_array, &v)
   	}
	return true
}

func (this *GuilddonateMgr) Get(id int32) *Guilddonate {
	return this.id2items[id]
}

func (this *GuilddonateMgr) GetByIndex(idx int32) *Guilddonate {
	if int(idx) >= len(this.items_array) {
		return nil
	}
	return this.items_array[idx]
}

func (this *GuilddonateMgr) GetNum() int32 {
	return int32(len(this.items_array))
}

