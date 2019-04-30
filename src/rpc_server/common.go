package main

import (
	"mm_server/libs/log"
	"mm_server/src/rpc_proto"
)

func get_player_base_info(player_id int32) (has bool, name string, level int32, head int32) {
	row := dbc.PlayerBaseInfos.GetRow(player_id)
	if row != nil {
		name = row.GetName()
		level = row.GetLevel()
		head = row.GetHead()
	} else {
		row = dbc.PlayerBaseInfos.AddRow(player_id)
	}

	if name == "" || level <= 0 {
		rpc_client := GetRpcClientByPlayerId(player_id)
		if rpc_client != nil {
			var args = rpc_proto.R2G_GetPlayerBaseInfo{
				PlayerId: player_id,
			}
			var result rpc_proto.R2G_GetPlayerBaseInfoResult
			err := rpc_client.Call("R2G_PlayerProc.GetPlayerBaseInfo", args, &result)
			if err != nil {
				log.Error("@@@ R2G_PlayerProc.GetPlayerBaseInfo err %v", err.Error())
			} else {
				if name != result.BaseInfo.Name {
					name = result.BaseInfo.Name
					row.SetName(name)
				}
				if level != result.BaseInfo.Level {
					level = result.BaseInfo.Level
					row.SetLevel(level)
				}
				if head != result.BaseInfo.Head {
					head = result.BaseInfo.Head
					row.SetHead(head)
				}
			}
		}
	}

	has = true
	return
}
