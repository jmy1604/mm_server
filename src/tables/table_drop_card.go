package tables

import (
	"encoding/xml"
	"io/ioutil"
	"mm_server/libs/log"
	"mm_server/src/server_config"
)

type XmlDropCardItem struct {
	GroupId  int32 `xml:"GroupId,attr"`
	DropType int32 `xml:"Type,attr"`
	Weight   int32 `xml:"Weight,attr"`
	Min      int32 `xml:"min,attr"`
	Max      int32 `xml:"max,attr"`
	DropId   int32 `xml:"ID,attr"`
}

type XmlDropCardConfig struct {
	Items []*XmlDropCardItem `xml:"item"`
}

type DropCardTypeLib struct {
	DropLibType int32
	TotalCount  int32
	TotalWeight int32
	DropItems   []*XmlDropCardItem
}

type DropCardTableManager struct {
	Map map[int32]*DropCardTypeLib
}

func (this *DropCardTableManager) Init(table_file string) bool {
	if table_file == "" {
		table_file = "DropCard.xml"
	}
	file_path := server_config.GetGameDataPathFile(table_file)
	data, err := ioutil.ReadFile(file_path)
	if nil != err {
		log.Error("DropCardTableManager load read file failed[%s] !", err.Error())
		return false
	}

	tmp_cfg := &XmlDropCardConfig{}
	err = xml.Unmarshal(data, tmp_cfg)
	if nil != err {
		log.Error("DropCardTableManager load xml unmarshal failed [%s]!", err.Error())
		return false
	}

	this.Map = make(map[int32]*DropCardTypeLib)

	tmp_len := len(tmp_cfg.Items)
	var tmp_item *XmlDropCardItem
	var tmp_lib *DropCardTypeLib
	for i := 0; i < tmp_len; i++ {
		tmp_item = tmp_cfg.Items[i]
		if nil == tmp_item {
			continue
		}

		tmp_lib = this.Map[tmp_item.GroupId]
		if nil == tmp_lib {
			tmp_lib := &DropCardTypeLib{}
			tmp_lib.DropLibType = tmp_item.GroupId //tmp_item.DropType
			tmp_lib.TotalCount = 1
			tmp_lib.TotalWeight = tmp_item.Weight
			this.Map[tmp_item.GroupId] = tmp_lib
		} else {
			tmp_lib.TotalCount++
			tmp_lib.TotalWeight += tmp_item.Weight
		}
	}

	for i := 0; i < tmp_len; i++ {
		tmp_item = tmp_cfg.Items[i]
		if nil == tmp_item {
			continue
		}

		tmp_lib := this.Map[tmp_item.GroupId]
		if nil == tmp_lib {
			continue
		}

		if nil == tmp_lib.DropItems {
			//log.Info("类型%d的随机总权重%d", tmp_lib.DropLibType, tmp_lib.TotalWeight)
			tmp_lib.DropItems = make([]*XmlDropCardItem, 0, tmp_lib.TotalCount)
		}

		tmp_lib.DropItems = append(tmp_lib.DropItems, tmp_item)

	}

	return true
}
