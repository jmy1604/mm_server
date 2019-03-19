package tables

import (
	"encoding/xml"
	"io/ioutil"
	"mm_server/libs/log"
	"mm_server/src/server_config"
)

type XmlCharacterItem struct {
	Id                     int32  `xml:"Id,attr"`
	Color                  int32  `xml:"Color,attr"`
	MainColor              int32  `xml:"MainColor,attr`
	PieceStr               string `xml:"Piece,attr"`
	PieceId                int32
	PieceNum               int32
	Stone                  int32  `xml:"Stone,attr"`
	RangeStr               string `xml:"Range,attr"`
	RangeMin               int32
	RangeMax               int32
	CoinAbilityRangeStr    string `xml:"CoinAbilityRange,attr"`
	CoinAbilityRangeMin    int32
	CoinAbilityRangeMax    int32
	ExploreAbilityRange    string `xml:"ExploreAbilityRange,attr"`
	ExploreAbilityRangeMin int32
	ExploreAbilityRangeMax int32
	MatchAbilityRange      string `xml:"MatchAbilityRange,attr"`
	MatchAbilityRangeMin   int32
	MatchAbilityRangeMax   int32
	GrowthRate             int32  `xml:"GrowthRate,attr"`
	InitialRate            int32  `xml:"InitialRate,attr"`
	SkillId                int32  `xml:"SkillId,attr"`
	UpgradeExpStr          string `xml:"UpgradeExp,attr"`
	UpgradeExps            []int32
	UpstarCostStr          string `xml:"UpstarCost,attr"`
	UpStarCosts            []int32
	UpstarCatStr           string `xml:"UpstarCat,attr"`
	UpstarCats             []int32
	UpstarMaxLevelStr      string `xml:"UpstarMaxLevel,attr"`
	UpstarMaxLevels        []int32
	AddCoinStr             string `xml:"AddCoin,attr"`
	AddCoins               []int32
	AddExploreStr          string `xml:"AddExplore,attr"`
	AddExplores            []int32
	AddMatchStr            string `xml:"AddMatch,attr"`
	AddMatchs              []int32
	UpSkillStr             string `xml:"UpSkill,attr"`
	UpSkills               []int32
	UpSkillCostStr         string `xml:"UpSkillCost,attr"`
	UpSkillCosts           []int32
	Rarity                 int32  `xml:"Rarity,attr"`
	AvatarId               int32  `xml:"AvatarId,attr"`
	FeedCostStr            string `xml:"FeedCost,attr"`
	FeedCosts              []int32
	CriticalChanceStr      string `xml:"CriticalChance,attr"`
	CriticalChances        []int32
	SkillLevelScoreStr     string `xml:"SkillLevelScore,attr"`
	SkillLevelScores       []int32
}

func (this *XmlCharacterItem) GetMaxLevel() int32 {
	return int32(len(this.UpgradeExps))
}

func (this *XmlCharacterItem) GetMaxStar() int32 {
	return int32(len(this.UpStarCosts))
}

func (this *XmlCharacterItem) GetMaxSkillLevel() int32 {
	return int32(len(this.UpSkillCosts))
}

// 等级经验
func (this *XmlCharacterItem) GetLevelExp(level int32) int32 {
	if level < 1 || int(level) > len(this.UpgradeExps) {
		return -1
	}
	return this.UpgradeExps[level-1]
}

// 星级消耗金币
func (this *XmlCharacterItem) GetStarCostCoin(star int32) int32 {
	if star < 1 || int(star) > len(this.UpStarCosts) {
		return -1
	}
	return this.UpStarCosts[star-1]
}

// 升星消耗猫数
func (this *XmlCharacterItem) GetUpstarCostCatNum(star int32) int32 {
	if star < 1 || int(star) > len(this.UpstarCats) {
		return -1
	}
	return this.UpstarCats[star-1]
}

// 星级对应最大等级
func (this *XmlCharacterItem) GetStarMaxLevel(star int32) int32 {
	if star < 1 || int(star) > len(this.UpstarMaxLevels) {
		return -1
	}
	return this.UpstarMaxLevels[star-1]
}

// 星级对应探索能力
func (this *XmlCharacterItem) GetStarExplore(star int32) int32 {
	if star < 1 || int(star) > len(this.AddExplores) {
		return -1
	}
	return this.AddExplores[star-1]
}

// 星级对应最大产金
func (this *XmlCharacterItem) GetStarAddCoin(star int32) int32 {
	if star < 1 || int(star) > len(this.AddCoins) {
		return -1
	}
	return this.AddCoins[star-1]
}

// 星级对应消除能力
func (this *XmlCharacterItem) GetStarMatch(star int32) int32 {
	if star < 1 || int(star) > len(this.AddMatchs) {
		return -1
	}
	return this.AddMatchs[star-1]
}

// 星级对应金币消耗
func (this *XmlCharacterItem) GetSkillLevelCostCoin(skill_level int32) int32 {
	if skill_level < 1 || int(skill_level) > len(this.UpSkillCosts) {
		return -1
	}
	return this.UpSkillCosts[skill_level-1]
}

type XmlCharacterConfig struct {
	Items []XmlCharacterItem `xml:"item"`
}

type CharacterTableMgr struct {
	Map map[int32]*XmlCharacterItem
}

func (this *CharacterTableMgr) GetCat(id int32) *XmlCharacterItem {
	item, o := this.Map[id]
	if !o {
		return nil
	}
	return item
}

