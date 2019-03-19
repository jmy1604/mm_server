package csv_readers

import (
	"encoding/csv"
	"io/ioutil"
	"log"
	"strconv"
	"strings"
)

type Itemupgrade struct {
	UpgradeID int32
	ItemID int32
	UpgradeType int32
	ResultDropID int32
	ResCondtion string
}

type ItemupgradeMgr struct {
	id2items map[int32]*Itemupgrade
	items_array []*Itemupgrade
}

func (this *ItemupgradeMgr) Read(file_path_name string) bool {
	if file_path_name == "" {
		file_path_name = "../game_csv/itemupgrade.csv"
	}
	cs, err := ioutil.ReadFile(file_path_name)
	if err != nil {
		log.Printf("ItemupgradeMgr.Read err: %v", err.Error())
		return false
	}

 	r := csv.NewReader(strings.NewReader(string(cs)))
	ss, _ := r.ReadAll()
   	sz := len(ss)
	this.id2items = make(map[int32]*Itemupgrade)
    for i := int32(1); i < int32(sz); i++ {
		//if i < 5 {
		//	continue
		//}
		var v Itemupgrade
		var intv, id int
		// UpgradeID
		intv, err = strconv.Atoi(ss[i][0])
		if err != nil {
			log.Printf("table Itemupgrade convert column UpgradeID value %v with row %v err %v", ss[i][0], 0, err.Error())
			return false
		}
		v.UpgradeID = int32(intv)
		id = intv
		// ItemID
		intv, err = strconv.Atoi(ss[i][1])
		if err != nil {
			log.Printf("table Itemupgrade convert column ItemID value %v with row %v err %v", ss[i][1], 1, err.Error())
			return false
		}
		v.ItemID = int32(intv)
		// UpgradeType
		intv, err = strconv.Atoi(ss[i][2])
		if err != nil {
			log.Printf("table Itemupgrade convert column UpgradeType value %v with row %v err %v", ss[i][2], 2, err.Error())
			return false
		}
		v.UpgradeType = int32(intv)
		// ResultDropID
		intv, err = strconv.Atoi(ss[i][3])
		if err != nil {
			log.Printf("table Itemupgrade convert column ResultDropID value %v with row %v err %v", ss[i][3], 3, err.Error())
			return false
		}
		v.ResultDropID = int32(intv)
		// ResCondtion
		v.ResCondtion = ss[i][4]
		if id <= 0 {
			continue
		}
		this.id2items[int32(id)] = &v
		this.items_array = append(this.items_array, &v)
   	}
	return true
}

func (this *ItemupgradeMgr) Get(id int32) *Itemupgrade {
	return this.id2items[id]
}

func (this *ItemupgradeMgr) GetByIndex(idx int32) *Itemupgrade {
	if int(idx) >= len(this.items_array) {
		return nil
	}
	return this.items_array[idx]
}

func (this *ItemupgradeMgr) GetNum() int32 {
	return int32(len(this.items_array))
}

