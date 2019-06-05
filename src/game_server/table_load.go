package main

import (
	"errors"
	"mm_server/src/server_config"
	"mm_server/src/share_data"
	"mm_server/src/tables"
)

var pay_list share_data.PayChannelConfig
var global_config tables.GlobalConfig
var area_unlock_mgr tables.AreaUnlockMgr
var block_table_mgr tables.BlockTableMgr
var box_table_mgr tables.BoxTableManager
var build_area_mgr tables.BuildAreaMgr
var building_table_mgr tables.BuildingTableMgr
var cathouse_table_mgr tables.CatHouseTableMgr
var chapter_table_mgr tables.ChapterTableManager
var cat_table_mgr tables.CharacterTableMgr
var crop_table_mgr tables.CropTableMgr
var draw_table_mgr tables.DrawTableMgr
var drop_card_table_mgr tables.DropCardTableManager
var expedition_table_mgr tables.ExpeditionTableMgr
var extract_table_mgr tables.ExtractTableManager
var formula_table_mgr tables.FormulaTableMgr
var foster_table_mgr tables.FosterTableMgr
var handbook_table_mgr tables.HandbookTableMgr
var item_table_mgr tables.ItemTableMgr
var level_table_mgr tables.LevelTableMgr
var map_chest_mgr tables.MapChestMgr
var other_table_mgr tables.OtherTableManager
var pay_table_mgr tables.PayTableMgr
var player_level_table_mgr tables.PlayerLevelTableManager
var position_table tables.PositionTable
var shop_table_mgr tables.ShopTableManager
var skill_table_mgr tables.SkillTableMgr
var stage_table_mgr tables.StageTableManager
var suit_table_mgr tables.SuitTableMgr
var sysmsg_table_mgr tables.SysMsgTableMgr
var task_table_mgr tables.TaskTableMgr
var vip_table_mgr tables.VipTableMgr
var activity_table_mgr tables.ActivityTableMgr
var sub_activity_table_mgr tables.SubActivityTableMgr
var mail_table_mgr tables.MailTableMgr
var fashion_table_mgr tables.FashionTableMgr
var activity_old_table_mgr tables.ActivityOldTableMgr
var sign_table_mgr tables.SignTableMgr
var seven_days_table_mgr tables.SevenDaysTableMgr

func table_init() error {
	if !global_config.Init("") {
		return errors.New("global config init failed")
	}

	if !area_unlock_mgr.Init("") {
		return errors.New("area unlock init failed")
	}

	if !block_table_mgr.Init("") {
		return errors.New("block table init failed")
	}

	if !box_table_mgr.Init("") {
		return errors.New("box table init failed")
	}

	if !build_area_mgr.Init("", "") {
		return errors.New("build area init failed")
	}

	if !building_table_mgr.Init("") {
		return errors.New("building table init failed")
	}

	if !cathouse_table_mgr.Init("") {
		return errors.New("cat house table init failed")
	}

	if !chapter_table_mgr.Init("") {
		return errors.New("chapter table init failed")
	}

	if !cat_table_mgr.Init("") {
		return errors.New("cat table init failed")
	}

	if !crop_table_mgr.Init("") {
		return errors.New("crop table init failed")
	}

	if !draw_table_mgr.Init("") {
		return errors.New("draw table init failed")
	}

	if !drop_card_table_mgr.Init("") {
		return errors.New("drop card table init failed")
	}

	if !expedition_table_mgr.Init("", "") {
		return errors.New("expedition table init failed")
	}

	if !extract_table_mgr.Init("") {
		return errors.New("extract table init failed")
	}

	if !formula_table_mgr.Init("") {
		return errors.New("formula table init failed")
	}

	if !foster_table_mgr.Init("") {
		return errors.New("foster table init failed")
	}

	if !handbook_table_mgr.Init("") {
		return errors.New("handbook table init failed")
	}

	if !item_table_mgr.Init("", "") {
		return errors.New("item table init failed")
	}

	if !level_table_mgr.Init("") {
		return errors.New("level table init failed")
	}

	if !map_chest_mgr.Init("") {
		return errors.New("map chest init failed")
	}

	/*if !other_table_mgr.Init("") {
		return errors.New("other table init failed")
	}*/

	if !pay_table_mgr.Init("") {
		return errors.New("pay table init failed")
	}

	if !player_level_table_mgr.Init("") {
		return errors.New("player level table init failed")
	}

	if !position_table.Init("") {
		return errors.New("positioin table init failed")
	}

	if !shop_table_mgr.Init("") {
		return errors.New("shop table init failed")
	}

	if !skill_table_mgr.Init("") {
		return errors.New("skill table init failed")
	}

	if !stage_table_mgr.Init("") {
		return errors.New("stage table init failed")
	}

	if !suit_table_mgr.Init("") {
		return errors.New("suit table init failed")
	}

	if !sysmsg_table_mgr.Init("") {
		return errors.New("sysmsg table init failed")
	}

	if !task_table_mgr.Init("") {
		return errors.New("task table init failed")
	}

	if !vip_table_mgr.Init("") {
		return errors.New("vip table init failed")
	}

	if !activity_table_mgr.Init("") {
		return errors.New("activity_table_mgr init failed")
	}

	if !sub_activity_table_mgr.Init("") {
		return errors.New("sub_activity_table_mgr init failed")
	}

	if !mail_table_mgr.Init("") {
		return errors.New("mail table manager init failed")
	}

	if !fashion_table_mgr.Init("") {
		return errors.New("fashion table manager init failed")
	}

	if !activity_old_table_mgr.Init("", "", "", "") {
		return errors.New("activity old table manager init failed")
	}

	if !sign_table_mgr.Init("") {
		return errors.New("sign table manager init failed")
	}

	if !seven_days_table_mgr.Init("") {
		return errors.New("seven day table manager init failed")
	}

	if !pay_list.LoadConfig(server_config.GetConfPathFile("pay.json")) {
		return errors.New("pay channel list init failed")
	}

	return nil
}
