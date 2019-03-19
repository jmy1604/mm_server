package tables

import (
	"encoding/xml"
	"io/ioutil"
	"mm_server/libs/log"
	"mm_server/src/server_config"
)

type XmlPlayerLevelItem struct {
	Level        int32 `xml:"Level,attr"`
	MaxExp       int32 `xml:"MaxExp,attr"`
	MaxFarm      int32 `xml:"MaxFarm,attr"`
	MaxCattery   int32 `xml:"MaxCattery,attr"`
	MaxPower     int32 `xml:"MaxPower,attr"`
	FosteredSlot int32 `xml:"BeFriendFosterSlot,attr"` // 被寄养上限
	FosterSlot   int32 `xml:"FriendFosterSlot,attr"`   // 寄养上限
}

type XmlPlayerLevelConfig struct {
	Items []XmlPlayerLevelItem `xml:"item"`
}

type PlayerLevelTableManager struct {
	Map      map[int32]*XmlPlayerLevelItem
	Array    []*XmlPlayerLevelItem
	MaxLevel int32
}

func (this *PlayerLevelTableManager) Init(table_file string) bool {
	if table_file == "" {
		table_file = "PlayerLevel.xml"
	}
	file_path := server_config.GetGameDataPathFile(table_file)
	data, err := ioutil.ReadFile(file_path)
	if nil != err {
		log.Error("PlayerLevelTableManager Load read file err[%s] !", err.Error())
		return false
	}

	tmp_cfg := &XmlPlayerLevelConfig{}
	err = xml.Unmarshal(data, tmp_cfg)
	if nil != err {
		log.Error("PlayerLevelTableManager Load xml Unmarshal failed error [%s] !", err.Error())
		return false
	}

	if this.Map == nil {
		this.Map = make(map[int32]*XmlPlayerLevelItem)
	}
	if this.Array == nil {
		this.Array = make([]*XmlPlayerLevelItem, 0)
	}
	tmp_len := int32(len(tmp_cfg.Items))
	var tmp_item *XmlPlayerLevelItem
	for idx := int32(0); idx < tmp_len; idx++ {
		tmp_item = &tmp_cfg.Items[idx]
		this.Map[tmp_item.Level] = tmp_item
		this.Array = append(this.Array, tmp_item)
	}

	this.MaxLevel = tmp_len

	return true
}
