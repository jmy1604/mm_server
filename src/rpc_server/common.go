package main

func get_player_base_info(player_id int32) (has bool, name string, level int32, head int32) {
	row := dbc.PlayerBaseInfos.GetRow(player_id)
	if row == nil {
		return
	}
	name = row.GetName()
	level = row.GetLevel()
	head = row.GetHead()
	has = true
	return
}
