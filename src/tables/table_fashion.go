package tables

import (
	"encoding/xml"
	"io/ioutil"
	"mm_server/libs/log"
	"mm_server/src/server_config"
)

type XmlFashionItem struct {
	Id        int32 `xml:"ID,attr"`
	Gender    int32 `xml:"RoleType,attr"`
	EquipType int32 `xml:"EquipType,attr"`
}

type XmlFashionConfig struct {
	Items []XmlFashionItem `xml:"item"`
}

type FashionTableMgr struct {
	Map   map[int32]*XmlFashionItem
	Array []*XmlFashionItem
}

func (this *FashionTableMgr) Init(table_file string) bool {
	if table_file == "" {
		table_file = "patrs.xml"
	}
	file_path := server_config.GetGameDataPathFile(table_file)
	data, err := ioutil.ReadFile(file_path)
	if nil != err {
		log.Error("FashionTableMgr read file err[%s] !", err.Error())
		return false
	}

	tmp_cfg := &XmlFashionConfig{}
	err = xml.Unmarshal(data, tmp_cfg)
	if nil != err {
		log.Error("FashionTableMgr xml Unmarshal failed error [%s] !", err.Error())
		return false
	}

	if this.Map == nil {
		this.Map = make(map[int32]*XmlFashionItem)
	}

	if this.Array == nil {
		this.Array = make([]*XmlFashionItem, 0)
	}

	tmp_len := int32(len(tmp_cfg.Items))
	var tmp_item *XmlFashionItem
	for idx := int32(0); idx < tmp_len; idx++ {
		tmp_item = &tmp_cfg.Items[idx]

		this.Map[tmp_item.Id] = tmp_item
		this.Array = append(this.Array, tmp_item)
	}

	return true
}

func (this *FashionTableMgr) Has(id int32) bool {
	if d := this.Map[id]; d == nil {
		return false
	}
	return true
}
