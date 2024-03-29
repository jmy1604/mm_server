package main

import (
	"math/rand"
	"mm_server/libs/log"
	"mm_server/proto/gen_go/client_message"
	"time"
)

func rand31n_from_range(min, max int32) (bool, int32) {
	if min > max {
		return false, 0
	} else if min == max {
		return true, min
	}
	return true, (rand.Int31n(max-min+1) + min)
}

func randn_different(array []int32, n int32) (nums []int32) {
	l := int32(len(array))
	if n > l {
		return
	}

	for i := int32(0); i < n; i++ {
		r := rand.Int31n(l)
		if nums != nil {
			for n := int32(0); n < l; n++ {
				f := false
				for j := 0; j < len(nums); j++ {
					if array[r] == nums[j] {
						f = true
						break
					}
				}
				if !f {
					break
				}
				r = (r + 1) % l
			}
		}
		nums = append(nums, array[r])
	}
	return
}

func GetRemainSeconds(start_time int32, duration int32) int32 {
	now := time.Now().Unix()
	if duration <= (int32(now) - start_time) {
		return 0
	}
	return duration - (int32(now) - start_time)
}

func GetRoundValue(value float32) int32 {
	v := int32(value)
	if value-float32(v) >= float32(0.5) {
		return v + 1
	} else {
		return v
	}
}

func GetPlayerBaseInfo(player_id int32) (name string, level int32, head int32) {
	player := player_mgr.GetPlayerById(player_id)
	if player != nil {
		name = player.db.GetName()
		level = player.db.GetLevel()
		head = player.db.Info.GetHead()
	}
	return
}

func Map2ItemInfos(items map[int32]int32) (item_infos []*msg_client_message.ItemInfo) {
	if items == nil {
		return
	}
	for k, v := range items {
		item_infos = append(item_infos, &msg_client_message.ItemInfo{
			ItemCfgId: k,
			ItemNum:   v,
		})
	}
	return
}

// 分离本服务器与其他服务器的玩家ID，返回值表示索引之前包括该索引都是本服务器的玩家
func SplitLocalAndRemotePlayers(player_ids []int32) int32 {
	if player_ids == nil || len(player_ids) == 0 {
		return -1
	}

	var found_local bool
	var i, j, tmp int32
	l := int32(len(player_ids))
	i = int32(l) - 1
	j = 0
	for j <= i {
		// 本服务器
		for ; i >= j; i-- {
			id := player_ids[i]
			if player_mgr.GetPlayerById(id) != nil {
				found_local = true
				break
			}
		}
		// 其他服务器
		for ; j <= i; j++ {
			id := player_ids[j]
			if player_mgr.GetPlayerById(id) == nil {
				break
			}
		}
		// 交换
		if i > j {
			tmp = player_ids[i]
			player_ids[i] = player_ids[j]
			player_ids[j] = tmp
		}
	}

	if !found_local {
		i = -1
	}

	log.Trace("splited player_ids: %v, idx: %v", player_ids, i)

	return i
}
