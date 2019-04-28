package common

import (
	"mm_server/libs/utils"
	"time"
)

const (
	RANK_LIST_TYPE_NONE              = iota
	RANK_LIST_TYPE_STAGE_TOTAL_SCORE = 1 // 关卡总分
	RANK_LIST_TYPE_CHARM             = 2 // 魅力
	RANK_LIST_TYPE_CAT_OUQI          = 3 // 猫欧气
	RANK_LIST_TYPE_BE_ZANED          = 4 // 被赞
	RANK_LIST_TYPE_MAX               = 10
)

type PlayerInt32RankItem struct {
	Value      int32
	UpdateTime int32
	PlayerId   int32
}

type PlayerInt64RankItem struct {
	Value      int64
	UpdateTime int32
	PlayerId   int32
}

type PlayerCatOuqiRankItem struct {
	PlayerId   int32
	CatId      int32
	Ouqi       int32
	UpdateTime int32
}

func (this *PlayerInt32RankItem) Less(value interface{}) bool {
	item := value.(*PlayerInt32RankItem)
	if item == nil {
		return false
	}
	if this.Value < item.Value {
		return true
	} else if this.Value == item.Value {
		if this.UpdateTime > item.UpdateTime {
			return true
		}
		if this.UpdateTime == item.UpdateTime {
			if this.PlayerId > item.PlayerId {
				return true
			}
		}
	}
	return false
}

func (this *PlayerInt32RankItem) Greater(value interface{}) bool {
	item := value.(*PlayerInt32RankItem)
	if item == nil {
		return false
	}
	if this.Value > item.Value {
		return true
	} else if this.Value == item.Value {
		if this.UpdateTime < item.UpdateTime {
			return true
		}
		if this.UpdateTime == item.UpdateTime {
			if this.PlayerId < item.PlayerId {
				return true
			}
		}
	}
	return false
}

func (this *PlayerInt32RankItem) KeyEqual(value interface{}) bool {
	item := value.(*PlayerInt32RankItem)
	if item == nil {
		return false
	}
	if item == nil {
		return false
	}
	if this.PlayerId == item.PlayerId {
		return true
	}
	return false
}

func (this *PlayerInt32RankItem) GetKey() interface{} {
	return this.PlayerId
}

func (this *PlayerInt32RankItem) GetValue() interface{} {
	return this.Value
}

func (this *PlayerInt32RankItem) SetValue(value interface{}) {
	this.Value = value.(int32)
	this.UpdateTime = int32(time.Now().Unix())
}

func (this *PlayerInt32RankItem) New() utils.SkiplistNode {
	return &PlayerInt32RankItem{}
}

func (this *PlayerInt32RankItem) Assign(node utils.SkiplistNode) {
	n := node.(*PlayerInt32RankItem)
	if n == nil {
		return
	}
	this.Value = n.Value
	this.UpdateTime = n.UpdateTime
	this.PlayerId = n.PlayerId
}

func (this *PlayerInt32RankItem) CopyDataTo(node interface{}) {
	n := node.(*PlayerInt32RankItem)
	if n == nil {
		return
	}
	n.Value = this.Value
	n.UpdateTime = this.UpdateTime
	n.PlayerId = this.PlayerId
}

func (this *PlayerInt64RankItem) Less(value interface{}) bool {
	item := value.(*PlayerInt64RankItem)
	if item == nil {
		return false
	}
	if this.Value < item.Value {
		return true
	} else if this.Value == item.Value {
		if this.UpdateTime > item.UpdateTime {
			return true
		}
		if this.UpdateTime == item.UpdateTime {
			if this.PlayerId > item.PlayerId {
				return true
			}
		}
	}
	return false
}

func (this *PlayerInt64RankItem) Greater(value interface{}) bool {
	item := value.(*PlayerInt64RankItem)
	if item == nil {
		return false
	}
	if this.Value > item.Value {
		return true
	} else if this.Value == item.Value {
		if this.UpdateTime < item.UpdateTime {
			return true
		}
		if this.UpdateTime == item.UpdateTime {
			if this.PlayerId < item.PlayerId {
				return true
			}
		}
	}
	return false
}

