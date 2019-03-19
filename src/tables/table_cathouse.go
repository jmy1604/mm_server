package tables

import (
	"encoding/xml"
	"io/ioutil"
	"mm_server/libs/log"
	"mm_server/src/server_config"
)

type XmlCatHouseItem struct {
	//ID          int32 `xml:"Id,attr"`
	BuildingId  int32 `xml:"BuildingId,attr"`
	Level       int32 `xml:"Level,attr"`
	UnlockStar  int32 `xml:"UnlockStar,attr"`
	Cost        int32 `xml:"Cost,attr"`
	CatStorage  int32 `xml:"CatStorage,attr"`
	CoinStorage int32 `xml:"CoinStorage,attr"`
	Time        int32 `xml:"Time,attr"`
	Color       int32 `xml:"Color,attr"`
	SalePrice   int32 `xml:"SalePrice,attr"`
}

type XmlCatHouseConfig struct {
	Items []XmlCatHouseItem `xml:"item"`
}

type CatHouseTableMgr struct {
	Map   map[int32][]*XmlCatHouseItem
	Array []*XmlCatHouseItem
}

func (this *CatHouseTableMgr) Init(table_file string) bool {
	if table_file == "" {
		table_file = "Cathouse.xml"
	}
	table_path := server_config.GetGameDataPathFile(table_file)
	data, err := ioutil.ReadFile(table_path)
	if nil != err {
		log.Error("CatHouseTableMgr read file err[%s] !", err.Error())
		return false
	}

	tmp_cfg := &XmlCatHouseConfig{}
	err = xml.Unmarshal(data, tmp_cfg)
	if nil != err {
		log.Error("CatHouseTableMgr xml Unmarshal failed error [%s] !", err.Error())
		return false
	}

	if this.Map == nil {
		this.Map = make(map[int32][]*XmlCatHouseItem)
	}

	if this.Array == nil {
		this.Array = make([]*XmlCatHouseItem, 0)
	}

	tmp_len := int32(len(tmp_cfg.Items))
	var tmp_item *XmlCatHouseItem
	for idx := int32(0); idx < tmp_len; idx++ {
		tmp_item = &tmp_cfg.Items[idx]
		if this.Map[tmp_item.BuildingId] == nil {
			this.Map[tmp_item.BuildingId] = make([]*XmlCatHouseItem, 0)
		}
		this.Map[tmp_item.BuildingId] = append(this.Map[tmp_item.BuildingId], tmp_item)
		this.Array = append(this.Array, tmp_item)
	}

	return true
}
