package main

import (
	"mm_server/libs/log"
	"mm_server/proto/gen_go/client_message"
	"mm_server/src/tables"

	"github.com/golang/protobuf/proto"
)

func (this *Player) DirectUnlockArea(area_id int32) {
	this.ChkUpdateMyBuildingAreas()
	if nil != this.db.Areas.Get(area_id) {
		return
	}

	area_cfg := build_area_mgr.Idx2area[area_id]
	if nil == area_cfg {
		return
	}
}

func (this *Player) UnlockArea(area_id, ifquick int32) (int32, []int32) {
	this.ChkUpdateMyBuildingAreas()
	if nil != this.db.Areas.Get(area_id) {
		log.Error("Player UnlockArea already have [%d]", area_id)
		return int32(msg_client_message.E_ERR_BUILDING_AREA_ALERADY_UNLOCKED), nil
	}

	area_cfg := build_area_mgr.Idx2area[area_id]
	if nil == area_cfg {
		log.Error("Player UnlockArea no cfg [%d]", area_id)
		return int32(msg_client_message.E_ERR_BUILDING_AREA_NO_CFG), nil
	}

	area_unlock_cfg := area_unlock_mgr.Map[area_id]

	/*
		area_unlock_cfg := cfg_areaunlock_mgr.Map[area_id]
		if nil == area_cfg {
			return int32(msg_client_message.E_ERR_BUILDING_AREA_NO_UNLOCK_CFG)
		}
	*/

	for _, pre_area := range area_unlock_cfg.FrontAreas {
		if nil == this.db.Areas.Get(pre_area) {
			log.Error("Player UnlockArea needpre [%d] [%d]", area_id, pre_area)
			return int32(msg_client_message.E_ERR_BUILDING_AREA_NEED_PRE), nil
		}
	}

	if 1 != ifquick {
		if this.db.GetLevel() < area_unlock_cfg.UnlockLevel {
			log.Error("Player UnlockArea [%d] need lvl [%d]", area_id, area_unlock_cfg.UnlockLevel)
			return int32(msg_client_message.E_ERR_BUILDING_AREA_LESS_UNLOCK_LVL), nil
		}

		if this.db.Stages.GetTotalTopStar() < area_unlock_cfg.UnlockStar {
			log.Error("Player UnlockArea [%d] need star [%d]", area_id, area_unlock_cfg.UnlockStar)
			return int32(msg_client_message.E_ERR_BUILDING_AREA_LESS_UNLOCK_STAR), nil
		}

		if !this.ChkResEnough(area_unlock_cfg.UnlockCost) {
			log.Error("Player UnlockArea [%d] need cost [%v]", area_id, area_unlock_cfg.UnlockCost)
			return int32(msg_client_message.E_ERR_BUILDING_AREA_LESS_UNLOCK_RES), nil
		}

		this.RemoveResources(area_unlock_cfg.UnlockCost, "area_unlock", "area")
	} else {
		if 2 != len(area_unlock_cfg.QuickUnlockCost) {
			return int32(msg_client_message.E_ERR_BUILDING_AREA_CANNOT_QUICK_UNLOCK), nil
		}

		if !this.ChkResEnough(area_unlock_cfg.QuickUnlockCost) {
			return int32(msg_client_message.E_ERR_BUILDING_AREA_LESS_UNLOCK_RES), nil
		}

		this.RemoveResources(area_unlock_cfg.QuickUnlockCost, "area_unlock", "area")
	}

	for tmp_xy, _ := range area_cfg.ArenaXYsMap {
		this.cur_open_pos_map[tmp_xy] = 1
		//log.Info("玩家 %d 解锁区域 添加默认位置 %d", this.Id, tmp_xy)
	}

	tmp_db := &dbPlayerAreaData{}
	tmp_db.CfgId = area_id
	this.db.Areas.Add(tmp_db)

	// 区域默认建筑
	log.Info("玩家 %d 添加区域%v 的默认建筑 %v", this.Id, area_cfg.Index, area_cfg.DefMapBuildings)
	var tmp_building *tables.DefaultMapBuilding
	//var db_building *dbPlayerBuildingData
	var new_building_id int32
	new_building_ids := make([]int32, 0, len(area_cfg.DefMapBuildings))
	for _, tmp_building = range area_cfg.DefMapBuildings {

		/*
			db_building = &dbPlayerBuildingData{}
			db_building.Id = this.db.Info.IncbyNextBuildingId(1)
			db_building.CfgId = tmp_building.Id
			db_building.Dir = tmp_building.Rotation
			db_building.X = tmp_building.Point[0]
			db_building.Y = tmp_building.Point[1]



			this.db.Buildings.Add(db_building)
		*/

		new_building_id = this.SetMapBuilding(tmp_building.Id, tmp_building.Point[0], tmp_building.Point[1], tmp_building.Rotation, true) //this.TrySetMapBuildingDefDir(tmp_building.Id)
		log.Info("添加解锁区域默认建筑 %v %v", new_building_id, tmp_building)
		if new_building_id > 0 {
			new_building_ids = append(new_building_ids, new_building_id)
			if nil != block_table_mgr.Map[tmp_building.Id] {
				this.cur_areablocknum_map[area_id] = this.cur_areablocknum_map[area_id] + 1
			}
		}

	}

	return 1, new_building_ids
}
func (this *Player) OnPlayerAreaCreate() {
	//this.InitPlayerArea()
	var tmp_building *tables.DefaultMapBuilding
	var db_building *dbPlayerBuildingData
	for idx := int32(0); idx < build_area_mgr.DefaultMapBuildingCount; idx++ {
		tmp_building = build_area_mgr.DefaultMapBuildings[idx]

		db_building = &dbPlayerBuildingData{}
		db_building.Id = this.db.Info.IncbyNextBuildingId(1)
		db_building.CfgId = tmp_building.Id
		db_building.Dir = tmp_building.Rotation
		db_building.X = tmp_building.Point[0]
		db_building.Y = tmp_building.Point[1]

		this.db.Buildings.Add(db_building)
	}
}

