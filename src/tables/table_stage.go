package tables

import (
	"encoding/xml"
	"io/ioutil"
	"mm_server/libs/log"
	"mm_server/src/server_config"
	"strconv"
	"strings"
)

type XmlStageItem struct {
	StageCfgId            int32  `xml:"Id,attr"`               // 配置Id
	FirstClearRewardStr   string `xml:"FirstClearReward,attr"` // 首次通关奖励
	FirstClearReward      []int32
	FirstAllStarRewardStr string `xml:"FirstAllStarReward,attr"` // 关卡奖励
	FirstAllStarReward    int32
	FirstAllStarRewardNum int32
	CoinReward            int32  `xml:"CoinReward,attr"`   // 基础金币奖励
	ExtraReward1Str       string `xml:"ExtraReward1,attr"` // 额外奖励1
	ExtraReward1          int32
	ExtraReward1Per       int32
	ExtraReward2Str       string `xml:"ExtraReward2,attr"` // 额外奖励2
	ExtraReward2          int32
	ExtraReward2Per       int32
}

type XmlStageConfig struct {
	Items []XmlStageItem `xml:"item"`
}

type StageTableManager struct {
	Map map[int32]*XmlStageItem
}

func (this *StageTableManager) Init(table_file string) bool {
	if table_file == "" {
		table_file = "StageRewardConfig.xml"
	}
	file_path := server_config.GetGameDataPathFile(table_file)
	content, err := ioutil.ReadFile(file_path)
	if nil != err {
		log.Error("StageTableManager Load ReadFile error(%s)", err.Error())
		return false
	}

	tmp_cfg := &XmlStageConfig{}
	err = xml.Unmarshal(content, tmp_cfg)
	if nil != err {
		log.Error("CfgStageManager Load unmarshal err (%s)", err.Error())
		return false
	}

	this.Map = make(map[int32]*XmlStageItem)
	tmp_len := int32(len(tmp_cfg.Items))
	var str_len int32
	//var tmp_str string
	var tmp_item *XmlStageItem
	for idx := int32(0); idx < tmp_len; idx++ {
		tmp_item = &tmp_cfg.Items[idx]
		if nil == tmp_item {
			continue
		}

		tmp_item.FirstClearReward = parse_xml_str_arr(tmp_item.FirstClearRewardStr, ",")
		if tmp_item.FirstClearReward != nil && len(tmp_item.FirstClearReward)%2 != 0 {
			log.Error("StageTableManager parse field FirstClearReward[%v] failed", tmp_item.FirstClearRewardStr)
			return false
		}

		str_len = int32(len(tmp_item.FirstAllStarRewardStr))
		if str_len > 2 {
			tmp_item.FirstAllStarRewardStr = string([]byte(tmp_item.FirstAllStarRewardStr)[1 : str_len-1])
			str_arr := strings.Split(tmp_item.FirstAllStarRewardStr, ",")
			if len(str_arr) == 2 {
				ival, err := strconv.Atoi(str_arr[0])
				if nil != err {
					log.Error("StageTableManager load convert [%d] first_0 %s failed %s", tmp_item.StageCfgId, str_arr[0], err.Error())
					return false
				}
				tmp_item.FirstAllStarReward = int32(ival)

				ival, err = strconv.Atoi(str_arr[1])
				if nil != err {
					log.Error("StageTableManager load convert [%d] first_1 %s failed %s", tmp_item.StageCfgId, str_arr[0], err.Error())
					return false
				}
				tmp_item.FirstAllStarRewardNum = int32(ival)
			}
		}

		str_len = int32(len(tmp_item.ExtraReward1Str))
		if str_len > 0 {
			str_arr := strings.Split(tmp_item.ExtraReward1Str, ",")
			if len(str_arr) == 2 {
				ival, err := strconv.Atoi(str_arr[0])
				if nil != err {
					log.Error("StageTableManager load convert [%d] ext_reward1_0 %s failed %s", tmp_item.StageCfgId, str_arr[0], err.Error())
					return false
				}
				tmp_item.ExtraReward1 = int32(ival)

				ival, err = strconv.Atoi(str_arr[1])
				if nil != err {
					log.Error("StageTableManager load convert [%d] ext_reward1_1 %s failed %s", tmp_item.StageCfgId, str_arr[0], err.Error())
					return false
				}
				tmp_item.ExtraReward1Per = int32(ival)
			}
		}

		str_len = int32(len(tmp_item.ExtraReward2Str))
		if str_len > 0 {
			str_arr := strings.Split(tmp_item.ExtraReward2Str, ",")
			if len(str_arr) == 2 {
				ival, err := strconv.Atoi(str_arr[0])
				if nil != err {
					log.Error("StageTableManager load convert [%d] ext_reward2_0 %s failed %s", tmp_item.StageCfgId, str_arr[0], err.Error())
					return false
				}
				tmp_item.ExtraReward2 = int32(ival)

				ival, err = strconv.Atoi(str_arr[1])
				if nil != err {
					log.Error("StageTableManager load convert [%d] ext_reward2_1 %s failed %s", tmp_item.StageCfgId, str_arr[0], err.Error())
					return false
				}
				tmp_item.ExtraReward2Per = int32(ival)
			}
		}

		this.Map[tmp_item.StageCfgId] = tmp_item
	}
	return true
}
