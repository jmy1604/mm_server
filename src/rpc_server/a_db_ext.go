package main

import (
	"mm_server/libs/log"
	"mm_server/src/common"

	"time"
)

func (this *dbPlayerStageTotalScoreTable) GetAllId() []int32 {
	this.m_lock.RLock("dbPlayerStageTotalScoreTable.GetAllId")
	defer this.m_lock.RUnlock()

	var ids []int32
	for id, _ := range this.m_rows {
		ids = append(ids, id)
	}

	return ids
}

func (this *dbPlayerCharmTable) GetAllId() []int32 {
	this.m_lock.RLock("dbPlayerCharmTable.GetAllId")
	defer this.m_lock.RUnlock()

	var ids []int32
	for id, _ := range this.m_rows {
		ids = append(ids, id)
	}
	return ids
}

func (this *dbPlayerCatOuqiTable) GetAllId() []int32 {
	this.m_lock.RLock("dbPlayerCatOuqiTable.GetAllId")
	defer this.m_lock.RUnlock()

	var ids []int32
	for id, _ := range this.m_rows {
		ids = append(ids, id)
	}
	return ids
}

func (this *dbPlayerBeZanedTable) GetAllId() []int32 {
	this.m_lock.RLock("dbPlayerBeZanedTable.GetAllId")
	defer this.m_lock.RUnlock()

	var ids []int32
	for id, _ := range this.m_rows {
		ids = append(ids, id)
	}

	return ids
}

func (this *dbPlayerBeZanedRow) Zan() int32 {
	this.m_lock.Lock("dbPlayerBeZanedRow.Zan")
	defer this.m_lock.Unlock()

	this.m_Zaned += 1
	this.m_Zaned_changed = true
	this.m_UpdateTime = int32(time.Now().Unix())
	this.m_UpdateTime_changed = true
	return this.m_Zaned
}

func (this *DBC) on_preload() (err error) {
	rank_list_mgr.Init()

	ids := dbc.PlayerStageTotalScores.GetAllId()
	log.Trace("Player stage total score ids: %v", ids)
	for _, id := range ids {
		row := dbc.PlayerStageTotalScores.GetRow(id)
		if row == nil {
			continue
		}
		rank_list_mgr.UpdateItem(common.RANK_LIST_TYPE_STAGE_TOTAL_SCORE, &common.PlayerInt32RankItem{
			Value:      row.GetScore(),
			PlayerId:   row.GetPlayerId(),
			UpdateTime: row.GetUpdateTime(),
		})
	}

	ids = dbc.PlayerCharms.GetAllId()
	log.Trace("Player charm ids: %v", ids)
	for _, id := range ids {
		row := dbc.PlayerCharms.GetRow(id)
		if row == nil {
			continue
		}
		rank_list_mgr.UpdateItem(common.RANK_LIST_TYPE_CHARM, &common.PlayerInt32RankItem{
			Value:      row.GetCharmValue(),
			PlayerId:   row.GetPlayerId(),
			UpdateTime: row.GetUpdateTime(),
		})
	}

	ids = dbc.PlayerCatOuqis.GetAllId()
	log.Trace("Player cat ouqi ids: %v", ids)
	for _, id := range ids {
		row := dbc.PlayerCatOuqis.GetRow(id)
		if row == nil {
			continue
		}
		cat_ids := row.Cats.GetAllIndex()
		if cat_ids == nil {
			continue
		}
		for _, cid := range cat_ids {
			ouqi, _ := row.Cats.GetOuqi(cid)
			update_time, _ := row.Cats.GetUpdateTime(cid)
			rank_list_mgr.UpdateItem(common.RANK_LIST_TYPE_CAT_OUQI, &common.PlayerCatOuqiRankItem{
				PlayerId:   id,
				CatId:      cid,
				Ouqi:       ouqi,
				UpdateTime: update_time,
			})
		}
	}

	ids = dbc.PlayerBeZaneds.GetAllId()
	log.Trace("Player be zaned ids: %v", ids)
	for _, id := range ids {
		row := dbc.PlayerBeZaneds.GetRow(id)
		if row == nil {
			continue
		}
		rank_list_mgr.UpdateItem(common.RANK_LIST_TYPE_BE_ZANED, &common.PlayerInt32RankItem{
			Value:      row.GetZaned(),
			PlayerId:   row.GetPlayerId(),
			UpdateTime: row.GetUpdateTime(),
		})
	}

	return
}
