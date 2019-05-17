package tables

import (
	"encoding/xml"
	"io/ioutil"
	"mm_server/libs/log"
	"mm_server/src/server_config"
)

const (
	ITEM_CFG_TYPE_BUILDING = 0
	ITEM_CFG_TYPE_PROP     = 1
	ITEM_CFG_TYPE_THINGS   = 2
)

type XmlItemItem struct {
	CfgId        int32 `xml:"Id,attr"`
	ItemType     int32
	SaleCoin     int32 `xml:"SaleCoin,attr"`
	MaxNumber    int32 `xml:"MaxNumber,attr"`
	Cost         int32 `xml:"Cost,attr"`
	Diamond      int32 `xml:"Diamond,attr"`
	Type         int32 `xml:"Type,attr"`
	UseType      int32 `xml:"UseType,attr"`
	ConstantTime int32 `xml:"ConstantTime,attr"`
	ValidTime    int32 `xml:"ValidTime,attr"`
	Numbers      []int32
	NumberStr    string `xml:"Number,attr"`
	Gender       int32  `xml:"RoleType,attr"`
	EquipType    int32  `xml:"EquipType,attr"`
}

type XmlItemConfig struct {
	Items []XmlItemItem `xml:"item"`
}

type ItemTableMgr struct {
	Map   map[int32]*XmlItemItem
	Array []*XmlItemItem
}

func (this *ItemTableMgr) Init(prop_table, things_table string) bool {
	/*if !this.LoadProp(prop_table) {
		return false
	}*/

	if !this.LoadThings(things_table) {
		return false
	}

	return true
}

func (this *ItemTableMgr) LoadProp(prop_table string) bool {
	if prop_table == "" {
		prop_table = "Prop.xml"
	}
	file_path := server_config.GetGameDataPathFile(prop_table)
	data, err := ioutil.ReadFile(file_path)
	if nil != err {
		log.Error("ItemTableMgr LoadProp read file err[%s] !", err.Error())
		return false
	}

	tmp_cfg := &XmlItemConfig{}
	err = xml.Unmarshal(data, tmp_cfg)
	if nil != err {
		log.Error("ItemTableMgr LoadProp xml Unmarshal failed error [%s] !", err.Error())
		return false
	}

	if this.Map == nil {
		this.Map = make(map[int32]*XmlItemItem)
	}
	if this.Array == nil {
		this.Array = make([]*XmlItemItem, 0)
	}
	tmp_len := int32(len(tmp_cfg.Items))
	var tmp_item *XmlItemItem
	for idx := int32(0); idx < tmp_len; idx++ {
		tmp_item = &tmp_cfg.Items[idx]
		tmp_item.ItemType = ITEM_CFG_TYPE_PROP
		this.Map[tmp_item.CfgId] = tmp_item
		this.Array = append(this.Array, tmp_item)
	}

	return true
}

func (this *ItemTableMgr) LoadThings(things_table string) bool {
	if things_table == "" {
		things_table = "Things.xml"
	}
	file_path := server_config.GetGameDataPathFile(things_table)
	data, err := ioutil.ReadFile(file_path)
	if nil != err {
		log.Error("ItemTableMgr LoadThings read file err[%s] !", err.Error())
		return false
	}

	tmp_cfg := &XmlItemConfig{}
	err = xml.Unmarshal(data, tmp_cfg)
	if nil != err {
		log.Error("ItemTableMgr LoadThings xml Unmarshal failed error [%s] !", err.Error())
		return false
	}

	if this.Map == nil {
		this.Map = make(map[int32]*XmlItemItem)
	}
	if this.Array == nil {
		this.Array = make([]*XmlItemItem, 0)
	}
	tmp_len := int32(len(tmp_cfg.Items))
	var tmp_item *XmlItemItem
	for idx := int32(0); idx < tmp_len; idx++ {
		tmp_item = &tmp_cfg.Items[idx]
		tmp_item.ItemType = ITEM_CFG_TYPE_THINGS
		tmp_item.Numbers = parse_xml_str_arr(tmp_item.NumberStr, ",")
		this.Map[tmp_item.CfgId] = tmp_item
		this.Array = append(this.Array, tmp_item)
		//log.Info("item[%v] config: %v", tmp_item.CfgId, tmp_item)
	}

	return true
}

func (this *ItemTableMgr) Get(item_id int32) *XmlItemItem {
	return this.Map[item_id]
}
