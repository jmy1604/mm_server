package tables

import (
	"encoding/xml"
	"io/ioutil"
	"mm_server/libs/log"
	"mm_server/src/server_config"
)

type XmlLevelItem struct {
	Id                    int32  `xml:"Id,attr"`
	Chapter               int32  `xml:"Chapter,attr"`
	MapId                 int32  `xml:"MapId,attr"`
	PositionStr           string `xml:"Position,attr"`
	Position              []int32
	Notes                 int32  `xml:"Notes,attr"`
	NotesIcon             int32  `xml:"NotesIcon,attr"`
	NeedPower             int32  `xml:"NeedPower,attr"`
	CatChoiceLevel        int32  `xml:"CatChoiceLevel,attr"`
	CatChoiceStar         int32  `xml:"CatChoiceStar,attr"`
	ItemChoice1           int32  `xml:"ItemChoice1,attr"`
	Step                  int32  `xml:"Step,attr"`
	Time                  int32  `xml:"Time,attr"`
	RedWeight1            int32  `xml:"RedWeight1,attr"`
	YellowWeight1         int32  `xml:"YellowWeight1,attr"`
	BlueWeight1           int32  `xml:"BlueWeight1,attr"`
	GreenWeight1          int32  `xml:"GreenWeight1,attr"`
	PurpleWeight1         int32  `xml:"PurpleWeight1,attr"`
	BrownWeight1          int32  `xml:"BrownWeight1,attr"`
	RedWeight2            int32  `xml:"RedWeight2,attr"`
	YellowWeight2         int32  `xml:"YellowWeight2,attr"`
	BlueWeight2           int32  `xml:"BlueWeight2,attr"`
	GreenWeight2          int32  `xml:"GreenWeight2,attr"`
	PurpleWeight2         int32  `xml:"PurpleWeight2,attr"`
	BrownWeight2          int32  `xml:"BrownWeight2,attr"`
	PowerWeight           int32  `xml:"PowerWeight,attr"`
	PowerCorrectRatio     int32  `xml:"PowerCorrectRatio,attr"`
	SpecialElement        int32  `xml:"SpecialElement,attr"`
	ItemChoice2           int32  `xml:"ItemChoice2,attr"`
	MissionType           int32  `xml:"MissionType,attr"`
	Mission1Str           string `xml:"Mission1,attr"`
	Mission1              []*ItemInfo
	Mission2Str           string `xml:"Mission2,attr"`
	Mission2              []*ItemInfo
	Mission3Str           string `xml:"Mission3,attr"`
	Mission3              []*ItemInfo
	Mission4Str           string `xml:"Mission4,attr"`
	Mission4              []*ItemInfo
	MissionScore          int32  `xml:"MissionScore,attr"`
	StarScore1            int32  `xml:"StarScore1,attr"`
	StarScore2            int32  `xml:"StarScore2,attr"`
	StarScore3            int32  `xml:"StarScore3,attr"`
	FirstClearRewardStr   string `xml:"FirstClearReward,attr"` // 首次通关奖励
	FirstClearReward      []int32
	FirstAllStarRewardStr string `xml:"FirstAllStarReward,attr"`
	FirstAllStarReward    []int32
	CoinReward            int32  `xml:"CoinReward,attr"`
	ExtraReward1Str       string `xml:"ExtraReward1,attr"`
	ExtraReward1          []int32
	ExtraReward2Str       string `xml:"ExtraReward2,attr"`
	ExtraReward2          []int32
	N                     int32 `xml:"N,attr"`
	P1                    int32 `xml:"P1,attr"`
	P2                    int32 `xml:"P2,attr"`
	M                     int32 `xml:"M,attr"`
	NextLevel             int32 `xml:"NextLevel,attr"`
	Guidance              int32 `xml:"Guidance,attr"`
}

type XmlLevelConfig struct {
	Items []XmlLevelItem `xml:"item"`
}

type XmlChapterLevelItems struct {
	Levels []*XmlLevelItem
	Num    int32
}

