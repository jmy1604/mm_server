package tables

import (
	"encoding/xml"
	"io/ioutil"
	"mm_server/libs/log"
	"mm_server/src/server_config"
)

type XmlSuitItem struct {
	Id        int32  `xml:"Id,attr"`
	RewardStr string `xml:"Reward,attr"`
	Rewards   []int32
}

type XmlSuitConfig struct {
	Items []XmlSuitItem `xml:"item"`
}

type SuitTableMgr struct {
	Map   map[int32]*XmlSuitItem
	Array []*XmlSuitItem
}

func (this *SuitTableMgr) Init(table_file string) bool {
	if table_file == "" {
		table_file = "Suit.xml"
	}
	file_path := server_config.GetGameDataPathFile(table_file)
	data, err := ioutil.ReadFile(file_path)
	if nil != err {
		log.Error("SuitTableMgr read file err[%s] !", err.Error())
		return false
	}

	tmp_cfg := &XmlSuitConfig{}
	err = xml.Unmarshal(data, tmp_cfg)
	if nil != err {
		log.Error("SuitTableMgr xml Unmarshal failed error [%s] !", err.Error())
		return false
	}

	if this.Map == nil {
		this.Map = make(map[int32]*XmlSuitItem)
	}

	if this.Array == nil {
		this.Array = make([]*XmlSuitItem, 0)
	}

	tmp_len := int32(len(tmp_cfg.Items))
	var tmp_item *XmlSuitItem
	for idx := int32(0); idx < tmp_len; idx++ {
		tmp_item = &tmp_cfg.Items[idx]

		rewards := parse_xml_str_arr(tmp_item.RewardStr, ",")
		if rewards == nil || len(rewards)%2 != 0 {
			log.Error("Suit table parse field Reward[%v] error", tmp_item.RewardStr)
			return false
		}

		tmp_item.Rewards = rewards

		this.Map[tmp_item.Id] = tmp_item
		this.Array = append(this.Array, tmp_item)
	}

	return true
}

func (this *SuitTableMgr) Has(id int32) bool {
	if d := this.Map[id]; d == nil {
		return false
	}
	return true
}
