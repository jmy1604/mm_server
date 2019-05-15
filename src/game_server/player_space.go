package main

import (
	"mm_server/libs/log"
	"mm_server/proto/gen_go/client_message"
	"mm_server/proto/gen_go/rpc_message"
	"mm_server/src/rpc_proto"

	"github.com/golang/protobuf/proto"
)

func (this *Player) send_focus_data() int32 {
	var focus_players []*msg_client_message.FocusPlayer
	pids := this.db.FocusPlayers.GetAllIndex()
	if pids != nil && len(pids) > 0 {
		idx := SplitLocalAndRemotePlayers(pids)
		if idx >= 0 {
			for i := int32(0); i <= idx; i++ {
				p := player_mgr.GetPlayerById(pids[i])
				if p == nil {
					continue
				}
				focus_players = append(focus_players, &msg_client_message.FocusPlayer{
					Id:    p.Id,
					Name:  p.db.GetName(),
					Level: p.db.GetLevel(),
					Head:  p.db.Info.GetHead(),
				})
			}
		}
		res := this.rpc_get_players_base_info(pids[idx+1:])
		if res != nil {
			if res.PlayersInfo != nil {
				for _, pi := range res.PlayersInfo {
					focus_players = append(focus_players, &msg_client_message.FocusPlayer{
						Id:    pi.Id,
						Name:  pi.Name,
						Level: pi.Level,
						Head:  pi.Head,
					})
				}
			}
		}
	}
	response := &msg_client_message.S2CFocusDataResponse{
		BeFocusNum: this.db.FocusCommon.GetBeFocusNum(),
		Players:    focus_players,
	}
	this.Send(uint16(msg_client_message.S2CFocusDataResponse_ProtoID), response)
	log.Trace("Player %v get focus data %v", this.Id, response)
	return 1
}

func remote_focus_player(from_player_id, to_player_id int32) (resp *msg_rpc_message.G2GFocusPlayerResponse, err_code int32) {
	var req msg_rpc_message.G2GFocusPlayerRequest
	var response msg_rpc_message.G2GFocusPlayerResponse
	err_code = RemoteGetUsePB(from_player_id, rpc_proto.OBJECT_TYPE_PLAYER, to_player_id, int32(msg_rpc_message.MSGID_G2G_FOCUS_PLAYER_REQUEST), &req, &response)
	resp = &response
	return
}

func remote_focus_player_response(from_player_id int32, to_player_id int32, req_data []byte) (resp_data []byte, err_code int32) {
	var req msg_rpc_message.G2GFocusPlayerRequest
	err := _unmarshal_msg(req_data, &req)
	if err != nil {
		err_code = -1
		return
	}

	player := player_mgr.GetPlayerById(to_player_id)
	if player == nil {
		log.Error("remote request focus player by id %v not found", to_player_id)
		err_code = int32(msg_client_message.E_ERR_PLAYER_NOT_EXIST)
		return
	}

	player.db.FocusCommon.IncbyBeFocusNum(1)

	var response = msg_rpc_message.G2GFocusPlayerResponse{
		PlayerId:    to_player_id,
		PlayerName:  player.db.GetName(),
		PlayerLevel: player.db.GetLevel(),
		PlayerHead:  player.db.Info.GetHead(),
	}

	resp_data, err = _marshal_msg(&response)
	if err != nil {
		err_code = -1
		return
	}

	err_code = 1
	return
}

