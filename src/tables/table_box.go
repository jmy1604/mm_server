package tables

import (
	"encoding/xml"
	"io/ioutil"
	"mm_server/libs/log"
	"mm_server/src/server_config"
)

type BoxItem struct {
	Type int32
	Id   int32
	Num  int32
}

type XmlBoxItem struct {
	Id    int32  `xml:"Id,attr"`
	Type1 string `xml:"Type1,attr"`
	Type2 string `xml:"Type2,attr"`
	Type3 string `xml:"Type3,attr"`
	Items []*BoxItem
}

type XmlBoxConfig struct {
	Items []*XmlBoxItem `xml:"item"`
}

type BoxTableManager struct {
	items map[int32]*XmlBoxItem
}

func (this *BoxTableManager) Init(table_file string) bool {
	if table_file == "" {
		table_file = "BoxConfig.xml"
	}
	table_path := server_config.GetGameDataPathFile(table_file)
	data, err := ioutil.ReadFile(table_path)
	if nil != err {
		log.Error("BoxTableManager Load read file err[%s] !", err.Error())
		return false
	}

	tmp_cfg := &XmlBoxConfig{}
	err = xml.Unmarshal(data, tmp_cfg)
	if nil != err {
		log.Error("BoxTableManager Load xml Unmarshal failed error [%s] !", err.Error())
		return false
	}

	tmp_len := len(tmp_cfg.Items)
	if tmp_len <= 0 {
		log.Error("BoxTableManager load items is none")
		return false
	}

	this.items = make(map[int32]*XmlBoxItem)
	for _, v := range tmp_cfg.Items {
		values1 := parse_xml_str_arr(v.Type1, ",")
		if values1 == nil {
			log.Error("BoxTableManager parse Type1[%v] failed", v.Type1)
			return false
		}

		item := &XmlBoxItem{}
		item.Id = v.Id

		if len(values1) >= 3 {
			item.Items = make([]*BoxItem, 3)
			item.Items[0] = &BoxItem{}
			item.Items[0].Type = values1[0]
			item.Items[0].Id = values1[1]
			item.Items[0].Num = values1[2]
			log.Debug("BoxTableManager load BoxItem[%v] item0[%v]", v.Id, *item.Items[0])
		}

		values2 := parse_xml_str_arr(v.Type2, ",")
		if values2 == nil {
			log.Error("BoxTableManager parse Type2[%v] failed", v.Type2)
			return false
		}
		if len(values2) >= 3 {
			item.Items[1] = &BoxItem{}
			item.Items[1].Type = values2[0]
			item.Items[1].Id = values2[1]
			item.Items[1].Num = values2[2]

			log.Debug("BoxTableManager load BoxItem[%v] item1[%v]", v.Id, *item.Items[1])
		}

		values3 := parse_xml_str_arr(v.Type3, ",")
		if values3 == nil {
			log.Error("BoxTableManager parse Type2[%v] failed", v.Type3)
			return false
		}
		if len(values3) >= 3 {
			item.Items[2] = &BoxItem{}
			item.Items[2].Type = values3[0]
			item.Items[2].Id = values3[1]
			item.Items[2].Num = values3[2]

			log.Debug("BoxTableManager load BoxItem[%v] item2[%v]", v.Id, *item.Items[2])
		}

		this.items[v.Id] = item
	}

	log.Info("Box table load items count(%v)", tmp_len)

	return true
}

func (this *BoxTableManager) GetItem(id int32) *XmlBoxItem {
	item, o := this.items[id]
	if !o {
		return nil
	}
	return item
}
