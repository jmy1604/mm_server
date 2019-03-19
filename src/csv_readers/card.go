package csv_readers

import (
	"encoding/csv"
	"io/ioutil"
	"log"
	"strconv"
	"strings"
)

type Card struct {
	ClientID int32
	ID int32
	Rank int32
	MaxLevel int32
	MaxRank int32
	Rarity int32
	Type int32
	Camp int32
	Label string
	BaseHP int32
	BaseAttack int32
	BaseDefence int32
	GrowthHP int32
	GrowthAttack int32
	GrowthDefence int32
	NormalSkillID int32
	SuperSkillID int32
	PassiveSkillID string
	DecomposeRes string
	BattlePower int32
	BattlePowerGrowth int32
	HeadItem int32
	BagFullChangeItem string
	ConvertID1 int32
	ConvertID2 int32
	ConvertItem string
}

type CardMgr struct {
	id2items map[int32]*Card
	items_array []*Card
}

func (this *CardMgr) Read(file_path_name string) bool {
	if file_path_name == "" {
		file_path_name = "../game_csv/card.csv"
	}
	cs, err := ioutil.ReadFile(file_path_name)
	if err != nil {
		log.Printf("CardMgr.Read err: %v", err.Error())
		return false
	}

 	r := csv.NewReader(strings.NewReader(string(cs)))
	ss, _ := r.ReadAll()
   	sz := len(ss)
	this.id2items = make(map[int32]*Card)
    for i := int32(1); i < int32(sz); i++ {
		//if i < 5 {
		//	continue
		//}
		var v Card
		var intv, id int
		// ClientID
		intv, err = strconv.Atoi(ss[i][0])
		if err != nil {
			log.Printf("table Card convert column ClientID value %v with row %v err %v", ss[i][0], 0, err.Error())
			return false
		}
		v.ClientID = int32(intv)
		id = intv
		// ID
		intv, err = strconv.Atoi(ss[i][1])
		if err != nil {
			log.Printf("table Card convert column ID value %v with row %v err %v", ss[i][1], 1, err.Error())
			return false
		}
		v.ID = int32(intv)
		// Rank
		intv, err = strconv.Atoi(ss[i][2])
		if err != nil {
			log.Printf("table Card convert column Rank value %v with row %v err %v", ss[i][2], 2, err.Error())
			return false
		}
		v.Rank = int32(intv)
		// MaxLevel
		intv, err = strconv.Atoi(ss[i][3])
		if err != nil {
			log.Printf("table Card convert column MaxLevel value %v with row %v err %v", ss[i][3], 3, err.Error())
			return false
		}
		v.MaxLevel = int32(intv)
		// MaxRank
		intv, err = strconv.Atoi(ss[i][4])
		if err != nil {
			log.Printf("table Card convert column MaxRank value %v with row %v err %v", ss[i][4], 4, err.Error())
			return false
		}
		v.MaxRank = int32(intv)
		// Rarity
		intv, err = strconv.Atoi(ss[i][5])
		if err != nil {
			log.Printf("table Card convert column Rarity value %v with row %v err %v", ss[i][5], 5, err.Error())
			return false
		}
		v.Rarity = int32(intv)
		// Type
		intv, err = strconv.Atoi(ss[i][6])
		if err != nil {
			log.Printf("table Card convert column Type value %v with row %v err %v", ss[i][6], 6, err.Error())
			return false
		}
		v.Type = int32(intv)
		// Camp
		intv, err = strconv.Atoi(ss[i][7])
		if err != nil {
			log.Printf("table Card convert column Camp value %v with row %v err %v", ss[i][7], 7, err.Error())
			return false
		}
		v.Camp = int32(intv)
		// Label
		v.Label = ss[i][8]
		// BaseHP
		intv, err = strconv.Atoi(ss[i][9])
		if err != nil {
			log.Printf("table Card convert column BaseHP value %v with row %v err %v", ss[i][9], 9, err.Error())
			return false
		}
		v.BaseHP = int32(intv)
		// BaseAttack
		intv, err = strconv.Atoi(ss[i][10])
		if err != nil {
			log.Printf("table Card convert column BaseAttack value %v with row %v err %v", ss[i][10], 10, err.Error())
			return false
		}
		v.BaseAttack = int32(intv)
		// BaseDefence
		intv, err = strconv.Atoi(ss[i][11])
		if err != nil {
			log.Printf("table Card convert column BaseDefence value %v with row %v err %v", ss[i][11], 11, err.Error())
			return false
		}
		v.BaseDefence = int32(intv)
		// GrowthHP
		intv, err = strconv.Atoi(ss[i][12])
		if err != nil {
			log.Printf("table Card convert column GrowthHP value %v with row %v err %v", ss[i][12], 12, err.Error())
			return false
		}
		v.GrowthHP = int32(intv)
		// GrowthAttack
		intv, err = strconv.Atoi(ss[i][13])
		if err != nil {
			log.Printf("table Card convert column GrowthAttack value %v with row %v err %v", ss[i][13], 13, err.Error())
			return false
		}
		v.GrowthAttack = int32(intv)
		// GrowthDefence
		intv, err = strconv.Atoi(ss[i][14])
		if err != nil {
			log.Printf("table Card convert column GrowthDefence value %v with row %v err %v", ss[i][14], 14, err.Error())
			return false
		}
		v.GrowthDefence = int32(intv)
		// NormalSkillID
		intv, err = strconv.Atoi(ss[i][15])
		if err != nil {
			log.Printf("table Card convert column NormalSkillID value %v with row %v err %v", ss[i][15], 15, err.Error())
			return false
		}
		v.NormalSkillID = int32(intv)
		// SuperSkillID
		intv, err = strconv.Atoi(ss[i][16])
		if err != nil {
			log.Printf("table Card convert column SuperSkillID value %v with row %v err %v", ss[i][16], 16, err.Error())
			return false
		}
		v.SuperSkillID = int32(intv)
		// PassiveSkillID
		v.PassiveSkillID = ss[i][17]
		// DecomposeRes
		v.DecomposeRes = ss[i][18]
		// BattlePower
		intv, err = strconv.Atoi(ss[i][19])
		if err != nil {
			log.Printf("table Card convert column BattlePower value %v with row %v err %v", ss[i][19], 19, err.Error())
			return false
		}
		v.BattlePower = int32(intv)
		// BattlePowerGrowth
		intv, err = strconv.Atoi(ss[i][20])
		if err != nil {
			log.Printf("table Card convert column BattlePowerGrowth value %v with row %v err %v", ss[i][20], 20, err.Error())
			return false
		}
		v.BattlePowerGrowth = int32(intv)
		// HeadItem
		intv, err = strconv.Atoi(ss[i][21])
		if err != nil {
			log.Printf("table Card convert column HeadItem value %v with row %v err %v", ss[i][21], 21, err.Error())
			return false
		}
		v.HeadItem = int32(intv)
		// BagFullChangeItem
		v.BagFullChangeItem = ss[i][22]
		// ConvertID1
		intv, err = strconv.Atoi(ss[i][23])
		if err != nil {
			log.Printf("table Card convert column ConvertID1 value %v with row %v err %v", ss[i][23], 23, err.Error())
			return false
		}
		v.ConvertID1 = int32(intv)
		// ConvertID2
		intv, err = strconv.Atoi(ss[i][24])
		if err != nil {
			log.Printf("table Card convert column ConvertID2 value %v with row %v err %v", ss[i][24], 24, err.Error())
			return false
		}
		v.ConvertID2 = int32(intv)
		// ConvertItem
		v.ConvertItem = ss[i][25]
		if id <= 0 {
			continue
		}
		this.id2items[int32(id)] = &v
		this.items_array = append(this.items_array, &v)
   	}
	return true
}

func (this *CardMgr) Get(id int32) *Card {
	return this.id2items[id]
}

func (this *CardMgr) GetByIndex(idx int32) *Card {
	if int(idx) >= len(this.items_array) {
		return nil
	}
	return this.items_array[idx]
}

func (this *CardMgr) GetNum() int32 {
	return int32(len(this.items_array))
}

