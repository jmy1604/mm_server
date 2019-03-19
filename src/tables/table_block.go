package tables

import (
	"encoding/xml"
	"io/ioutil"
	"math/rand"
	"mm_server/libs/log"
	"mm_server/src/server_config"
)

type XmlBlockItem struct {
	Id             int32  `xml:"Id,attr"`
	RemoveItemsStr string `xml:"Remove,attr"`
	RemoveItems    []int32
	Exp            string `xml:"Exp,attr"`
	DropIdStr      string `xml:"DropID,attr"`
	DropIds        []int32
	Weight         int32 `xml:"Weight,attr"`
}

type XmlBlockConfig struct {
	Items []XmlBlockItem `xml:"item"`
}

type BlockTableMgr struct {
	Map         map[int32]*XmlBlockItem
	Array       []*XmlBlockItem
	TotalCount  int32
	TotalWeight int32
}

func (this *BlockTableMgr) Init(table_file string) bool {
	if table_file == "" {
		table_file = "obstacle.xml"
	}
	table_path := server_config.GetGameDataPathFile(table_file)

	data, err := ioutil.ReadFile(table_path)
	if nil != err {
		log.Error("BlockTableMgr LoadBlock read file err[%s] !", err.Error())
		return false
	}

	tmp_cfg := &XmlBlockConfig{}
	err = xml.Unmarshal(data, tmp_cfg)
	if nil != err {
		log.Error("BlockTableMgr LoadBlock xml Unmarshal failed error [%s] !", err.Error())
		return false
	}

	this.Map = make(map[int32]*XmlBlockItem)
	tmp_len := int32(len(tmp_cfg.Items))
	this.Array = make([]*XmlBlockItem, 0, tmp_len)

	var tmp_item *XmlBlockItem
	for idx := int32(0); idx < tmp_len; idx++ {
		tmp_item = &tmp_cfg.Items[idx]
		this.TotalWeight += tmp_item.Weight

		tmp_item.DropIds = parse_xml_str_arr(tmp_item.DropIdStr, ",")
		if len(tmp_item.DropIds)%2 != 0 {
			log.Error("BlockTableMgr LoadBlock DropId[%s] error !", tmp_item.DropIdStr)
			return false
		}

		tmp_item.RemoveItems = parse_xml_str_arr(tmp_item.RemoveItemsStr, ",")
		if len(tmp_item.RemoveItems)%2 != 0 {
			log.Error("BlockTableMgr LoadBlock RemoveItems[%s] error !", tmp_item.RemoveItemsStr)
			return false
		}

		this.Array = append(this.Array, tmp_item)
		this.Map[tmp_item.Id] = tmp_item
		this.TotalCount++
	}

	//log.Info("CfgExpeditionMgr total count %d info %v", this.TotalCount, this.Map)
	if this.TotalWeight < 0 {
		log.Error("BlockTableMgr LoadBlock xml unmarshal failed error [%s] !", err.Error())
		return false
	}

	return true
}

func (this *BlockTableMgr) RandBlock() *XmlBlockItem {
	rand_val := rand.Int31n(this.TotalWeight)
	var tmp_item *XmlBlockItem
	for idx := int32(0); idx < this.TotalCount; idx++ {
		tmp_item = this.Array[idx]
		if nil == tmp_item {
			continue
		}

		if rand_val < tmp_item.Weight {
			return tmp_item
		} else {
			rand_val -= tmp_item.Weight
			continue
		}
	}

	return nil
}
