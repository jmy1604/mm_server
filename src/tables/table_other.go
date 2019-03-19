package tables

import (
	"encoding/xml"
	"io/ioutil"
	"mm_server/libs/log"
	"mm_server/src/server_config"
)

type XmlOtherItem struct {
	Id     int32 `xml:"Id,attr"`
	Type   int32 `xml:"Type:attr"`
	Cost   int32 `xml:"Cost:attr"`
	Money  int32 `xml:"Money,attr"`
	Cost2  int32 `xml:"Cost2,attr"`
	Money2 int32 `xml:"Money2,attr"`
}

type XmlOtherConfig struct {
	Items []XmlOtherItem `xml:"item"`
}

type OtherTableManager struct {
	Map   map[int32]*XmlOtherItem
	Array []*XmlOtherItem
}

func (this *OtherTableManager) Init(table_file string) bool {
	if table_file == "" {
		table_file = "Other.xml"
	}
	file_path := server_config.GetGameDataPathFile(table_file)
	data, err := ioutil.ReadFile(file_path)
	if nil != err {
		log.Error("OtherTableManager Load read file err[%s] !", err.Error())
		return false
	}

	tmp_cfg := &XmlOtherConfig{}
	err = xml.Unmarshal(data, tmp_cfg)
	if nil != err {
		log.Error("OtherTableManager Load xml Unmarshal failed error [%s] !", err.Error())
		return false
	}

	if this.Map == nil {
		this.Map = make(map[int32]*XmlOtherItem)
	}
	if this.Array == nil {
		this.Array = make([]*XmlOtherItem, 0)
	}

	tmp_len := int32(len(tmp_cfg.Items))
	var tmp_item *XmlOtherItem
	for idx := int32(0); idx < tmp_len; idx++ {
		tmp_item = &tmp_cfg.Items[idx]
		this.Map[tmp_item.Id] = tmp_item
		this.Array = append(this.Array, tmp_item)
	}

	return true
}
