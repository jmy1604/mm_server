package csv_readers

import (
	"encoding/csv"
	"io/ioutil"
	"log"
	"strconv"
	"strings"
)

type Fusion struct {
	FormulaID int32
	FusionType int32
	ResultDropID int32
	MainCardID int32
	MainCardLevelCond int32
	ResCondtion string
	Cost1IDCond int32
	Cost1CampCond int32
	Cost1TypeCond int32
	Cost1StarCond int32
	Cost1NumCond int32
	Cost2IDCond int32
	Cost2CampCond int32
	Cost2TypeCond int32
	Cost2StarCond int32
	Cost2NumCond int32
	Cost3IDCond int32
	Cost3CampCond int32
	Cost3TypeCond int32
	Cost3StarCond int32
	Cost3NumCond int32
}

type FusionMgr struct {
	id2items map[int32]*Fusion
	items_array []*Fusion
}

func (this *FusionMgr) Read(file_path_name string) bool {
	if file_path_name == "" {
		file_path_name = "../game_csv/fusion.csv"
	}
	cs, err := ioutil.ReadFile(file_path_name)
	if err != nil {
		log.Printf("FusionMgr.Read err: %v", err.Error())
		return false
	}

 	r := csv.NewReader(strings.NewReader(string(cs)))
	ss, _ := r.ReadAll()
   	sz := len(ss)
	this.id2items = make(map[int32]*Fusion)
    for i := int32(1); i < int32(sz); i++ {
		//if i < 5 {
		//	continue
		//}
		var v Fusion
		var intv, id int
		// FormulaID
		intv, err = strconv.Atoi(ss[i][0])
		if err != nil {
			log.Printf("table Fusion convert column FormulaID value %v with row %v err %v", ss[i][0], 0, err.Error())
			return false
		}
		v.FormulaID = int32(intv)
		id = intv
		// FusionType
		intv, err = strconv.Atoi(ss[i][1])
		if err != nil {
			log.Printf("table Fusion convert column FusionType value %v with row %v err %v", ss[i][1], 1, err.Error())
			return false
		}
		v.FusionType = int32(intv)
		// ResultDropID
		intv, err = strconv.Atoi(ss[i][2])
		if err != nil {
			log.Printf("table Fusion convert column ResultDropID value %v with row %v err %v", ss[i][2], 2, err.Error())
			return false
		}
		v.ResultDropID = int32(intv)
		// MainCardID
		intv, err = strconv.Atoi(ss[i][3])
		if err != nil {
			log.Printf("table Fusion convert column MainCardID value %v with row %v err %v", ss[i][3], 3, err.Error())
			return false
		}
		v.MainCardID = int32(intv)
		// MainCardLevelCond
		intv, err = strconv.Atoi(ss[i][4])
		if err != nil {
			log.Printf("table Fusion convert column MainCardLevelCond value %v with row %v err %v", ss[i][4], 4, err.Error())
			return false
		}
		v.MainCardLevelCond = int32(intv)
		// ResCondtion
		v.ResCondtion = ss[i][5]
		// Cost1IDCond
		intv, err = strconv.Atoi(ss[i][6])
		if err != nil {
			log.Printf("table Fusion convert column Cost1IDCond value %v with row %v err %v", ss[i][6], 6, err.Error())
			return false
		}
		v.Cost1IDCond = int32(intv)
		// Cost1CampCond
		intv, err = strconv.Atoi(ss[i][7])
		if err != nil {
			log.Printf("table Fusion convert column Cost1CampCond value %v with row %v err %v", ss[i][7], 7, err.Error())
			return false
		}
		v.Cost1CampCond = int32(intv)
		// Cost1TypeCond
		intv, err = strconv.Atoi(ss[i][8])
		if err != nil {
			log.Printf("table Fusion convert column Cost1TypeCond value %v with row %v err %v", ss[i][8], 8, err.Error())
			return false
		}
		v.Cost1TypeCond = int32(intv)
		// Cost1StarCond
		intv, err = strconv.Atoi(ss[i][9])
		if err != nil {
			log.Printf("table Fusion convert column Cost1StarCond value %v with row %v err %v", ss[i][9], 9, err.Error())
			return false
		}
		v.Cost1StarCond = int32(intv)
		// Cost1NumCond
		intv, err = strconv.Atoi(ss[i][10])
		if err != nil {
			log.Printf("table Fusion convert column Cost1NumCond value %v with row %v err %v", ss[i][10], 10, err.Error())
			return false
		}
		v.Cost1NumCond = int32(intv)
		// Cost2IDCond
		intv, err = strconv.Atoi(ss[i][11])
		if err != nil {
			log.Printf("table Fusion convert column Cost2IDCond value %v with row %v err %v", ss[i][11], 11, err.Error())
			return false
		}
		v.Cost2IDCond = int32(intv)
		// Cost2CampCond
		intv, err = strconv.Atoi(ss[i][12])
		if err != nil {
			log.Printf("table Fusion convert column Cost2CampCond value %v with row %v err %v", ss[i][12], 12, err.Error())
			return false
		}
		v.Cost2CampCond = int32(intv)
		// Cost2TypeCond
		intv, err = strconv.Atoi(ss[i][13])
		if err != nil {
			log.Printf("table Fusion convert column Cost2TypeCond value %v with row %v err %v", ss[i][13], 13, err.Error())
			return false
		}
		v.Cost2TypeCond = int32(intv)
		// Cost2StarCond
		intv, err = strconv.Atoi(ss[i][14])
		if err != nil {
			log.Printf("table Fusion convert column Cost2StarCond value %v with row %v err %v", ss[i][14], 14, err.Error())
			return false
		}
		v.Cost2StarCond = int32(intv)
		// Cost2NumCond
		intv, err = strconv.Atoi(ss[i][15])
		if err != nil {
			log.Printf("table Fusion convert column Cost2NumCond value %v with row %v err %v", ss[i][15], 15, err.Error())
			return false
		}
		v.Cost2NumCond = int32(intv)
		// Cost3IDCond
		intv, err = strconv.Atoi(ss[i][16])
		if err != nil {
			log.Printf("table Fusion convert column Cost3IDCond value %v with row %v err %v", ss[i][16], 16, err.Error())
			return false
		}
		v.Cost3IDCond = int32(intv)
		// Cost3CampCond
		intv, err = strconv.Atoi(ss[i][17])
		if err != nil {
			log.Printf("table Fusion convert column Cost3CampCond value %v with row %v err %v", ss[i][17], 17, err.Error())
			return false
		}
		v.Cost3CampCond = int32(intv)
		// Cost3TypeCond
		intv, err = strconv.Atoi(ss[i][18])
		if err != nil {
			log.Printf("table Fusion convert column Cost3TypeCond value %v with row %v err %v", ss[i][18], 18, err.Error())
			return false
		}
		v.Cost3TypeCond = int32(intv)
		// Cost3StarCond
		intv, err = strconv.Atoi(ss[i][19])
		if err != nil {
			log.Printf("table Fusion convert column Cost3StarCond value %v with row %v err %v", ss[i][19], 19, err.Error())
			return false
		}
		v.Cost3StarCond = int32(intv)
		// Cost3NumCond
		intv, err = strconv.Atoi(ss[i][20])
		if err != nil {
			log.Printf("table Fusion convert column Cost3NumCond value %v with row %v err %v", ss[i][20], 20, err.Error())
			return false
		}
		v.Cost3NumCond = int32(intv)
		if id <= 0 {
			continue
		}
		this.id2items[int32(id)] = &v
		this.items_array = append(this.items_array, &v)
   	}
	return true
}

func (this *FusionMgr) Get(id int32) *Fusion {
	return this.id2items[id]
}

func (this *FusionMgr) GetByIndex(idx int32) *Fusion {
	if int(idx) >= len(this.items_array) {
		return nil
	}
	return this.items_array[idx]
}

func (this *FusionMgr) GetNum() int32 {
	return int32(len(this.items_array))
}

