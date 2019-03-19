package csv_readers

import (
	"encoding/csv"
	"io/ioutil"
	"log"
	"strconv"
	"strings"
)

type Arenadivision struct {
	Division int32
	DivisionScoreMin int32
	DivisionScoreMax int32
	WinScore int32
	WinningStreakScoreBonus int32
	LoseScore int32
	NewSeasonScore int32
	RewardList string
}

type ArenadivisionMgr struct {
	id2items map[int32]*Arenadivision
	items_array []*Arenadivision
}

func (this *ArenadivisionMgr) Read(file_path_name string) bool {
	if file_path_name == "" {
		file_path_name = "../game_csv/arenadivision.csv"
	}
	cs, err := ioutil.ReadFile(file_path_name)
	if err != nil {
		log.Printf("ArenadivisionMgr.Read err: %v", err.Error())
		return false
	}

 	r := csv.NewReader(strings.NewReader(string(cs)))
	ss, _ := r.ReadAll()
   	sz := len(ss)
	this.id2items = make(map[int32]*Arenadivision)
    for i := int32(1); i < int32(sz); i++ {
		//if i < 5 {
		//	continue
		//}
		var v Arenadivision
		var intv, id int
		// Division
		intv, err = strconv.Atoi(ss[i][0])
		if err != nil {
			log.Printf("table Arenadivision convert column Division value %v with row %v err %v", ss[i][0], 0, err.Error())
			return false
		}
		v.Division = int32(intv)
		id = intv
		// DivisionScoreMin
		intv, err = strconv.Atoi(ss[i][1])
		if err != nil {
			log.Printf("table Arenadivision convert column DivisionScoreMin value %v with row %v err %v", ss[i][1], 1, err.Error())
			return false
		}
		v.DivisionScoreMin = int32(intv)
		// DivisionScoreMax
		intv, err = strconv.Atoi(ss[i][2])
		if err != nil {
			log.Printf("table Arenadivision convert column DivisionScoreMax value %v with row %v err %v", ss[i][2], 2, err.Error())
			return false
		}
		v.DivisionScoreMax = int32(intv)
		// WinScore
		intv, err = strconv.Atoi(ss[i][3])
		if err != nil {
			log.Printf("table Arenadivision convert column WinScore value %v with row %v err %v", ss[i][3], 3, err.Error())
			return false
		}
		v.WinScore = int32(intv)
		// WinningStreakScoreBonus
		intv, err = strconv.Atoi(ss[i][4])
		if err != nil {
			log.Printf("table Arenadivision convert column WinningStreakScoreBonus value %v with row %v err %v", ss[i][4], 4, err.Error())
			return false
		}
		v.WinningStreakScoreBonus = int32(intv)
		// LoseScore
		intv, err = strconv.Atoi(ss[i][5])
		if err != nil {
			log.Printf("table Arenadivision convert column LoseScore value %v with row %v err %v", ss[i][5], 5, err.Error())
			return false
		}
		v.LoseScore = int32(intv)
		// NewSeasonScore
		intv, err = strconv.Atoi(ss[i][6])
		if err != nil {
			log.Printf("table Arenadivision convert column NewSeasonScore value %v with row %v err %v", ss[i][6], 6, err.Error())
			return false
		}
		v.NewSeasonScore = int32(intv)
		// RewardList
		v.RewardList = ss[i][7]
		if id <= 0 {
			continue
		}
		this.id2items[int32(id)] = &v
		this.items_array = append(this.items_array, &v)
   	}
	return true
}

func (this *ArenadivisionMgr) Get(id int32) *Arenadivision {
	return this.id2items[id]
}

func (this *ArenadivisionMgr) GetByIndex(idx int32) *Arenadivision {
	if int(idx) >= len(this.items_array) {
		return nil
	}
	return this.items_array[idx]
}

func (this *ArenadivisionMgr) GetNum() int32 {
	return int32(len(this.items_array))
}