type LevelTableMgr struct {
	Map            map[int32]*XmlLevelItem
	Array          []*XmlLevelItem
	Chapter2Levels map[int32]*XmlChapterLevelItems
}

func (this *LevelTableMgr) Init(table_file string) bool {
	if table_file == "" {
		table_file = "level.xml"
	}
	file_path := server_config.GetGameDataPathFile(table_file)
	data, err := ioutil.ReadFile(file_path)
	if nil != err {
		log.Error("LevelTableMgr read file err[%s] !", err.Error())
		return false
	}

	tmp_cfg := &XmlLevelConfig{}
	err = xml.Unmarshal(data, tmp_cfg)
	if nil != err {
		log.Error("LevelTableMgr xml Unmarshal failed error [%s] !", err.Error())
		return false
	}

	if this.Map == nil {
		this.Map = make(map[int32]*XmlLevelItem)
	}

	if this.Array == nil {
		this.Array = make([]*XmlLevelItem, 0)
	}

	if this.Chapter2Levels == nil {
		this.Chapter2Levels = make(map[int32]*XmlChapterLevelItems)
	}

	tmp_len := int32(len(tmp_cfg.Items))

	var tmp_item *XmlLevelItem
	chapter2levelnum := make(map[int32]int32)
	for idx := int32(0); idx < tmp_len; idx++ {
		tmp_item = &tmp_cfg.Items[idx]

		this.Map[tmp_item.Id] = tmp_item
		this.Array = append(this.Array, tmp_item)
		n := chapter2levelnum[tmp_item.Chapter]
		chapter2levelnum[tmp_item.Chapter] = n + 1
	}

	for idx := int32(0); idx < tmp_len; idx++ {
		tmp_item = &tmp_cfg.Items[idx]
		chapter_levels := this.Chapter2Levels[tmp_item.Chapter]
		if chapter_levels == nil {
			chapter_levels = &XmlChapterLevelItems{}
			num := chapter2levelnum[tmp_item.Chapter]
			if num == 0 {
				continue
			}
			chapter_levels.Levels = make([]*XmlLevelItem, num)
			this.Chapter2Levels[tmp_item.Chapter] = chapter_levels
		}

		tmp_item.FirstClearReward = parse_xml_str_arr(tmp_item.FirstClearRewardStr, ",")
		if tmp_item.FirstClearReward == nil || len(tmp_item.FirstClearReward)%2 != 0 {
			log.Error("LevelTableMgr parse field FirstClearReward[%v] failed", tmp_item.FirstClearRewardStr)
			return false
		}
		tmp_item.FirstAllStarReward = parse_xml_str_arr(tmp_item.FirstAllStarRewardStr, ",")
		if tmp_item.FirstAllStarReward == nil || len(tmp_item.FirstAllStarReward)%2 != 0 {
			log.Error("LevelTableMgr parse field FirstAllStarReward[%v] failed", tmp_item.FirstAllStarRewardStr)
			return false
		}
		tmp_item.ExtraReward1 = parse_xml_str_arr(tmp_item.ExtraReward1Str, ",")
		if tmp_item.ExtraReward1 == nil || len(tmp_item.ExtraReward1)%2 != 0 {
			log.Error("LevelTableMgr parse field ExtraReward1[%v] failed", tmp_item.ExtraReward1Str)
			return false
		}
		tmp_item.ExtraReward2 = parse_xml_str_arr(tmp_item.ExtraReward2Str, ",")
		if tmp_item.ExtraReward2 == nil || len(tmp_item.ExtraReward2)%2 != 0 {
			log.Error("LevelTableMgr parse field ExtraReward2[%v] failed", tmp_item.ExtraReward2Str)
			return false
		}

		chapter_levels.Levels[chapter_levels.Num] = tmp_item
		chapter_levels.Num += 1
	}

	return true
}

func (this *LevelTableMgr) GetChapter(chapter_id int32) *XmlChapterLevelItems {
	return this.Chapter2Levels[chapter_id]
}

func (this *LevelTableMgr) GetLevel(level_id int32) *XmlLevelItem {
	return this.Map[level_id]
}
