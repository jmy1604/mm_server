package csv_readers

import (
	"encoding/csv"
	"io/ioutil"
	"log"
	"strconv"
	"strings"
)

type Sign struct {
	TotalIndex int32
	Group int32
	GroupIndex int32
	Reward string
}

type SignMgr struct {
	id2items map[int32]*Sign
	items_array []*Sign
}

func (this *SignMgr) Read(file_path_name string) bool {
	if file_path_name == "" {
		file_path_name = "../game_csv/sign.csv"
	}
	cs, err := ioutil.ReadFile(file_path_name)
	if err != nil {
		log.Printf("SignMgr.Read err: %v", err.Error())
		return false
	}

 	r := csv.NewReader(strings.NewReader(string(cs)))
	ss, _ := r.ReadAll()
   	sz := len(ss)
	this.id2items = make(map[int32]*Sign)
    for i := int32(1); i < int32(sz); i++ {
		//if i < 5 {
		//	continue
		//}
		var v Sign
		var intv, id int
		// TotalIndex
		intv, err = strconv.Atoi(ss[i][0])
		if err != nil {
			log.Printf("table Sign convert column TotalIndex value %v with row %v err %v", ss[i][0], 0, err.Error())
			return false
		}
		v.TotalIndex = int32(intv)
		id = intv
		// Group
		intv, err = strconv.Atoi(ss[i][1])
		if err != nil {
			log.Printf("table Sign convert column Group value %v with row %v err %v", ss[i][1], 1, err.Error())
			return false
		}
		v.Group = int32(intv)
		// GroupIndex
		intv, err = strconv.Atoi(ss[i][2])
		if err != nil {
			log.Printf("table Sign convert column GroupIndex value %v with row %v err %v", ss[i][2], 2, err.Error())
			return false
		}
		v.GroupIndex = int32(intv)
		// Reward
		v.Reward = ss[i][3]
		if id <= 0 {
			continue
		}
		this.id2items[int32(id)] = &v
		this.items_array = append(this.items_array, &v)
   	}
	return true
}

func (this *SignMgr) Get(id int32) *Sign {
	return this.id2items[id]
}

func (this *SignMgr) GetByIndex(idx int32) *Sign {
	if int(idx) >= len(this.items_array) {
		return nil
	}
	return this.items_array[idx]
}

func (this *SignMgr) GetNum() int32 {
	return int32(len(this.items_array))
}

