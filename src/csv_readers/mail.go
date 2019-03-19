package csv_readers

import (
	"encoding/csv"
	"io/ioutil"
	"log"
	"strconv"
	"strings"
)

type Mail struct {
	MailID int32
	MailType int32
	MailTitleID int32
	MailContentID int32
}

type MailMgr struct {
	id2items map[int32]*Mail
	items_array []*Mail
}

func (this *MailMgr) Read(file_path_name string) bool {
	if file_path_name == "" {
		file_path_name = "../game_csv/mail.csv"
	}
	cs, err := ioutil.ReadFile(file_path_name)
	if err != nil {
		log.Printf("MailMgr.Read err: %v", err.Error())
		return false
	}

 	r := csv.NewReader(strings.NewReader(string(cs)))
	ss, _ := r.ReadAll()
   	sz := len(ss)
	this.id2items = make(map[int32]*Mail)
    for i := int32(1); i < int32(sz); i++ {
		//if i < 5 {
		//	continue
		//}
		var v Mail
		var intv, id int
		// MailID
		intv, err = strconv.Atoi(ss[i][0])
		if err != nil {
			log.Printf("table Mail convert column MailID value %v with row %v err %v", ss[i][0], 0, err.Error())
			return false
		}
		v.MailID = int32(intv)
		id = intv
		// MailType
		intv, err = strconv.Atoi(ss[i][1])
		if err != nil {
			log.Printf("table Mail convert column MailType value %v with row %v err %v", ss[i][1], 1, err.Error())
			return false
		}
		v.MailType = int32(intv)
		// MailTitleID
		intv, err = strconv.Atoi(ss[i][2])
		if err != nil {
			log.Printf("table Mail convert column MailTitleID value %v with row %v err %v", ss[i][2], 2, err.Error())
			return false
		}
		v.MailTitleID = int32(intv)
		// MailContentID
		intv, err = strconv.Atoi(ss[i][3])
		if err != nil {
			log.Printf("table Mail convert column MailContentID value %v with row %v err %v", ss[i][3], 3, err.Error())
			return false
		}
		v.MailContentID = int32(intv)
		if id <= 0 {
			continue
		}
		this.id2items[int32(id)] = &v
		this.items_array = append(this.items_array, &v)
   	}
	return true
}

func (this *MailMgr) Get(id int32) *Mail {
	return this.id2items[id]
}

func (this *MailMgr) GetByIndex(idx int32) *Mail {
	if int(idx) >= len(this.items_array) {
		return nil
	}
	return this.items_array[idx]
}

func (this *MailMgr) GetNum() int32 {
	return int32(len(this.items_array))
}

