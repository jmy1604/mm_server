package tables

import (
	"encoding/json"
	"io/ioutil"
	"mm_server/libs/log"
	"mm_server/src/server_config"
)

const (
	BUILD_AREA_UNLOCK_TYPE_COIN    = 17
	BUILD_AREA_UNLOCK_TYPE_DIAMOND = 18
	BUILD_AREA_UNLOCK_TYPE_RMB     = 49

	BUILD_AREA_TYPE_GROUND = 1
	BUILD_AREA_TYPE_WATER  = 2
)

type GridRectConfig struct {
	X int32 `json:"x"`
	Y int32 `json:"y"`
	Z int32 `json:"z"`
	W int32 `json:"w"`
}

type AreaXY struct {
	X int32 `json:"x"`
	Y int32 `json:"y"`
}

type SingleBuildArea struct {
	Index         int32          `json:"index"`
	GridRect      GridRectConfig `json:"gridRect"`
	AddGrids      []*AreaXY      `json:"addGrids"`
	DelGrids      []*AreaXY      `json:"delGrids"`
	Geography     int32          `json:"geography"`
	Requirements  []int32        `json:"requirements"`
	ArenaXYsMap   map[int32]bool
	ArenaXYsArray []int32
	ArenaXYCount  int32

	DefMapBuildings map[int32]*DefaultMapBuilding
}

type BulidAreasConfig struct {
	Areas []*SingleBuildArea `json:"Area"`
}

type DefaultMapBuilding struct {
	Id       int32   `json:"id"`
	Rotation int32   `json:"rotation"`
	Point    []int32 `json:"point"`
}

type DefaultMapBuildingConfig struct {
	Buildings []DefaultMapBuilding `json:"buildings"`
}

type BuildAreaMgr struct {
	Idx2area      map[int32]*SingleBuildArea
	AreaXY2Type   map[int32]int32
	AreaXY2AreaId map[int32]int32

	DefaultMapBuildingCount int32
	DefaultMapBuildings     []*DefaultMapBuilding

	//InitAreaIds []int32
}

func (this *BuildAreaMgr) Init(area_config, build_map string) bool {
	if !this.LoadArea(area_config) {
		return false
	}

	if !this.LoadDefMapBuildings(build_map) {
		return false
	}

	return true
}

func (this *BuildAreaMgr) LoadArea(area_config string) bool {
	if area_config == "" {
		area_config = "Area.txt"
	}
	table_path := server_config.GetGameDataPathFile(area_config)
	data, err := ioutil.ReadFile(table_path)
	if nil != err {
		log.Error("BuildAreaMgr LoadArea read file err[%s] !", err.Error())
		return false
	}

	tmp_cfg := &BulidAreasConfig{}
	err = json.Unmarshal(data, tmp_cfg)
	if nil != err {
		log.Error("BuildAreaMgr LoadArea json Unmarshal failed error [%s] !", err.Error())
		return false
	}

	this.Idx2area = make(map[int32]*SingleBuildArea)
	this.AreaXY2Type = make(map[int32]int32)
	this.AreaXY2AreaId = make(map[int32]int32)
	tmp_len := int32(len(tmp_cfg.Areas))
	//this.InitAreaIds = make([]int32, 0, tmp_len)
	var tmp_area *SingleBuildArea
	var tmp_x, tmp_y int32
	var tmp_xy *AreaXY
	var area_xy int32
	for idx := int32(0); idx < tmp_len; idx++ {
		tmp_area = tmp_cfg.Areas[idx]
		if nil == tmp_area {
			continue
		}

		tmp_area.ArenaXYsMap = make(map[int32]bool)

		// 范围内区域
		for tmp_x = int32(0); tmp_x <= tmp_area.GridRect.Z; tmp_x++ {
			for tmp_y = int32(0); tmp_y <= tmp_area.GridRect.W; tmp_y++ {
				area_xy = ((tmp_x + tmp_area.GridRect.X) << 16) | ((tmp_y + tmp_area.GridRect.Y) & 0x0000FFFF)
				tmp_area.ArenaXYsMap[area_xy] = true
				this.AreaXY2Type[area_xy] = tmp_area.Geography
				this.AreaXY2AreaId[area_xy] = tmp_area.Index
			}
		}

		// 增加区域
		for _, tmp_xy = range tmp_area.AddGrids {
			area_xy = ((tmp_xy.X + tmp_area.GridRect.X) << 16) | ((tmp_xy.Y + tmp_area.GridRect.Y) & 0x0000FFFF)
			tmp_area.ArenaXYsMap[area_xy] = true
			this.AreaXY2Type[area_xy] = tmp_area.Geography
			this.AreaXY2AreaId[area_xy] = tmp_area.Index
		}

		// 删除区域
		for _, tmp_xy = range tmp_area.DelGrids {
			area_xy = ((tmp_xy.X + tmp_area.GridRect.X) << 16) | ((tmp_xy.Y + tmp_area.GridRect.Y) & 0x0000FFFF)
			tmp_area.ArenaXYsMap[area_xy] = false
			delete(this.AreaXY2Type, area_xy)
			delete(this.AreaXY2AreaId, area_xy)
		}

		/*
			if 0 != tmp_arena.UnlockMoneyType {
				tmp_arena.UnlockCost = make([]int32, 2)
				tmp_arena.UnlockCost[0] = tmp_arena.UnlockMoneyType
				tmp_arena.UnlockCost[1] = tmp_arena.UnlockMoneyCost
			}

			if 0 != tmp_arena.QuickUnlockMoneyType {
				tmp_arena.QuickUnlockCost = make([]int32, 2)
				tmp_arena.QuickUnlockCost[0] = tmp_arena.QuickUnlockMoneyType
				tmp_arena.QuickUnlockCost[1] = tmp_arena.QuickUnlockMoneyCost
			}

			if tmp_arena.UnlockLvl <= 0 {
				this.InitAreaIds = append(this.InitAreaIds, tmp_arena.Index)
			}
		*/

		tmp_area.DefMapBuildings = make(map[int32]*DefaultMapBuilding)

		this.Idx2area[tmp_area.Index] = tmp_area
	}

	//log.Info("初始区域 %v", this.InitAreaIds)

	return true
}