func (this *Player) focus_player(player_id int32) int32 {
	if this.Id == player_id {
		log.Error("Player %v cant focus self", this.Id)
		return -1
	}
	if this.db.FocusPlayers.HasIndex(player_id) {
		log.Error("Player %v already focus player %v", this.Id, player_id)
		return -1
	}

	var name string
	var level, head int32
	p := player_mgr.GetPlayerById(player_id)
	if p != nil {
		p.db.FocusCommon.IncbyBeFocusNum(1)
		name = p.db.GetName()
		level = p.db.GetLevel()
		head = p.db.Info.GetHead()
	} else {
		resp, err_code := remote_focus_player(this.Id, player_id)
		if err_code < 0 {
			return err_code
		}
		name = resp.PlayerName
		level = resp.PlayerLevel
		head = resp.PlayerHead
	}
	this.db.FocusPlayers.Add(&dbPlayerFocusPlayerData{
		PlayerId: player_id,
	})
	response := &msg_client_message.S2CFocusPlayerResponse{
		PlayerInfo: &msg_client_message.FocusPlayer{
			Id:    player_id,
			Name:  name,
			Level: level,
			Head:  head,
		},
	}
	this.Send(uint16(msg_client_message.S2CFocusPlayerResponse_ProtoID), response)
	log.Trace("Player %v focused player %v", this.Id, player_id)
	return 1
}

func remote_unfocus_player(from_player_id, to_player_id int32) (resp *msg_rpc_message.G2GUnfocusPlayerResponse, err_code int32) {
	var req msg_rpc_message.G2GUnfocusPlayerRequest
	var response msg_rpc_message.G2GUnfocusPlayerResponse
	err_code = RemoteGetUsePB(from_player_id, rpc_proto.OBJECT_TYPE_PLAYER, to_player_id, int32(msg_rpc_message.MSGID_G2G_UNFOCUS_PLAYER_REQUEST), &req, &response)
	resp = &response
	return
}

func remote_unfocus_player_response(from_player_id int32, to_player_id int32, req_data []byte) (resp_data []byte, err_code int32) {
	var req msg_rpc_message.G2GUnfocusPlayerRequest
	err := _unmarshal_msg(req_data, &req)
	if err != nil {
		err_code = -1
		return
	}

	player := player_mgr.GetPlayerById(to_player_id)
	if player == nil {
		log.Error("remote request unfocus player by id %v not found", to_player_id)
		err_code = int32(msg_client_message.E_ERR_PLAYER_NOT_EXIST)
		return
	}

	player.db.FocusCommon.IncbyBeFocusNum(-1)

	var response = msg_rpc_message.G2GUnfocusPlayerResponse{}

	resp_data, err = _marshal_msg(&response)
	if err != nil {
		err_code = -1
		return
	}

	err_code = 1
	return
}

func (this *Player) unfocus_player(player_id int32) int32 {
	if !this.db.FocusPlayers.HasIndex(player_id) {
		log.Error("Player %v no focused player %v", this.Id, player_id)
		return -1
	}
	p := player_mgr.GetPlayerById(player_id)
	if p != nil {
		p.db.FocusCommon.IncbyBeFocusNum(-1)
	} else {
		_, err_code := remote_unfocus_player(this.Id, player_id)
		if err_code < 0 {
			return err_code
		}
	}
	this.db.FocusPlayers.Remove(player_id)
	response := &msg_client_message.S2CFocusPlayerCancelResponse{
		PlayerId: player_id,
	}
	this.Send(uint16(msg_client_message.S2CFocusPlayerCancelResponse_ProtoID), response)
	log.Trace("Player %v unfocus player %v", this.Id, player_id)
	return 1
}

func (this *Player) get_my_pics() []*msg_client_message.MyPictureData {
	var my_pics []*msg_client_message.MyPictureData
	cat_ids := this.db.MyPictureDatas.GetAllIndex()
	if cat_ids != nil {
		for _, cat_id := range cat_ids {
			pos, _ := this.db.MyPictureDatas.GetPos(cat_id)
			my_pics = append(my_pics, &msg_client_message.MyPictureData{
				Pos:   pos,
				CatId: cat_id,
			})
		}
	}
	return my_pics
}

func (this *Player) send_my_picture_data() int32 {
	response := &msg_client_message.S2CMyPictureDataResponse{
		Pictures: this.get_my_pics(),
	}
	this.Send(uint16(msg_client_message.S2CMyPictureDataResponse_ProtoID), response)
	log.Trace("Player %v get my pictures data %v", this.Id, response)
	return 1
}

