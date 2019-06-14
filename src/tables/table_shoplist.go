package tables

import (
	"encoding/xml"
	"io/ioutil"
	"mm_server/libs/log"
	"mm_server/src/server_config"
)

type XmlShopTypeItem struct {
	Id              int32  `xml:"ID,attr"`
	AutoRefreshTime string `xml:"AutoRefreshTime,attr"`
	RefreshDays     int32  `xml:"RefreshDays,attr"`
}

type XmlShopTypeConfig struct {
	Items []*XmlShopTypeItem `xml:"item"`
}

type ShopTypeTableManager struct {
	items map[int32]*XmlShopTypeItem
}

func (this *ShopTypeTableManager) Init(table_file string) bool {
	if table_file == "" {
		table_file = "ShopList.xml"
	}
	file_path := server_config.GetGameDataPathFile(table_file)
	data, err := ioutil.ReadFile(file_path)
	if nil != err {
		log.Error("ShopTypeTableManager Load read file err[%s] !", err.Error())
		return false
	}

	tmp_cfg := &XmlShopTypeConfig{}
	err = xml.Unmarshal(data, tmp_cfg)
	if nil != err {
		log.Error("ShopTypeTableManager Load xml Unmarshal failed error [%s] !", err.Error())
		return false
	}

	this.items = make(map[int32]*XmlShopTypeItem)
	tmp_len := int32(len(tmp_cfg.Items))
	for i := int32(0); i < tmp_len; i++ {
		c := tmp_cfg.Items[i]
		this.items[c.Id] = c
	}

	log.Info("ShopList table load items count(%v)", tmp_len)

	return true
}

func (this *ShopTypeTableManager) GetShopType(shop_id int32) *XmlShopTypeItem {
	return this.items[shop_id]
}
