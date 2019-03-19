package tables

import (
	"encoding/xml"
	"io/ioutil"
	"mm_server/libs/log"
	"mm_server/src/server_config"
)

const (
	SHOP_TYPE_NONE = iota
	SHOP_TYPE_SPECIAL
	SHOP_TYPE_FRIEND_POINTS
	SHOP_TYPE_CHARM_MEDAL
	SHOP_TYPE_RMB
	SHOP_TYPE_SOUL_STONE
)

type XmlShopItem struct {
	Id             int32  `xml:"Id,attr"`
	Type           int32  `xml:"Tag,attr"`
	CommodityType  int32  `xml:"CommodityType,attr"`
	CommodityId    int32  `xml:"CommodityId,attr"`
	Number         int32  `xml:"Number,attr"`
	Cost           string `xml:"Cost,attr"`
	CostResourceId int32
	CostNum        int32
	LimitedType    int32  `xml:"LimitedType,attr"`
	LimitedNumber  int32  `xml:"LimitedNumber,attr"`
	LimitedTime    int32  `xml:"LimitedTime,attr"`
	CatLevel       int32  `xml:"Level,attr"`
	CatStar        int32  `xml:"Star,attr"`
	BagItemsStr    string `xml:"Show,attr"`
	BagItems       []int32
	BundleId       string `xml:"BundleID,attr"`
}

type XmlShopConfig struct {
	Items []*XmlShopItem `xml:"item"`
}

type ShopData struct {
	items     map[int32]*XmlShopItem
	limit_num int32
}

func (this *ShopData) GetItems() map[int32]*XmlShopItem {
	return this.items
}

func (this *ShopData) GetLimitNum() int32 {
	return this.limit_num
}

type ShopLimitedItems struct {
	GlobalItems     []*XmlShopItem
	GlobalItemsId   []int32
	PersonalItems   []*XmlShopItem
	PersonalItemsId []int32
}

type ShopTableManager struct {
	arr_shop  []*ShopData
	items     map[int32]*XmlShopItem
	days2item map[int32]*ShopLimitedItems
}

func (this *ShopTableManager) Init(table_file string) bool {
	if table_file == "" {
		table_file = "Shop.xml"
	}
	file_path := server_config.GetGameDataPathFile(table_file)
	data, err := ioutil.ReadFile(file_path)
	if nil != err {
		log.Error("ShopTableManager Load read file err[%s] !", err.Error())
		return false
	}

	tmp_cfg := &XmlShopConfig{}
	err = xml.Unmarshal(data, tmp_cfg)
	if nil != err {
		log.Error("ShopTableManager Load xml Unmarshal failed error [%s] !", err.Error())
		return false
	}

	max_shop_id := int32(0)
	tmp_len := int32(len(tmp_cfg.Items))
	for idx := int32(0); idx < tmp_len; idx++ {
		if tmp_cfg.Items[idx].Type > max_shop_id {
			max_shop_id = tmp_cfg.Items[idx].Type
		}
	}

	this.arr_shop = make([]*ShopData, max_shop_id)
	this.items = make(map[int32]*XmlShopItem)
	this.days2item = make(map[int32]*ShopLimitedItems)
	for i := int32(0); i < tmp_len; i++ {
		c := tmp_cfg.Items[i]
		values := parse_xml_str_arr(c.Cost, ",")
		if values == nil || len(values) < 2 {
			log.Error("ShopTableManager Load item[%v] parse Cost[%v] Field failed", c.Id, c.Cost)
			return false
		}
		c.CostResourceId = values[0]
		c.CostNum = values[1]

		s := this.arr_shop[c.Type-1]
		if s == nil {
			s = &ShopData{}
			this.arr_shop[c.Type-1] = s
		}

		if s.items == nil {
			s.items = make(map[int32]*XmlShopItem)
		}
		s.items[c.Id] = c
		if c.LimitedType == 1 || c.LimitedType == 2 || c.LimitedType == 0 {
			s.limit_num += 1
		}

		c.BagItems = parse_xml_str_arr(c.BagItemsStr, ",")
		this.items[c.Id] = c

		if c.LimitedTime > 0 {
			v := this.days2item[c.LimitedTime]
			if v == nil {
				v = &ShopLimitedItems{}
				v.GlobalItems = make([]*XmlShopItem, 0)
				v.GlobalItemsId = make([]int32, 0)
				v.PersonalItems = make([]*XmlShopItem, 0)
				v.PersonalItemsId = make([]int32, 0)
				this.days2item[c.LimitedTime] = v
			}
			if c.LimitedType == 1 {
				v.GlobalItems = append(v.GlobalItems, c)
				v.GlobalItemsId = append(v.GlobalItemsId, c.Id)
			} else if c.LimitedType == 2 {
				v.PersonalItems = append(v.PersonalItems, c)
				v.PersonalItemsId = append(v.PersonalItemsId, c.Id)
			}
		}
	}

	for days, v := range this.days2item {
		log.Info("#### 限时[%v]天商品：", days)
		log.Info("####### 全服商品 %v", v.GlobalItemsId)
		for _, gd := range v.GlobalItems {
			if gd == nil {
				continue
			}
			log.Info("######### %v", *gd)
		}
		log.Info("####### 个人商品 %v", v.PersonalItemsId)
		for _, pd := range v.PersonalItems {
			if pd == nil {
				continue
			}
			log.Info("######### %v", *pd)
		}
	}

	log.Info("Shop table load items count(%v)", tmp_len)

	return true
}

func (this *ShopTableManager) GetShop(shop_id int32) *ShopData {
	if shop_id < 1 || int(shop_id) > len(this.arr_shop) {
		return nil
	}
	return this.arr_shop[shop_id-1]
}

func (this *ShopTableManager) GetItem(item_id int32) *XmlShopItem {
	return this.items[item_id]
}

func (this *ShopTableManager) GetItems() map[int32]*XmlShopItem {
	return this.items
}

func (this *ShopTableManager) GetLimitedItems4Days(days int32) *ShopLimitedItems {
	return this.days2item[days]
}

func (this *ShopTableManager) GetAllLimitedItems4Days() map[int32]*ShopLimitedItems {
	return this.days2item
}
