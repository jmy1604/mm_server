package csv_readers

import (
	"encoding/csv"
	"io/ioutil"
	"log"
	"strconv"
	"strings"
)

type Suit struct {
	SuitID int32
	AttrSuit2 string
	AttrSuit3 string
	AttrSuit4 string
	BpowerSuit2 int32
	BpowerSuit3 int32
	BpowerSuit4 int32
}

type SuitMgr struct {
	id2items map[int32]*Suit
	items_array []*Suit
}

func (this *SuitMgr) Read(file_path_name string) bool {
	if file_path_name == "" {
		file_path_name = "../game_csv/suit.csv"
	}
	cs, err := ioutil.ReadFile(file_path_name)
	if err != nil {
		log.Printf("SuitMgr.Read err: %v", err.Error())
		return false
	}

 	r := csv.NewReader(strings.NewReader(string(cs)))
	ss, _ := r.ReadAll()
   	sz := len(ss)
	this.id2items = make(map[int32]*Suit)
    for i := int32(1); i < int32(sz); i++ {
		//if i < 5 {
		//	continue
		//}
		var v Suit
		var intv, id int
		// SuitID
		intv, err = strconv.Atoi(ss[i][0])
		if err != nil {
			log.Printf("table Suit convert column SuitID value %v with row %v err %v", ss[i][0], 0, err.Error())
			return false
		}
		v.SuitID = int32(intv)
		id = intv
		// AttrSuit2
		v.AttrSuit2 = ss[i][1]
		// AttrSuit3
		v.AttrSuit3 = ss[i][2]
		// AttrSuit4
		v.AttrSuit4 = ss[i][3]
		// BpowerSuit2
		intv, err = strconv.Atoi(ss[i][4])
		if err != nil {
			log.Printf("table Suit convert column BpowerSuit2 value %v with row %v err %v", ss[i][4], 4, err.Error())
			return false
		}
		v.BpowerSuit2 = int32(intv)
		// BpowerSuit3
		intv, err = strconv.Atoi(ss[i][5])
		if err != nil {
			log.Printf("table Suit convert column BpowerSuit3 value %v with row %v err %v", ss[i][5], 5, err.Error())
			return false
		}
		v.BpowerSuit3 = int32(intv)
		// BpowerSuit4
		intv, err = strconv.Atoi(ss[i][6])
		if err != nil {
			log.Printf("table Suit convert column BpowerSuit4 value %v with row %v err %v", ss[i][6], 6, err.Error())
			return false
		}
		v.BpowerSuit4 = int32(intv)
		if id <= 0 {
			continue
		}
		this.id2items[int32(id)] = &v
		this.items_array = append(this.items_array, &v)
   	}
	return true
}

func (this *SuitMgr) Get(id int32) *Suit {
	return this.id2items[id]
}

func (this *SuitMgr) GetByIndex(idx int32) *Suit {
	if int(idx) >= len(this.items_array) {
		return nil
	}
	return this.items_array[idx]
}

func (this *SuitMgr) GetNum() int32 {
	return int32(len(this.items_array))
}

