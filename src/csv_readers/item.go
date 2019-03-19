package csv_readers

import (
	"encoding/csv"
	"io/ioutil"
	"log"
	"strconv"
	"strings"
)

type Item struct {
	ID int32
	ItemType int32
	EquipType int32
	SellReward string
	Quality int32
	ShowStar int32
	EquipAttr string
	EquipSkill string
	ComposeNum int32
	ComposeType int32
	ComposeDropID int32
	BattlePower int32
	SuitID int32
}

type ItemMgr struct {
	id2items map[int32]*Item
	items_array []*Item
}

func (this *ItemMgr) Read(file_path_name string) bool {
	if file_path_name == "" {
		file_path_name = "../game_csv/item.csv"
	}
	cs, err := ioutil.ReadFile(file_path_name)
	if err != nil {
		log.Printf("ItemMgr.Read err: %v", err.Error())
		return false
	}

 	r := csv.NewReader(strings.NewReader(string(cs)))
	ss, _ := r.ReadAll()
   	sz := len(ss)
	this.id2items = make(map[int32]*Item)
    for i := int32(1); i < int32(sz); i++ {
		//if i < 5 {
		//	continue
		//}
		var v Item
		var intv, id int
		// ID
		intv, err = strconv.Atoi(ss[i][0])
		if err != nil {
			log.Printf("table Item convert column ID value %v with row %v err %v", ss[i][0], 0, err.Error())
			return false
		}
		v.ID = int32(intv)
		id = intv
		// ItemType
		intv, err = strconv.Atoi(ss[i][1])
		if err != nil {
			log.Printf("table Item convert column ItemType value %v with row %v err %v", ss[i][1], 1, err.Error())
			return false
		}
		v.ItemType = int32(intv)
		// EquipType
		intv, err = strconv.Atoi(ss[i][2])
		if err != nil {
			log.Printf("table Item convert column EquipType value %v with row %v err %v", ss[i][2], 2, err.Error())
			return false
		}
		v.EquipType = int32(intv)
		// SellReward
		v.SellReward = ss[i][3]
		// Quality
		intv, err = strconv.Atoi(ss[i][4])
		if err != nil {
			log.Printf("table Item convert column Quality value %v with row %v err %v", ss[i][4], 4, err.Error())
			return false
		}
		v.Quality = int32(intv)
		// ShowStar
		intv, err = strconv.Atoi(ss[i][5])
		if err != nil {
			log.Printf("table Item convert column ShowStar value %v with row %v err %v", ss[i][5], 5, err.Error())
			return false
		}
		v.ShowStar = int32(intv)
		// EquipAttr
		v.EquipAttr = ss[i][6]
		// EquipSkill
		v.EquipSkill = ss[i][7]
		// ComposeNum
		intv, err = strconv.Atoi(ss[i][8])
		if err != nil {
			log.Printf("table Item convert column ComposeNum value %v with row %v err %v", ss[i][8], 8, err.Error())
			return false
		}
		v.ComposeNum = int32(intv)
		// ComposeType
		intv, err = strconv.Atoi(ss[i][9])
		if err != nil {
			log.Printf("table Item convert column ComposeType value %v with row %v err %v", ss[i][9], 9, err.Error())
			return false
		}
		v.ComposeType = int32(intv)
		// ComposeDropID
		intv, err = strconv.Atoi(ss[i][10])
		if err != nil {
			log.Printf("table Item convert column ComposeDropID value %v with row %v err %v", ss[i][10], 10, err.Error())
			return false
		}
		v.ComposeDropID = int32(intv)
		// BattlePower
		intv, err = strconv.Atoi(ss[i][11])
		if err != nil {
			log.Printf("table Item convert column BattlePower value %v with row %v err %v", ss[i][11], 11, err.Error())
			return false
		}
		v.BattlePower = int32(intv)
		// SuitID
		intv, err = strconv.Atoi(ss[i][12])
		if err != nil {
			log.Printf("table Item convert column SuitID value %v with row %v err %v", ss[i][12], 12, err.Error())
			return false
		}
		v.SuitID = int32(intv)
		if id <= 0 {
			continue
		}
		this.id2items[int32(id)] = &v
		this.items_array = append(this.items_array, &v)
   	}
	return true
}

func (this *ItemMgr) Get(id int32) *Item {
	return this.id2items[id]
}

func (this *ItemMgr) GetByIndex(idx int32) *Item {
	if int(idx) >= len(this.items_array) {
		return nil
	}
	return this.items_array[idx]
}

func (this *ItemMgr) GetNum() int32 {
	return int32(len(this.items_array))
}

