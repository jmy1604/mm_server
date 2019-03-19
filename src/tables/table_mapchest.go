package tables

import (
	"encoding/xml"
	"io/ioutil"
	"math/rand"
	"mm_server/libs/log"
	"mm_server/src/server_config"
)

type XmlMapChestItem struct {
	Id         int32  `xml:"Id,attr"`
	Weight     int32  `xml:"Weight,attr"`
	RemoveStr  string `xml:"Remove,attr"`
	RemoveCost []int32
	Exp        int32  `xml:"Exp,attr"`
	FriPoint   int32  `xml:"FriPoint,attr"`
	LastSec    int32  `xml:"Time,attr"`
	RewardStr  string `xml:"Reward,attr"`
	Rewards    []int32
}

type XmlMapChestConfig struct {
	Items []XmlMapChestItem `xml:"item"`
}

type MapChestMgr struct {
	Map           map[int32]*XmlMapChestItem
	Array         []*XmlMapChestItem
	TotalCount    int32
	TotalWeight   int32
	MaxBoxLastSec int32
}

func (this *MapChestMgr) Init(table_file string) bool {
	if table_file == "" {
		table_file = "box.xml"
	}
	file_path := server_config.GetGameDataPathFile(table_file)
	data, err := ioutil.ReadFile(file_path)
	if nil != err {
		log.Error("MapChestMgr LoadMapChest read file err[%s] !", err.Error())
		return false
	}

	tmp_cfg := &XmlMapChestConfig{}
	err = xml.Unmarshal(data, tmp_cfg)
	if nil != err {
		log.Error("MapChestMgr LoadMapChest xml Unmarshal failed error [%s] !", err.Error())
		return false
	}

	this.Map = make(map[int32]*XmlMapChestItem)
	tmp_len := int32(len(tmp_cfg.Items))
	this.Array = make([]*XmlMapChestItem, 0, tmp_len)

	var tmp_item *XmlMapChestItem
	for idx := int32(0); idx < tmp_len; idx++ {
		tmp_item = &tmp_cfg.Items[idx]
		this.TotalWeight += tmp_item.Weight
		if tmp_item.LastSec > this.MaxBoxLastSec {
			this.MaxBoxLastSec = tmp_item.LastSec
		}
		tmp_item.RemoveCost = parse_xml_str_arr(tmp_item.RemoveStr, ",")
		if len(tmp_item.RemoveCost)%2 != 0 {
			log.Error("MapChestMgr LoadMapChest [%d] reomvecost [%s] error !", tmp_item.Id, tmp_item.RemoveStr)
			return false
		}

		tmp_item.Rewards = parse_xml_str_arr(tmp_item.RewardStr, ",")
		if len(tmp_item.Rewards)%2 != 0 {
			log.Error("MapChestMgr LoadMapChest [%d] RemoveStr [%s] error !", tmp_item.Id, tmp_item.Rewards)
			return false
		}

		this.Array = append(this.Array, tmp_item)
		this.Map[tmp_item.Id] = tmp_item
		this.TotalCount++
	}

	log.Info("宝箱最大持续时间 %d", this.MaxBoxLastSec)
	//log.Info("MapChestMgr total count %d info %v", this.TotalCount, this.Map)
	if this.TotalWeight < 0 {
		log.Error("MapChestMgr LoadMapChest xml unmarshal failed error [%s] !", err.Error())
		return false
	}

	return true
}

func (this *MapChestMgr) RandMapChest() *XmlMapChestItem {
	rand_val := rand.Int31n(this.TotalWeight)
	var tmp_item *XmlMapChestItem
	for idx := int32(0); idx < this.TotalCount; idx++ {
		tmp_item = this.Array[idx]
		if nil == tmp_item {
			continue
		}

		if rand_val < tmp_item.Weight {
			return tmp_item
		} else {
			rand_val -= tmp_item.Weight
		}
	}

	return nil
}
