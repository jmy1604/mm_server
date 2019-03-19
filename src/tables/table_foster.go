package tables

import (
	"encoding/xml"
	"io/ioutil"
	"mm_server/libs/log"
	"mm_server/src/server_config"
)

type XmlFosterItem struct {
	Id                  int32  `xml:"ItemId,attr"`
	Rarity              int32  `xml:"Rarity,attr"`
	FosterTime          int32  `xml:"FosterTime,attr"`
	RewardStr           string `xml:"Reward,attr"`
	Rewards             []int32
	InnerType           int32  `xml:"InnerType,attr"`
	FusionScoreStr      string `xml:"FusionScore,attr"`
	FusionScores        []int32
	FusionTypeWeightStr string `xml:"FusionTypeWeight,attr"`
	FusionTypeWeights   []int32
	BeHitScoreStr       string `xml:"BeHitScore,attr"`
	BeHitScores         []int32
}

type XmlFosterConfig struct {
	Items []XmlFosterItem `xml:"item"`
}

type FosterTypeItems struct {
	Items []*XmlFosterItem
}

type FosterTableMgr struct {
	Map        map[int32]*XmlFosterItem
	Array      []*XmlFosterItem
	type2items map[int32]*FosterTypeItems
}

func (this *FosterTableMgr) Init(table_file string) bool {
	if table_file == "" {
		table_file = "foster.xml"
	}
	file_path := server_config.GetGameDataPathFile(table_file)
	data, err := ioutil.ReadFile(file_path)
	if nil != err {
		log.Error("FosterTableMgr read file err[%s] !", err.Error())
		return false
	}

	tmp_cfg := &XmlFosterConfig{}
	err = xml.Unmarshal(data, tmp_cfg)
	if nil != err {
		log.Error("FosterTableMgr xml Unmarshal failed error [%s] !", err.Error())
		return false
	}

	if this.Map == nil {
		this.Map = make(map[int32]*XmlFosterItem)
	}

	if this.Array == nil {
		this.Array = make([]*XmlFosterItem, 0)
	}

	if this.type2items == nil {
		this.type2items = make(map[int32]*FosterTypeItems)
	}

	tmp_len := int32(len(tmp_cfg.Items))
	var tmp_item *XmlFosterItem
	for idx := int32(0); idx < tmp_len; idx++ {
		tmp_item = &tmp_cfg.Items[idx]

		rewards := parse_xml_str_arr(tmp_item.RewardStr, ",")
		if rewards == nil || len(rewards)%2 != 0 {
			log.Error("foster table parse field Reward[%v] error", tmp_item.RewardStr)
			return false
		}
		tmp_item.Rewards = rewards

		fusion_scores := parse_xml_str_arr(tmp_item.FusionScoreStr, ",")
		if fusion_scores == nil || len(fusion_scores) < 2 || len(fusion_scores)%2 != 0 {
			log.Error("foster table parse field FusionScore[%v] error", tmp_item.FusionScoreStr)
			return false
		}
		tmp_item.FusionScores = fusion_scores

		fusion_type_weights := parse_xml_str_arr(tmp_item.FusionTypeWeightStr, ",")
		if fusion_type_weights == nil || len(fusion_type_weights) < 2 {
			log.Error("foster table parse field FusionTypeWeight[%v] error", tmp_item.FusionTypeWeightStr)
			return false
		}
		tmp_item.FusionTypeWeights = fusion_type_weights

		behit_scores := parse_xml_str_arr(tmp_item.BeHitScoreStr, ",")
		if behit_scores == nil || len(behit_scores) < 2 {
			log.Error("foster table parse field BeHitScore[%v] error", tmp_item.BeHitScoreStr)
			return false
		}
		tmp_item.BeHitScores = behit_scores

		items := this.type2items[tmp_item.InnerType]
		if items == nil {
			items = &FosterTypeItems{}
			items.Items = make([]*XmlFosterItem, 0)
			this.type2items[tmp_item.InnerType] = items
		}
		items.Items = append(items.Items, tmp_item)

		this.Map[tmp_item.Id] = tmp_item
		this.Array = append(this.Array, tmp_item)
	}

	return true
}

func (this *FosterTableMgr) Has(id int32) bool {
	if d := this.Map[id]; d == nil {
		return false
	}
	return true
}

func (this *FosterTableMgr) Get(id int32) *XmlFosterItem {
	return this.Map[id]
}

func (this *FosterTableMgr) GetInnerItems(typ int32) *FosterTypeItems {
	return this.type2items[typ]
}