func (this *CharacterTableMgr) Init(table_file string) bool {
	if table_file == "" {
		table_file = "Character.xml"
	}
	file_path := server_config.GetGameDataPathFile(table_file)
	data, err := ioutil.ReadFile(file_path)
	if nil != err {
		log.Error("CharacterTableMgr Init read file err[%s] !", err.Error())
		return false
	}

	tmp_cfg := &XmlCharacterConfig{}
	err = xml.Unmarshal(data, tmp_cfg)
	if nil != err {
		log.Error("CharacterTableMgr Init xml Unmarshal failed error [%s] !", err.Error())
		return false
	}

	this.Map = make(map[int32]*XmlCharacterItem)
	tmp_len := int32(len(tmp_cfg.Items))
	var tmp_item *XmlCharacterItem
	for idx := int32(0); idx < tmp_len; idx++ {
		tmp_item = &tmp_cfg.Items[idx]

		piece_data := parse_xml_str_arr(tmp_item.PieceStr, ",")
		if len(piece_data) >= 2 {
			tmp_item.PieceId = piece_data[0]
			tmp_item.PieceNum = piece_data[1]
			//log.Error("Cat[%v] Piece Column data len %v invalid, not enough length", tmp_item.Id, len(piece_data))
			//return false
		}

		all_ability_data := parse_xml_str_arr(tmp_item.RangeStr, ",")
		if len(all_ability_data) < 2 {
			log.Error("Cat[%v] Range Column data len %v invalid, RangeStr(%v)", tmp_item.Id, len(all_ability_data), tmp_item.RangeStr)
			return false
		}
		tmp_item.RangeMin = all_ability_data[0]
		tmp_item.RangeMax = all_ability_data[1]

		coin_ability_data := parse_xml_str_arr(tmp_item.CoinAbilityRangeStr, ",")
		if len(coin_ability_data) < 2 {
			log.Error("Cat[%v] CoinAbilityRange Column data len %v invalid, CoinAbilityRangeStr(%v)", tmp_item.Id, len(coin_ability_data), tmp_item.CoinAbilityRangeStr)
			return false
		}
		if coin_ability_data[0] > coin_ability_data[1] {
			log.Error("Cat[%v] CoinAbilityRange value is invalid", tmp_item.Id)
			return false
		}
		tmp_item.CoinAbilityRangeMin = coin_ability_data[0]
		tmp_item.CoinAbilityRangeMax = coin_ability_data[1]

		explore_data := parse_xml_str_arr(tmp_item.ExploreAbilityRange, ",")
		if len(explore_data) < 2 {
			log.Error("Cat[%v] ExploreAbilityRange Column data len %v invalid", tmp_item.Id, len(explore_data))
			return false
		}
		if explore_data[0] > explore_data[1] {
			log.Error("Cat[%v] ExploreAbilityRange value is invalid", tmp_item.Id)
			return false
		}
		tmp_item.ExploreAbilityRangeMin = explore_data[0]
		tmp_item.ExploreAbilityRangeMax = explore_data[1]

		match_data := parse_xml_str_arr(tmp_item.MatchAbilityRange, ",")
		if len(match_data) < 2 {
			log.Error("Cat[%v] MatchAbilityRange Column data len %v invalid", tmp_item.Id, len(match_data))
			return false
		}
		if match_data[0] > match_data[1] {
			log.Error("Cat[%v] MatchAbilityRange value is invalid", tmp_item.Id)
			return false
		}
		tmp_item.MatchAbilityRangeMin = match_data[0]
		tmp_item.MatchAbilityRangeMax = match_data[1]

		tmp_item.UpstarCats = parse_xml_str_arr(tmp_item.UpstarCatStr, ",")
		tmp_item.UpstarMaxLevels = parse_xml_str_arr(tmp_item.UpstarMaxLevelStr, ",")
		tmp_item.AddCoins = parse_xml_str_arr(tmp_item.AddCoinStr, ",")
		tmp_item.AddMatchs = parse_xml_str_arr(tmp_item.AddMatchStr, ",")
		tmp_item.AddExplores = parse_xml_str_arr(tmp_item.AddExploreStr, ",")

		tmp_item.UpgradeExps = parse_xml_str_arr(tmp_item.UpgradeExpStr, ",")
		tmp_item.UpStarCosts = parse_xml_str_arr(tmp_item.UpSkillCostStr, ",")
		tmp_item.UpSkills = parse_xml_str_arr(tmp_item.UpSkillStr, ",")
		tmp_item.UpSkillCosts = parse_xml_str_arr(tmp_item.UpSkillCostStr, ",")
		tmp_item.FeedCosts = parse_xml_str_arr(tmp_item.FeedCostStr, ",")
		if len(tmp_item.FeedCosts) < len(tmp_item.UpgradeExps) {
			log.Error("Cat[%v] index[%v] FeedCost field array size must not less than UpgradeExp field")
			return false
		}
		tmp_item.CriticalChances = parse_xml_str_arr(tmp_item.CriticalChanceStr, ",")
		if len(tmp_item.CriticalChances) < len(tmp_item.UpgradeExps) {
			log.Error("Cat[%v] index[%v] CriticalChance field array size must not less than UpgradeExp field")
			return false
		}

		tmp_item.SkillLevelScores = parse_xml_str_arr(tmp_item.SkillLevelScoreStr, ",")
		if len(tmp_item.SkillLevelScores) < len(tmp_item.UpStarCosts) {
			log.Error("Cat[%v] index[%v] SkillLevelScore field array size must no less than UpstarCosts field")
			return false
		}

		this.Map[tmp_item.Id] = tmp_item
	}

	return true
}
