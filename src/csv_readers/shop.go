package csv_readers

import (
	"encoding/csv"
	"io/ioutil"
	"log"
	"strconv"
	"strings"
)

type Shop struct {
	ID int32
	ShopType int32
	ShopMaxSlot int32
	AutoRefreshTime string
	FreeRefreshTime int32
	RefreshRes string
}

type ShopMgr struct {
	id2items map[int32]*Shop
	items_array []*Shop
}

func (this *ShopMgr) Read(file_path_name string) bool {
	if file_path_name == "" {
		file_path_name = "../game_csv/shop.csv"
	}
	cs, err := ioutil.ReadFile(file_path_name)
	if err != nil {
		log.Printf("ShopMgr.Read err: %v", err.Error())
		return false
	}

 	r := csv.NewReader(strings.NewReader(string(cs)))
	ss, _ := r.ReadAll()
   	sz := len(ss)
	this.id2items = make(map[int32]*Shop)
    for i := int32(1); i < int32(sz); i++ {
		//if i < 5 {
		//	continue
		//}
		var v Shop
		var intv, id int
		// ID
		intv, err = strconv.Atoi(ss[i][0])
		if err != nil {
			log.Printf("table Shop convert column ID value %v with row %v err %v", ss[i][0], 0, err.Error())
			return false
		}
		v.ID = int32(intv)
		id = intv
		// ShopType
		intv, err = strconv.Atoi(ss[i][1])
		if err != nil {
			log.Printf("table Shop convert column ShopType value %v with row %v err %v", ss[i][1], 1, err.Error())
			return false
		}
		v.ShopType = int32(intv)
		// ShopMaxSlot
		intv, err = strconv.Atoi(ss[i][2])
		if err != nil {
			log.Printf("table Shop convert column ShopMaxSlot value %v with row %v err %v", ss[i][2], 2, err.Error())
			return false
		}
		v.ShopMaxSlot = int32(intv)
		// AutoRefreshTime
		v.AutoRefreshTime = ss[i][3]
		// FreeRefreshTime
		intv, err = strconv.Atoi(ss[i][4])
		if err != nil {
			log.Printf("table Shop convert column FreeRefreshTime value %v with row %v err %v", ss[i][4], 4, err.Error())
			return false
		}
		v.FreeRefreshTime = int32(intv)
		// RefreshRes
		v.RefreshRes = ss[i][5]
		if id <= 0 {
			continue
		}
		this.id2items[int32(id)] = &v
		this.items_array = append(this.items_array, &v)
   	}
	return true
}

func (this *ShopMgr) Get(id int32) *Shop {
	return this.id2items[id]
}

func (this *ShopMgr) GetByIndex(idx int32) *Shop {
	if int(idx) >= len(this.items_array) {
		return nil
	}
	return this.items_array[idx]
}

func (this *ShopMgr) GetNum() int32 {
	return int32(len(this.items_array))
}

