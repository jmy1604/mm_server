package tables

import (
	"encoding/xml"
	"io/ioutil"
	"mm_server/libs/log"
	"mm_server/src/server_config"
)

const (
	BUILDING_DIR_BIG_X_DIR = 0 // 水平朝向
	BUILDING_DIR_BIG_Y_DIR = 1 // 垂直朝向
)

type XmlBuildingItem struct {
	Id            int32  `xml:"Id,attr"`
	MaxLevel      int32  `xml:"MaxLevel,attr"`
	Type          int32  `xml:"Type,attr"`
	Tag           int32  `xml:"Tag,attr"`
	Rarity        int32  `xml:"Rarity,attr"`
	UnlockType    int32  `xml:"UnlockType,attr"`
	UnlockLevel   int32  `xml:"UnlockLevel,attr"`
	UnlockCostStr string `xml:"UnlockCost,attr"`
	UnlockCosts   []int32
	BuildTime     int32  `xml:"BuildTime,attr"`
	Charm         int32  `xml:"Charm,attr"`
	Geography     int32  `xml:"Geography,attr"`
	SaleCoin      int32  `xml:"SaleCoin,attr"`
	MapSizeStr    string `xml:"MapSize,attr"`
	MapSizes      []int32
	IfFunction    int32 `xml:"Function,attr"`
	SuitId        int32 `xml:"SuitId,attr"`
}

type XmlBuildingConfig struct {
	Items []XmlBuildingItem `xml:"item"`
}

type SuitItems struct {
	Items []int32
	Num   int32
}

type BuildingTableMgr struct {
	Map   map[int32]*XmlBuildingItem
	Suits map[int32]*SuitItems
}

func (this *BuildingTableMgr) Init(table_file string) bool {
	if table_file == "" {
		table_file = "Building.xml"
	}
	table_path := server_config.GetGameDataPathFile(table_file)
	data, err := ioutil.ReadFile(table_path)
	if nil != err {
		log.Error("BuildingTableMgr Init read file err[%s] !", err.Error())
		return false
	}

	tmp_cfg := &XmlBuildingConfig{}
	err = xml.Unmarshal(data, tmp_cfg)
	if nil != err {
		log.Error("BuildingTableMgr Init xml Unmarshal failed error [%s] !", err.Error())
		return false
	}

	suits := make(map[int32]int32)
	this.Map = make(map[int32]*XmlBuildingItem)
	tmp_len := int32(len(tmp_cfg.Items))
	var tmp_item *XmlBuildingItem
	for idx := int32(0); idx < tmp_len; idx++ {
		tmp_item = &tmp_cfg.Items[idx]

		tmp_item.UnlockCosts = parse_xml_str_arr(tmp_item.UnlockCostStr, ",")
		tmp_item.MapSizes = parse_xml_str_arr(tmp_item.MapSizeStr, ",")
		if len(tmp_item.MapSizes) < 2 {
			log.Error("BuildingTableMgr Init [%d] mapsize[%s] error", tmp_item.Id, tmp_item.MapSizeStr)
			return false
		}

		this.Map[tmp_item.Id] = tmp_item
		if tmp_item.SuitId > 0 {
			n := suits[tmp_item.SuitId]
			suits[tmp_item.SuitId] = n + 1
		}
	}

	this.Suits = make(map[int32]*SuitItems)
	for idx := int32(0); idx < tmp_len; idx++ {
		tmp_item = &tmp_cfg.Items[idx]
		if tmp_item.SuitId > 0 {
			ss := this.Suits[tmp_item.SuitId]
			if ss == nil {
				ss = &SuitItems{}
				ss.Items = make([]int32, suits[tmp_item.SuitId])
				this.Suits[tmp_item.SuitId] = ss
			}
			ss.Items[ss.Num] = tmp_item.Id
			ss.Num += 1
		}
	}

	for k, v := range this.Suits {
		log.Debug("@@@@@ Suits[%v]: %v", k, v)
	}

	return true
}
