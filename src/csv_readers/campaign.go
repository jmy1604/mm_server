package csv_readers

import (
	"encoding/csv"
	"io/ioutil"
	"log"
	"strconv"
	"strings"
)

type Campaign struct {
	ClientID int32
	CampaignID int32
	StageID int32
	UnlockMap int32
	Difficulty int32
	ChapterMap int32
	ChildMapID int32
	StaticRewardSec int32
	StaticRewardItem string
	RandomDropSec int32
	RandomDropIDList string
	CampainTask int32
}

type CampaignMgr struct {
	id2items map[int32]*Campaign
	items_array []*Campaign
}

func (this *CampaignMgr) Read(file_path_name string) bool {
	if file_path_name == "" {
		file_path_name = "../game_csv/campaign.csv"
	}
	cs, err := ioutil.ReadFile(file_path_name)
	if err != nil {
		log.Printf("CampaignMgr.Read err: %v", err.Error())
		return false
	}

 	r := csv.NewReader(strings.NewReader(string(cs)))
	ss, _ := r.ReadAll()
   	sz := len(ss)
	this.id2items = make(map[int32]*Campaign)
    for i := int32(1); i < int32(sz); i++ {
		//if i < 5 {
		//	continue
		//}
		var v Campaign
		var intv, id int
		// ClientID
		intv, err = strconv.Atoi(ss[i][0])
		if err != nil {
			log.Printf("table Campaign convert column ClientID value %v with row %v err %v", ss[i][0], 0, err.Error())
			return false
		}
		v.ClientID = int32(intv)
		id = intv
		// CampaignID
		intv, err = strconv.Atoi(ss[i][1])
		if err != nil {
			log.Printf("table Campaign convert column CampaignID value %v with row %v err %v", ss[i][1], 1, err.Error())
			return false
		}
		v.CampaignID = int32(intv)
		// StageID
		intv, err = strconv.Atoi(ss[i][2])
		if err != nil {
			log.Printf("table Campaign convert column StageID value %v with row %v err %v", ss[i][2], 2, err.Error())
			return false
		}
		v.StageID = int32(intv)
		// UnlockMap
		intv, err = strconv.Atoi(ss[i][3])
		if err != nil {
			log.Printf("table Campaign convert column UnlockMap value %v with row %v err %v", ss[i][3], 3, err.Error())
			return false
		}
		v.UnlockMap = int32(intv)
		// Difficulty
		intv, err = strconv.Atoi(ss[i][4])
		if err != nil {
			log.Printf("table Campaign convert column Difficulty value %v with row %v err %v", ss[i][4], 4, err.Error())
			return false
		}
		v.Difficulty = int32(intv)
		// ChapterMap
		intv, err = strconv.Atoi(ss[i][5])
		if err != nil {
			log.Printf("table Campaign convert column ChapterMap value %v with row %v err %v", ss[i][5], 5, err.Error())
			return false
		}
		v.ChapterMap = int32(intv)
		// ChildMapID
		intv, err = strconv.Atoi(ss[i][6])
		if err != nil {
			log.Printf("table Campaign convert column ChildMapID value %v with row %v err %v", ss[i][6], 6, err.Error())
			return false
		}
		v.ChildMapID = int32(intv)
		// StaticRewardSec
		intv, err = strconv.Atoi(ss[i][7])
		if err != nil {
			log.Printf("table Campaign convert column StaticRewardSec value %v with row %v err %v", ss[i][7], 7, err.Error())
			return false
		}
		v.StaticRewardSec = int32(intv)
		// StaticRewardItem
		v.StaticRewardItem = ss[i][8]
		// RandomDropSec
		intv, err = strconv.Atoi(ss[i][9])
		if err != nil {
			log.Printf("table Campaign convert column RandomDropSec value %v with row %v err %v", ss[i][9], 9, err.Error())
			return false
		}
		v.RandomDropSec = int32(intv)
		// RandomDropIDList
		v.RandomDropIDList = ss[i][10]
		// CampainTask
		intv, err = strconv.Atoi(ss[i][11])
		if err != nil {
			log.Printf("table Campaign convert column CampainTask value %v with row %v err %v", ss[i][11], 11, err.Error())
			return false
		}
		v.CampainTask = int32(intv)
		if id <= 0 {
			continue
		}
		this.id2items[int32(id)] = &v
		this.items_array = append(this.items_array, &v)
   	}
	return true
}

func (this *CampaignMgr) Get(id int32) *Campaign {
	return this.id2items[id]
}

func (this *CampaignMgr) GetByIndex(idx int32) *Campaign {
	if int(idx) >= len(this.items_array) {
		return nil
	}
	return this.items_array[idx]
}

func (this *CampaignMgr) GetNum() int32 {
	return int32(len(this.items_array))
}

