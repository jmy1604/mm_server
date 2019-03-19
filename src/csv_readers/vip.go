package csv_readers

import (
	"encoding/csv"
	"io/ioutil"
	"log"
	"strconv"
	"strings"
)

type Vip struct {
	VipLevel int32
	Money int32
	AccelTimes int32
	ActiveStageBuyTimes int32
	GoldFingerBonus int32
	HonorPointBonus int32
	MonthCardItemBonus string
	SearchTaskCount int32
}

type VipMgr struct {
	id2items map[int32]*Vip
	items_array []*Vip
}

func (this *VipMgr) Read(file_path_name string) bool {
	if file_path_name == "" {
		file_path_name = "../game_csv/vip.csv"
	}
	cs, err := ioutil.ReadFile(file_path_name)
	if err != nil {
		log.Printf("VipMgr.Read err: %v", err.Error())
		return false
	}

 	r := csv.NewReader(strings.NewReader(string(cs)))
	ss, _ := r.ReadAll()
   	sz := len(ss)
	this.id2items = make(map[int32]*Vip)
    for i := int32(1); i < int32(sz); i++ {
		//if i < 5 {
		//	continue
		//}
		var v Vip
		var intv, id int
		// VipLevel
		intv, err = strconv.Atoi(ss[i][0])
		if err != nil {
			log.Printf("table Vip convert column VipLevel value %v with row %v err %v", ss[i][0], 0, err.Error())
			return false
		}
		v.VipLevel = int32(intv)
		id = intv
		// Money
		intv, err = strconv.Atoi(ss[i][1])
		if err != nil {
			log.Printf("table Vip convert column Money value %v with row %v err %v", ss[i][1], 1, err.Error())
			return false
		}
		v.Money = int32(intv)
		// AccelTimes
		intv, err = strconv.Atoi(ss[i][2])
		if err != nil {
			log.Printf("table Vip convert column AccelTimes value %v with row %v err %v", ss[i][2], 2, err.Error())
			return false
		}
		v.AccelTimes = int32(intv)
		// ActiveStageBuyTimes
		intv, err = strconv.Atoi(ss[i][3])
		if err != nil {
			log.Printf("table Vip convert column ActiveStageBuyTimes value %v with row %v err %v", ss[i][3], 3, err.Error())
			return false
		}
		v.ActiveStageBuyTimes = int32(intv)
		// GoldFingerBonus
		intv, err = strconv.Atoi(ss[i][4])
		if err != nil {
			log.Printf("table Vip convert column GoldFingerBonus value %v with row %v err %v", ss[i][4], 4, err.Error())
			return false
		}
		v.GoldFingerBonus = int32(intv)
		// HonorPointBonus
		intv, err = strconv.Atoi(ss[i][5])
		if err != nil {
			log.Printf("table Vip convert column HonorPointBonus value %v with row %v err %v", ss[i][5], 5, err.Error())
			return false
		}
		v.HonorPointBonus = int32(intv)
		// MonthCardItemBonus
		v.MonthCardItemBonus = ss[i][6]
		// SearchTaskCount
		intv, err = strconv.Atoi(ss[i][7])
		if err != nil {
			log.Printf("table Vip convert column SearchTaskCount value %v with row %v err %v", ss[i][7], 7, err.Error())
			return false
		}
		v.SearchTaskCount = int32(intv)
		if id <= 0 {
			continue
		}
		this.id2items[int32(id)] = &v
		this.items_array = append(this.items_array, &v)
   	}
	return true
}

func (this *VipMgr) Get(id int32) *Vip {
	return this.id2items[id]
}

func (this *VipMgr) GetByIndex(idx int32) *Vip {
	if int(idx) >= len(this.items_array) {
		return nil
	}
	return this.items_array[idx]
}

func (this *VipMgr) GetNum() int32 {
	return int32(len(this.items_array))
}

