package csv_readers

import (
	"encoding/csv"
	"io/ioutil"
	"log"
	"strconv"
	"strings"
)

type Extract struct {
	Id int32
	DropID string
	ResCondition1 string
	ResCondition2 string
	FreeExtractTime int32
	NeedBlank int32
	FirstDropID string
}

type ExtractMgr struct {
	id2items map[int32]*Extract
	items_array []*Extract
}

func (this *ExtractMgr) Read(file_path_name string) bool {
	if file_path_name == "" {
		file_path_name = "../game_csv/extract.csv"
	}
	cs, err := ioutil.ReadFile(file_path_name)
	if err != nil {
		log.Printf("ExtractMgr.Read err: %v", err.Error())
		return false
	}

 	r := csv.NewReader(strings.NewReader(string(cs)))
	ss, _ := r.ReadAll()
   	sz := len(ss)
	this.id2items = make(map[int32]*Extract)
    for i := int32(1); i < int32(sz); i++ {
		//if i < 5 {
		//	continue
		//}
		var v Extract
		var intv, id int
		// Id
		intv, err = strconv.Atoi(ss[i][0])
		if err != nil {
			log.Printf("table Extract convert column Id value %v with row %v err %v", ss[i][0], 0, err.Error())
			return false
		}
		v.Id = int32(intv)
		id = intv
		// DropID
		v.DropID = ss[i][1]
		// ResCondition1
		v.ResCondition1 = ss[i][2]
		// ResCondition2
		v.ResCondition2 = ss[i][3]
		// FreeExtractTime
		intv, err = strconv.Atoi(ss[i][4])
		if err != nil {
			log.Printf("table Extract convert column FreeExtractTime value %v with row %v err %v", ss[i][4], 4, err.Error())
			return false
		}
		v.FreeExtractTime = int32(intv)
		// NeedBlank
		intv, err = strconv.Atoi(ss[i][5])
		if err != nil {
			log.Printf("table Extract convert column NeedBlank value %v with row %v err %v", ss[i][5], 5, err.Error())
			return false
		}
		v.NeedBlank = int32(intv)
		// FirstDropID
		v.FirstDropID = ss[i][6]
		if id <= 0 {
			continue
		}
		this.id2items[int32(id)] = &v
		this.items_array = append(this.items_array, &v)
   	}
	return true
}

func (this *ExtractMgr) Get(id int32) *Extract {
	return this.id2items[id]
}

func (this *ExtractMgr) GetByIndex(idx int32) *Extract {
	if int(idx) >= len(this.items_array) {
		return nil
	}
	return this.items_array[idx]
}

func (this *ExtractMgr) GetNum() int32 {
	return int32(len(this.items_array))
}

