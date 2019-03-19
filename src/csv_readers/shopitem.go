package csv_readers

import (
	"encoding/csv"
	"io/ioutil"
	"log"
	"strconv"
	"strings"
)

type Shopitem struct {
	GoodID int32
	ShopID int32
	LevelMin int32
	LevelMax int32
	ItemList string
	BuyCost string
	StockNum int32
	RandomWeight int32
}

type ShopitemMgr struct {
	id2items map[int32]*Shopitem
	items_array []*Shopitem
}

func (this *ShopitemMgr) Read(file_path_name string) bool {
	if file_path_name == "" {
		file_path_name = "../game_csv/shopitem.csv"
	}
	cs, err := ioutil.ReadFile(file_path_name)
	if err != nil {
		log.Printf("ShopitemMgr.Read err: %v", err.Error())
		return false
	}

 	r := csv.NewReader(strings.NewReader(string(cs)))
	ss, _ := r.ReadAll()
   	sz := len(ss)
	this.id2items = make(map[int32]*Shopitem)
    for i := int32(1); i < int32(sz); i++ {
		//if i < 5 {
		//	continue
		//}
		var v Shopitem
		var intv, id int
		// GoodID
		intv, err = strconv.Atoi(ss[i][0])
		if err != nil {
			log.Printf("table Shopitem convert column GoodID value %v with row %v err %v", ss[i][0], 0, err.Error())
			return false
		}
		v.GoodID = int32(intv)
		id = intv
		// ShopID
		intv, err = strconv.Atoi(ss[i][1])
		if err != nil {
			log.Printf("table Shopitem convert column ShopID value %v with row %v err %v", ss[i][1], 1, err.Error())
			return false
		}
		v.ShopID = int32(intv)
		// LevelMin
		intv, err = strconv.Atoi(ss[i][2])
		if err != nil {
			log.Printf("table Shopitem convert column LevelMin value %v with row %v err %v", ss[i][2], 2, err.Error())
			return false
		}
		v.LevelMin = int32(intv)
		// LevelMax
		intv, err = strconv.Atoi(ss[i][3])
		if err != nil {
			log.Printf("table Shopitem convert column LevelMax value %v with row %v err %v", ss[i][3], 3, err.Error())
			return false
		}
		v.LevelMax = int32(intv)
		// ItemList
		v.ItemList = ss[i][4]
		// BuyCost
		v.BuyCost = ss[i][5]
		// StockNum
		intv, err = strconv.Atoi(ss[i][6])
		if err != nil {
			log.Printf("table Shopitem convert column StockNum value %v with row %v err %v", ss[i][6], 6, err.Error())
			return false
		}
		v.StockNum = int32(intv)
		// RandomWeight
		intv, err = strconv.Atoi(ss[i][7])
		if err != nil {
			log.Printf("table Shopitem convert column RandomWeight value %v with row %v err %v", ss[i][7], 7, err.Error())
			return false
		}
		v.RandomWeight = int32(intv)
		if id <= 0 {
			continue
		}
		this.id2items[int32(id)] = &v
		this.items_array = append(this.items_array, &v)
   	}
	return true
}

func (this *ShopitemMgr) Get(id int32) *Shopitem {
	return this.id2items[id]
}

func (this *ShopitemMgr) GetByIndex(idx int32) *Shopitem {
	if int(idx) >= len(this.items_array) {
		return nil
	}
	return this.items_array[idx]
}

func (this *ShopitemMgr) GetNum() int32 {
	return int32(len(this.items_array))
}