const (
	MY_PICTURE_NUM = 9
)

func (this *Player) my_picture_set(cat_id, cat_pos int32) int32 {
	if !this.db.Cats.HasIndex(cat_id) {
		log.Error("Player %v have no cat %v", this.Id, cat_id)
		return int32(msg_client_message.E_ERR_CAT_NOT_FOUND)
	}

	has_pic := this.db.MyPictureDatas.HasIndex(cat_id)
	if has_pic && cat_pos < 0 {
		this.db.MyPictureDatas.Remove(cat_id)
	} else {
		cat_ids := this.db.MyPictureDatas.GetAllIndex()
		if cat_ids != nil {
			if has_pic && len(cat_ids) >= MY_PICTURE_NUM {
				log.Error("Player %v only set %v picture", this.Id, MY_PICTURE_NUM)
				return -1
			}
			for _, cat_id := range cat_ids {
				pos, o := this.db.MyPictureDatas.GetPos(cat_id)
				if o && pos == cat_pos {
					log.Error("Player %v picture pos %v already has cat %v", this.Id, pos, cat_id)
					return -1
				}
			}
		}
		this.db.MyPictureDatas.SetPos(cat_id, cat_pos)
	}
	response := &msg_client_message.S2CMyPictureSetResponse{
		PictureData: &msg_client_message.MyPictureData{
			Pos:   cat_pos,
			CatId: cat_id,
		},
	}
	this.Send(uint16(msg_client_message.S2CMyPictureSetResponse_ProtoID), response)
	log.Trace("Player %v set my picture cat_id(%v) pos(%v)", this.Id, cat_id, cat_pos)
	return 1
}

func remote_space_data(from_player_id, to_player_id int32) (resp *msg_rpc_message.G2GSpaceDataResponse, err_code int32) {
	var req msg_rpc_message.G2GSpaceDataRequest
	var response msg_rpc_message.G2GSpaceDataResponse
	err_code = RemoteGetUsePB(from_player_id, rpc_proto.OBJECT_TYPE_PLAYER, to_player_id, int32(msg_rpc_message.MSGID_G2G_SPACE_DATA_REQUEST), &req, &response)
	resp = &response
	return
}

func remote_space_data_response(from_player_id int32, to_player_id int32, req_data []byte) (resp_data []byte, err_code int32) {
	var req msg_rpc_message.G2GSpaceDataRequest
	err := _unmarshal_msg(req_data, &req)
	if err != nil {
		err_code = -1
		return
	}

	player := player_mgr.GetPlayerById(to_player_id)
	if player == nil {
		log.Error("remote request player space data by id %v not found", to_player_id)
		err_code = int32(msg_client_message.E_ERR_PLAYER_NOT_EXIST)
		return
	}

	var pics map[int32]int32
	cat_ids := player.db.MyPictureDatas.GetAllIndex()
	if cat_ids != nil {
		for _, cat_id := range cat_ids {
			pos, o := player.db.MyPictureDatas.GetPos(cat_id)
			if o {
				if pics == nil {
					pics = make(map[int32]int32)
				}
				pics[cat_id] = pos
			}
		}
	}

	var response = msg_rpc_message.G2GSpaceDataResponse{
		PlayerName:  player.db.GetName(),
		PlayerLevel: player.db.GetLevel(),
		PlayerHead:  player.db.Info.GetHead(),
		Zan:         player.db.Info.GetZan(),
		BeFocusNum:  player.db.FocusCommon.GetBeFocusNum(),
		Charm:       player.db.Info.GetCharmVal(),
		Pictures:    pics,
	}

	resp_data, err = _marshal_msg(&response)
	if err != nil {
		err_code = -1
		return
	}

	err_code = 1
	return
}