func (this *Player) InitPlayerArea() {
	this.ChkUpdateMyBuildingAreas()

	log.Info("============================初始化玩家[%d]区域[%v]", this.Id, area_unlock_mgr.InitAreaIds)
	for _, area_id := range area_unlock_mgr.InitAreaIds {
		this.UnlockArea(area_id, 0)
	}
}

// ----------------------------------------------------------------------------
func C2SGetAreasInfosHandler(p *Player, msg_data []byte) int32 {
	var req msg_client_message.C2SGetAreasInfos
	err := proto.Unmarshal(msg_data, &req)
	if err != nil {
		log.Error("unmarshal msg failed %v", err.Error())
		return -1
	}

	msg := &msg_client_message.S2CGetAreasInfos{}
	p.db.Areas.FillAllMsg(msg)

	p.Send(uint16(msg_client_message.S2CGetAreasInfos_ProtoID), msg)

	return 1
}

func C2SUnlockAreaHandler(p *Player, msg_data []byte) int32 {
	var req msg_client_message.C2SUnlockArea
	err := proto.Unmarshal(msg_data, &req)
	if err != nil {
		log.Error("unmarshal msg failed %v", err.Error())
		return -1
	}

	area_id := req.GetAreaId()
	ret_val, new_building_ids := p.UnlockArea(area_id, req.GetIfQuick())
	if 1 != ret_val {
		log.Error("C2SUnlockAreaHandler ret_val[%d] error ", ret_val)
		return -1
	}

	tmp_len := len(new_building_ids)
	if tmp_len > 0 {
		for i := 0; i < tmp_len; i++ {
			p.item_cat_building_change_info.building_add(p, new_building_ids[i])
		}
		p.item_cat_building_change_info.send_buildings_update(p)
	}

	msg := &msg_client_message.S2CGetAreasInfos{}
	msg.Areas = make([]*msg_client_message.AreaInfo, 1)
	msg.Areas[0] = &msg_client_message.AreaInfo{CfgId: area_id}

	p.Send(uint16(msg_client_message.S2CGetAreasInfos_ProtoID), msg)

	return 1
}
