package csv_readers

import (
	"encoding/csv"
	"io/ioutil"
	"log"
	"strconv"
	"strings"
)

type Rankup struct {
	Rank int32
	Type1RankUpRes string
	Type2RankUpRes string
	Type3RankUpRes string
	Type1DecomposeRes string
	Type2DecomposeRes string
	Type3DecomposeRes string
}

type RankupMgr struct {
	id2items map[int32]*Rankup
	items_array []*Rankup
}

func (this *RankupMgr) Read(file_path_name string) bool {
	if file_path_name == "" {
		file_path_name = "../game_csv/rankup.csv"
	}
	cs, err := ioutil.ReadFile(file_path_name)
	if err != nil {
		log.Printf("RankupMgr.Read err: %v", err.Error())
		return false
	}

 	r := csv.NewReader(strings.NewReader(string(cs)))
	ss, _ := r.ReadAll()
   	sz := len(ss)
	this.id2items = make(map[int32]*Rankup)
    for i := int32(1); i < int32(sz); i++ {
		//if i < 5 {
		//	continue
		//}
		var v Rankup
		var intv, id int
		// Rank
		intv, err = strconv.Atoi(ss[i][0])
		if err != nil {
			log.Printf("table Rankup convert column Rank value %v with row %v err %v", ss[i][0], 0, err.Error())
			return false
		}
		v.Rank = int32(intv)
		id = intv
		// Type1RankUpRes
		v.Type1RankUpRes = ss[i][1]
		// Type2RankUpRes
		v.Type2RankUpRes = ss[i][2]
		// Type3RankUpRes
		v.Type3RankUpRes = ss[i][3]
		// Type1DecomposeRes
		v.Type1DecomposeRes = ss[i][4]
		// Type2DecomposeRes
		v.Type2DecomposeRes = ss[i][5]
		// Type3DecomposeRes
		v.Type3DecomposeRes = ss[i][6]
		if id <= 0 {
			continue
		}
		this.id2items[int32(id)] = &v
		this.items_array = append(this.items_array, &v)
   	}
	return true
}

func (this *RankupMgr) Get(id int32) *Rankup {
	return this.id2items[id]
}

func (this *RankupMgr) GetByIndex(idx int32) *Rankup {
	if int(idx) >= len(this.items_array) {
		return nil
	}
	return this.items_array[idx]
}

func (this *RankupMgr) GetNum() int32 {
	return int32(len(this.items_array))
}

