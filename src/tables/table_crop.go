package tables

import (
	"encoding/xml"
	"io/ioutil"
	"mm_server/libs/log"
	"mm_server/src/server_config"
)

type XmlCropItem struct {
	Id     int32  `xml:"Id,attr"`
	Level  int32  `xml:"Level,attr"`
	Cost   int32  `xml:"Cost,attr"`
	Output int32  `xml:"OutPut,attr"`
	Time   string `xml:"Time,attr"`
	Times  []int32
	Exp    int32 `xml:"Exp,attr"`
}

type XmlCropConfig struct {
	Items []XmlCropItem `xml:"item"`
}

type CropTableMgr struct {
	Map   map[int32]*XmlCropItem
	Array []*XmlCropItem
}

func (this *CropTableMgr) Init(table_file string) bool {
	if table_file == "" {
		table_file = "crop.xml"
	}
	file_path := server_config.GetGameDataPathFile(table_file)
	data, err := ioutil.ReadFile(file_path)
	if nil != err {
		log.Error("CropTableMgr read file err[%s] !", err.Error())
		return false
	}

	tmp_cfg := &XmlCropConfig{}
	err = xml.Unmarshal(data, tmp_cfg)
	if nil != err {
		log.Error("CropTableMgr xml Unmarshal failed error [%s] !", err.Error())
		return false
	}

	if this.Map == nil {
		this.Map = make(map[int32]*XmlCropItem)
	}

	if this.Array == nil {
		this.Array = make([]*XmlCropItem, 0)
	}

	tmp_len := int32(len(tmp_cfg.Items))

	var tmp_item *XmlCropItem
	for idx := int32(0); idx < tmp_len; idx++ {
		tmp_item = &tmp_cfg.Items[idx]

		d := parse_xml_str_arr(tmp_item.Time, ",")
		if d == nil || len(d) != 2 {
			log.Error("parse field Time[%v] with column[%v] failed", tmp_item.Time, idx)
			return false
		}

		tmp_item.Times = d
		this.Map[tmp_item.Id] = tmp_item
		this.Array = append(this.Array, tmp_item)
	}

	return true
}
