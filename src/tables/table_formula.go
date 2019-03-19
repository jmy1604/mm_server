package tables

import (
	"encoding/xml"
	"io/ioutil"
	"mm_server/libs/log"
	"mm_server/src/server_config"
)

const (
	ITEM_MATERIAL_ID_BOARD  = 29 // 木板
	ITEM_MATERIAL_ID_BRICK  = 30 // 砖块
	ITEM_MATERIAL_ID_IRON   = 31 // 生铁
	ITEM_MATERIAL_ID_GOLD   = 32 // 金砖
	ITEM_MATERIAL_ID_LEAVES = 33 // 叶子
	ITEM_MATERIAL_ID_CLOTH  = 34 // 布
	ITEM_MATERIAL_ID_RUBBER = 35 // 橡胶
	ITEM_MATERIAL_ID_PAINT  = 36 // 油漆
)

type XmlFormulaItem struct {
	Id            int32  `xml:"Id,attr"`
	BuildID       int32  `xml:"BuildingId,attr"`
	Rarity        int32  `xml:"Rarity,attr"`
	UnlockChapter int32  `xml:"UnlockChapter,attr"`
	Star          int32  `xml:"Star,attr"`
	Time          int32  `xml:"Time,attr"`
	Cost          int32  `xml:"Cost,attr"`
	Group         string `xml:"Group,attr"`
	CostItems     []*ItemInfo
	Exp           int32 `xml:"Exp,attr"`
}

type XmlFormulaConfig struct {
	Items []XmlFormulaItem `xml:"item"`
}

type FormulaTableMgr struct {
	Map   map[int32]*XmlFormulaItem
	Array []*XmlFormulaItem
}

func (this *FormulaTableMgr) Init(table_file string) bool {
	if table_file == "" {
		table_file = "formula.xml"
	}
	file_path := server_config.GetGameDataPathFile(table_file)
	data, err := ioutil.ReadFile(file_path)
	if nil != err {
		log.Error("TableFormulaMgr read file err[%s] !", err.Error())
		return false
	}

	tmp_cfg := &XmlFormulaConfig{}
	err = xml.Unmarshal(data, tmp_cfg)
	if nil != err {
		log.Error("TableFormulaMgr xml Unmarshal failed error [%s] !", err.Error())
		return false
	}

	if this.Map == nil {
		this.Map = make(map[int32]*XmlFormulaItem)
	}

	if this.Array == nil {
		this.Array = make([]*XmlFormulaItem, 0)
	}

	tmp_len := int32(len(tmp_cfg.Items))

	var tmp_item *XmlFormulaItem
	for idx := int32(0); idx < tmp_len; idx++ {
		tmp_item = &tmp_cfg.Items[idx]

		d := parse_xml_str_arr(tmp_item.Group, ",")
		if d == nil || len(d)%2 != 0 {
			log.Error("parse field Group[%v] with column[%v] failed", tmp_item.Group, idx)
			return false
		}

		tmp_item.CostItems = make([]*ItemInfo, 0)
		for i := 0; i < len(d)/2; i++ {
			info := &ItemInfo{}
			info.Id = d[2*i]
			info.Num = d[2*i+1]
			tmp_item.CostItems = append(tmp_item.CostItems, info)
		}

		this.Map[tmp_item.Id] = tmp_item
		this.Array = append(this.Array, tmp_item)
	}

	return true
}
