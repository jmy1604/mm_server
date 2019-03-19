package csv_readers

import (
	"encoding/csv"
	"io/ioutil"
	"log"
	"strconv"
	"strings"
)

type Pay struct {
	ID int32
	ActivePay int32
	BundleID string
	GemRewardFirst int32
	GemReward int32
	MonthCardDay int32
	MonthCardReward int32
	ItemReward string
	RecordGold string
}

type PayMgr struct {
	id2items map[int32]*Pay
	items_array []*Pay
}

func (this *PayMgr) Read(file_path_name string) bool {
	if file_path_name == "" {
		file_path_name = "../game_csv/pay.csv"
	}
	cs, err := ioutil.ReadFile(file_path_name)
	if err != nil {
		log.Printf("PayMgr.Read err: %v", err.Error())
		return false
	}

 	r := csv.NewReader(strings.NewReader(string(cs)))
	ss, _ := r.ReadAll()
   	sz := len(ss)
	this.id2items = make(map[int32]*Pay)
    for i := int32(1); i < int32(sz); i++ {
		//if i < 5 {
		//	continue
		//}
		var v Pay
		var intv, id int
		// ID
		intv, err = strconv.Atoi(ss[i][0])
		if err != nil {
			log.Printf("table Pay convert column ID value %v with row %v err %v", ss[i][0], 0, err.Error())
			return false
		}
		v.ID = int32(intv)
		id = intv
		// ActivePay
		intv, err = strconv.Atoi(ss[i][1])
		if err != nil {
			log.Printf("table Pay convert column ActivePay value %v with row %v err %v", ss[i][1], 1, err.Error())
			return false
		}
		v.ActivePay = int32(intv)
		// BundleID
		v.BundleID = ss[i][2]
		// GemRewardFirst
		intv, err = strconv.Atoi(ss[i][3])
		if err != nil {
			log.Printf("table Pay convert column GemRewardFirst value %v with row %v err %v", ss[i][3], 3, err.Error())
			return false
		}
		v.GemRewardFirst = int32(intv)
		// GemReward
		intv, err = strconv.Atoi(ss[i][4])
		if err != nil {
			log.Printf("table Pay convert column GemReward value %v with row %v err %v", ss[i][4], 4, err.Error())
			return false
		}
		v.GemReward = int32(intv)
		// MonthCardDay
		intv, err = strconv.Atoi(ss[i][5])
		if err != nil {
			log.Printf("table Pay convert column MonthCardDay value %v with row %v err %v", ss[i][5], 5, err.Error())
			return false
		}
		v.MonthCardDay = int32(intv)
		// MonthCardReward
		intv, err = strconv.Atoi(ss[i][6])
		if err != nil {
			log.Printf("table Pay convert column MonthCardReward value %v with row %v err %v", ss[i][6], 6, err.Error())
			return false
		}
		v.MonthCardReward = int32(intv)
		// ItemReward
		v.ItemReward = ss[i][7]
		// RecordGold
		v.RecordGold = ss[i][8]
		if id <= 0 {
			continue
		}
		this.id2items[int32(id)] = &v
		this.items_array = append(this.items_array, &v)
   	}
	return true
}

func (this *PayMgr) Get(id int32) *Pay {
	return this.id2items[id]
}

func (this *PayMgr) GetByIndex(idx int32) *Pay {
	if int(idx) >= len(this.items_array) {
		return nil
	}
	return this.items_array[idx]
}

func (this *PayMgr) GetNum() int32 {
	return int32(len(this.items_array))
}