func (this *Player) space_data(player_id int32) int32 {
	if this.Id == player_id {
		return -1
	}

	var name string
	var level, head, zan, charm, be_focus_num int32
	var player_pics []*msg_client_message.MyPictureData
	p := player_mgr.GetPlayerById(player_id)
	if p != nil {
		name = p.db.GetName()
		level = p.db.GetLevel()
		head = p.db.Info.GetHead()
		zan = p.db.Info.GetZan()
		charm = p.db.Info.GetCharmVal()
		be_focus_num = p.db.FocusCommon.GetBeFocusNum()
		player_pics = p.get_my_pics()
	} else {
		resp, err_code := remote_space_data(this.Id, player_id)
		if err_code < 0 {
			return err_code
		}
		name = resp.PlayerName
		level = resp.PlayerLevel
		head = resp.PlayerHead
		zan = resp.Zan
		be_focus_num = resp.BeFocusNum
		charm = resp.Charm
		if resp.Pictures != nil {
			for cat_id, pos := range resp.Pictures {
				player_pics = append(player_pics, &msg_client_message.MyPictureData{
					CatId: cat_id,
					Pos:   pos,
				})
			}
		}
	}

	response := &msg_client_message.S2CSpaceDataResponse{
		PlayerId:    player_id,
		PlayerName:  name,
		PlayerLevel: level,
		PlayerHead:  head,
		Zaned:       zan,
		Charm:       charm,
		BeFocusNum:  be_focus_num,
		Pictures:    player_pics,
	}
	this.Send(uint16(msg_client_message.S2CSpaceDataResponse_ProtoID), response)
	log.Trace("Player %v get player %v space data %v", this.Id, player_id, response)
	return 1
}

func remote_space_cat(from_player_id, to_player_id, to_cat_id int32) (resp *msg_rpc_message.G2GSpaceCatResponse, err_code int32) {
	var req msg_rpc_message.G2GSpaceCatRequest
	var response msg_rpc_message.G2GSpaceCatResponse
	err_code = RemoteGetUsePB(from_player_id, rpc_proto.OBJECT_TYPE_PLAYER, to_player_id, int32(msg_rpc_message.MSGID_G2G_SPACE_CAT_REQUEST), &req, &response)
	resp = &response
	return
}

func remote_space_cat_response(from_player_id int32, to_player_id int32, req_data []byte) (resp_data []byte, err_code int32) {
	var req msg_rpc_message.G2GSpaceCatRequest
	err := _unmarshal_msg(req_data, &req)
	if err != nil {
		err_code = -1
		return
	}
	cat_id := req.GetCatId()
	player := player_mgr.GetPlayerById(to_player_id)
	if player == nil {
		log.Error("remote request player %v space cat %v not found", to_player_id, cat_id)
		err_code = int32(msg_client_message.E_ERR_PLAYER_NOT_EXIST)
		return
	}

	if !player.db.Cats.HasIndex(cat_id) {
		log.Error("remote request player %v space cat %v not found", to_player_id, cat_id)
		err_code = int32(msg_client_message.E_ERR_CAT_NOT_FOUND)
		return
	}

	cat_table_id, _ := player.db.Cats.GetCfgId(cat_id)
	coin_ability, _ := player.db.Cats.GetCoinAbility(cat_id)
	match_ability, _ := player.db.Cats.GetMatchAbility(cat_id)
	explore_ability, _ := player.db.Cats.GetExploreAbility(cat_id)
	ouqi := player.db.Cats.CalcOuqi(cat_id)

	var response = msg_rpc_message.G2GSpaceCatResponse{
		PlayerId:       to_player_id,
		CatId:          cat_id,
		CatTableId:     cat_table_id,
		CoinAbility:    coin_ability,
		MatchAbility:   match_ability,
		ExploreAbility: explore_ability,
		Ouqi:           ouqi,
	}

	resp_data, err = _marshal_msg(&response)
	if err != nil {
		err_code = -1
		return
	}

	err_code = 1
	return
}