func (this *BuildAreaMgr) LoadDefMapBuildings(building_map string) bool {
	if building_map == "" {
		building_map = "buildings.json"
	}
	file_path := server_config.GetGameDataPathFile(building_map)
	data, err := ioutil.ReadFile(file_path)
	if nil != err {
		log.Error("BuildAreaMgr LoadDefMapBuildings read file err[%s] !", err.Error())
		return false
	}

	tmp_cfg := &DefaultMapBuildingConfig{}
	err = json.Unmarshal(data, tmp_cfg)
	if nil != err {
		log.Error("BuildAreaMgr LoadDefMapBuildings json Unmarshal failed error [%s] !", err.Error())
		return false
	}

	tmp_len := int32(len(tmp_cfg.Buildings))
	this.DefaultMapBuildings = make([]*DefaultMapBuilding, 0, tmp_len)

	var tmp_building *DefaultMapBuilding
	var tmp_xy int32
	var tmp_arena *SingleBuildArea
	for idx := int32(0); idx < tmp_len; idx++ {
		tmp_building = &tmp_cfg.Buildings[idx]
		if nil == tmp_building {
			continue
		}

		if 2 != len(tmp_building.Point) {
			log.Error("BuildAreaMgr LoadDefMapBuildings [%d] point[%v] error !", tmp_building.Id, tmp_building.Point)
			return false
		}

		this.DefaultMapBuildingCount++

		this.DefaultMapBuildings = append(this.DefaultMapBuildings, tmp_building)

		tmp_xy = (tmp_building.Point[0])<<16 | (tmp_building.Point[1])&0x0000FFFF

		for _, tmp_arena = range this.Idx2area {
			if tmp_arena.ArenaXYsMap[tmp_xy] {
				tmp_arena.DefMapBuildings[tmp_building.Id] = tmp_building
			}
		}
	}

	log.Info("各个区域的建筑")
	for _, tmp_arena := range this.Idx2area {
		log.Info("==================区域%d==================", tmp_arena.Index)
		for _, tmp_building := range tmp_arena.DefMapBuildings {
			log.Info("	默认建筑[%v]", *tmp_building)
		}
		log.Info("===================End===================")
	}

	return true
}

func (this *BuildAreaMgr) Getidx2area() map[int32]*SingleBuildArea {
	return this.Idx2area
}