func (this *PlayerInt64RankItem) KeyEqual(value interface{}) bool {
	item := value.(*PlayerInt64RankItem)
	if item == nil {
		return false
	}
	if item == nil {
		return false
	}
	if this.PlayerId == item.PlayerId {
		return true
	}
	return false
}

func (this *PlayerInt64RankItem) GetKey() interface{} {
	return this.PlayerId
}

func (this *PlayerInt64RankItem) GetValue() interface{} {
	return this.Value
}

func (this *PlayerInt64RankItem) SetValue(value interface{}) {
	this.Value = value.(int64)
	this.UpdateTime = int32(time.Now().Unix())
}

func (this *PlayerInt64RankItem) New() utils.SkiplistNode {
	return &PlayerInt64RankItem{}
}

func (this *PlayerInt64RankItem) Assign(node utils.SkiplistNode) {
	n := node.(*PlayerInt64RankItem)
	if n == nil {
		return
	}
	this.Value = n.Value
	this.UpdateTime = n.UpdateTime
	this.PlayerId = n.PlayerId
}

func (this *PlayerInt64RankItem) CopyDataTo(node interface{}) {
	n := node.(*PlayerInt64RankItem)
	if n == nil {
		return
	}
	n.Value = this.Value
	n.UpdateTime = this.UpdateTime
	n.PlayerId = this.PlayerId
}

// --------------------------- PlayerCatOuqiRankItem --------------------------
func (this *PlayerCatOuqiRankItem) Less(value interface{}) bool {
	item := value.(*PlayerCatOuqiRankItem)
	if item == nil {
		return false
	}
	if this.Ouqi < item.Ouqi {
		return true
	} else if this.Ouqi == item.Ouqi {
		if this.UpdateTime > item.UpdateTime {
			return true
		}
		if this.UpdateTime == item.UpdateTime {
			if this.PlayerId > item.PlayerId {
				return true
			}
			if this.PlayerId == item.PlayerId {
				if this.CatId > item.CatId {
					return true
				}
			}
		}
	}
	return false
}

func (this *PlayerCatOuqiRankItem) Greater(value interface{}) bool {
	item := value.(*PlayerCatOuqiRankItem)
	if item == nil {
		return false
	}
	if this.Ouqi > item.Ouqi {
		return true
	} else if this.Ouqi == item.Ouqi {
		if this.UpdateTime < item.UpdateTime {
			return true
		}
		if this.UpdateTime == item.UpdateTime {
			if this.PlayerId < item.PlayerId {
				return true
			}
			if this.PlayerId == item.PlayerId {
				if this.Ouqi < item.Ouqi {
					return true
				}
			}
		}
	}
	return false
}

func (this *PlayerCatOuqiRankItem) KeyEqual(value interface{}) bool {
	item := value.(*PlayerCatOuqiRankItem)
	if item == nil {
		return false
	}
	if item == nil {
		return false
	}
	if this.PlayerId == item.PlayerId && this.CatId == item.CatId {
		return true
	}
	return false
}

func (this *PlayerCatOuqiRankItem) GetKey() interface{} {
	return utils.Int64From2Int32(this.PlayerId, this.CatId)
}

func (this *PlayerCatOuqiRankItem) GetValue() interface{} {
	return this.Ouqi
}

func (this *PlayerCatOuqiRankItem) SetValue(value interface{}) {
	this.Ouqi = value.(int32)
	this.UpdateTime = int32(time.Now().Unix())
}

func (this *PlayerCatOuqiRankItem) New() utils.SkiplistNode {
	return &PlayerCatOuqiRankItem{}
}

func (this *PlayerCatOuqiRankItem) Assign(node utils.SkiplistNode) {
	n := node.(*PlayerCatOuqiRankItem)
	if n == nil {
		return
	}
	this.Ouqi = n.Ouqi
	this.UpdateTime = n.UpdateTime
	this.PlayerId = n.PlayerId
	this.CatId = n.CatId
}

func (this *PlayerCatOuqiRankItem) CopyDataTo(node interface{}) {
	n := node.(*PlayerCatOuqiRankItem)
	if n == nil {
		return
	}
	n.Ouqi = this.Ouqi
	n.UpdateTime = this.UpdateTime
	n.PlayerId = this.PlayerId
	n.CatId = this.CatId
}