func (this *Player) space_cat(player_id, cat_id int32) int32 {
	if this.Id == player_id {
		return -1
	}

	var response = msg_client_message.S2CSpaceCatDataResponse{
		PlayerId: player_id,
		CatId:    cat_id,
	}
	p := player_mgr.GetPlayerById(player_id)
	if p != nil {
		if !p.db.Cats.HasIndex(cat_id) {
			log.Error("Player %v have not cat %v", this.Id, cat_id)
			return int32(msg_client_message.E_ERR_CAT_NOT_FOUND)
		}
		response.CatTableId, _ = p.db.Cats.GetCfgId(cat_id)
		response.CoinAbility, _ = p.db.Cats.GetCoinAbility(cat_id)
		response.MatchAbility, _ = p.db.Cats.GetMatchAbility(cat_id)
		response.ExploreAbility, _ = p.db.Cats.GetExploreAbility(cat_id)
		response.CatOuqi = p.db.Cats.CalcOuqi(cat_id)
	} else {
		resp, err_code := remote_space_cat(this.Id, player_id, cat_id)
		if err_code < 0 {
			return err_code
		}
		response.CatTableId = resp.CatTableId
		response.CoinAbility = resp.CoinAbility
		response.MatchAbility = resp.MatchAbility
		response.ExploreAbility = resp.ExploreAbility
		response.CatOuqi = resp.Ouqi
	}
	this.Send(uint16(msg_client_message.S2CSpaceCatDataResponse_ProtoID), &response)
	log.Trace("Player %v get player %v space cat %v data %v", this.Id, player_id, cat_id, response)
	return 1
}

func C2SFocusDataHandler(p *Player, msg_data []byte) int32 {
	var req msg_client_message.C2SFocusDataRequest
	err := proto.Unmarshal(msg_data, &req)
	if err != nil {
		log.Error("unmarshal msg failed %v", err.Error())
		return -1
	}
	return p.send_focus_data()
}

func C2SFocusPlayerHandler(p *Player, msg_data []byte) int32 {
	var req msg_client_message.C2SFocusPlayerRequest
	err := proto.Unmarshal(msg_data, &req)
	if err != nil {
		log.Error("unmarshal msg failed %v", err.Error())
		return -1
	}
	return p.focus_player(req.GetPlayerId())
}

func C2SUnfocusPlayerHandler(p *Player, msg_data []byte) int32 {
	var req msg_client_message.C2SFocusPlayerCancalRequest
	err := proto.Unmarshal(msg_data, &req)
	if err != nil {
		log.Error("unmarshal msg failed %v", err.Error())
		return -1
	}
	return p.unfocus_player(req.GetPlayerId())
}

func C2SMyPictureDataHandler(p *Player, msg_data []byte) int32 {
	var req msg_client_message.C2SMyPictureDataRequest
	err := proto.Unmarshal(msg_data, &req)
	if err != nil {
		log.Error("unmarshal msg failed %v", err.Error())
		return -1
	}
	return p.send_my_picture_data()
}

func C2SMyPictureSetHandler(p *Player, msg_data []byte) int32 {
	var req msg_client_message.C2SMyPictureSetRequest
	err := proto.Unmarshal(msg_data, &req)
	if err != nil {
		log.Error("unmarshal msg failed %v", err.Error())
		return -1
	}
	return p.my_picture_set(req.PictureData.GetCatId(), req.PictureData.GetPos())
}

func C2SSpaceDataHandler(p *Player, msg_data []byte) int32 {
	var req msg_client_message.C2SSpaceDataRequest
	err := proto.Unmarshal(msg_data, &req)
	if err != nil {
		log.Error("unmarshal msg failed %v", err.Error())
		return -1
	}
	return p.space_data(req.GetPlayerId())
}

func C2SSpaceCatHandler(p *Player, msg_data []byte) int32 {
	var req msg_client_message.C2SSpaceCatDataRequest
	err := proto.Unmarshal(msg_data, &req)
	if err != nil {
		log.Error("unmarshal msg failed %v", err.Error())
		return -1
	}
	return p.space_cat(req.GetPlayerId(), req.GetCatId())
}
