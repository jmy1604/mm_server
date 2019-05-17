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
		BeFocusNum: this.db.SpaceCommon.GetBeFocusNum(),
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

	player.db.SpaceCommon.IncbyBeFocusNum(1)

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
		return int32(msg_client_message.E_ERR_SPACE_CANT_FOCUS_SELF)
	}
	if this.db.FocusPlayers.HasIndex(player_id) {
		log.Error("Player %v already focus player %v", this.Id, player_id)
		return int32(msg_client_message.E_ERR_SPACE_ALREADY_FOCUSED_PLAYER)
	}

	var name string
	var level, head int32
	p := player_mgr.GetPlayerById(player_id)
	if p != nil {
		p.db.SpaceCommon.IncbyBeFocusNum(1)
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

	player.db.SpaceCommon.IncbyBeFocusNum(-1)

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
		return int32(msg_client_message.E_ERR_SPACE_NOT_FOCUS_PLAYER)
	}
	p := player_mgr.GetPlayerById(player_id)
	if p != nil {
		p.db.SpaceCommon.IncbyBeFocusNum(-1)
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

func (this *Player) get_my_pics() []int32 {
	var my_pics []int32
	cat_ids := this.db.MyPictureDatas.GetAllIndex()
	if cat_ids != nil {
		for _, cat_id := range cat_ids {
			my_pics = append(my_pics, cat_id)
		}
	}
	return my_pics
}

func (this *Player) send_my_picture_data() int32 {
	response := &msg_client_message.S2CMyPictureDataResponse{
		CatIds: this.get_my_pics(),
	}
	this.Send(uint16(msg_client_message.S2CMyPictureDataResponse_ProtoID), response)
	log.Trace("Player %v get my pictures data %v", this.Id, response)
	return 1
}

const (
	MY_PICTURE_NUM = 9
)

func (this *Player) my_picture_set(cat_id int32, is_cancel bool) int32 {
	if !this.db.Cats.HasIndex(cat_id) {
		log.Error("Player %v have no cat %v", this.Id, cat_id)
		return int32(msg_client_message.E_ERR_CAT_NOT_FOUND)
	}

	has_pic := this.db.MyPictureDatas.HasIndex(cat_id)
	if has_pic {
		if is_cancel {
			this.db.MyPictureDatas.Remove(cat_id)
		} else {
			log.Error("Player %v already set cat_id %v in picture", this.Id, cat_id)
			return -1
		}
	} else {
		cat_ids := this.db.MyPictureDatas.GetAllIndex()
		if cat_ids != nil {
			if len(cat_ids) >= MY_PICTURE_NUM {
				log.Error("Player %v only set %v picture", this.Id, MY_PICTURE_NUM)
				return int32(msg_client_message.E_ERR_SPACE_ALREADY_FULL)
			}
		}
		this.db.MyPictureDatas.Add(&dbPlayerMyPictureDataData{
			CatId: cat_id,
		})
	}
	response := &msg_client_message.S2CMyPictureSetResponse{
		CatId:    cat_id,
		IsCancel: is_cancel,
	}
	this.Send(uint16(msg_client_message.S2CMyPictureSetResponse_ProtoID), response)
	log.Trace("Player %v set my picture cat_id(%v) is_cancel(%v)", this.Id, cat_id, is_cancel)
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

	var cat_name string
	var cat_table_id, coin_ability, match_ability, explore_ability, ouqi int32
	cat_ids := player.db.MyPictureDatas.GetAllIndex()
	var cats []*msg_rpc_message.SpaceCatData
	for _, cat_id := range cat_ids {
		cat_name, cat_table_id, coin_ability, match_ability, explore_ability, ouqi, err_code = player.get_space_cat_data(cat_id)
		if err_code < 0 {
			return
		}
		cats = append(cats, &msg_rpc_message.SpaceCatData{
			CatId:          cat_id,
			CatTableId:     cat_table_id,
			CatName:        cat_name,
			CatOuqi:        ouqi,
			CoinAbility:    coin_ability,
			MatchAbility:   match_ability,
			ExploreAbility: explore_ability,
		})
	}
	var response = msg_rpc_message.G2GSpaceDataResponse{
		PlayerName:  player.db.GetName(),
		PlayerLevel: player.db.GetLevel(),
		PlayerHead:  player.db.Info.GetHead(),
		Zan:         player.db.Info.GetZan(),
		BeFocusNum:  player.db.SpaceCommon.GetBeFocusNum(),
		Charm:       player.db.Info.GetCharmVal(),
		Cats:        cats,
		Gender:      player.db.SpaceCommon.GetGender(),
		FashionIds:  player.db.SpaceCommon.GetFashionIds(),
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
	var level, head, zan, charm, be_focus_num, gender int32
	var fashion_ids []int32
	var cats []*msg_client_message.SpaceCatData
	p := player_mgr.GetPlayerById(player_id)
	if p != nil {
		name = p.db.GetName()
		level = p.db.GetLevel()
		head = p.db.Info.GetHead()
		zan = p.db.Info.GetZan()
		charm = p.db.Info.GetCharmVal()
		be_focus_num = p.db.SpaceCommon.GetBeFocusNum()
		player_pics := p.get_my_pics()
		for _, cid := range player_pics {
			cat_table_id, _ := p.db.Cats.GetCfgId(cid)
			cat_name, _ := p.db.Cats.GetNick(cid)
			ouqi := p.db.Cats.CalcOuqi(cid)
			coin_ability, _ := p.db.Cats.GetCoinAbility(cid)
			match_ability, _ := p.db.Cats.GetMatchAbility(cid)
			explore_ability, _ := p.db.Cats.GetExploreAbility(cid)
			cats = append(cats, &msg_client_message.SpaceCatData{
				CatId:          cid,
				CatTableId:     cat_table_id,
				CatName:        cat_name,
				CatOuqi:        ouqi,
				CoinAbility:    coin_ability,
				MatchAbility:   match_ability,
				ExploreAbility: explore_ability,
			})
		}
		gender = p.db.SpaceCommon.GetGender()
		fashion_ids = p.db.SpaceCommon.GetFashionIds()
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
		if resp.Cats != nil {
			for _, c := range resp.Cats {
				cats = append(cats, &msg_client_message.SpaceCatData{
					CatId:          c.CatId,
					CatTableId:     c.CatTableId,
					CatName:        c.CatName,
					CatOuqi:        c.CatOuqi,
					CoinAbility:    c.CoinAbility,
					MatchAbility:   c.MatchAbility,
					ExploreAbility: c.ExploreAbility,
				})
			}
		}
		gender = resp.Gender
		fashion_ids = resp.FashionIds
	}

	response := &msg_client_message.S2CSpaceDataResponse{
		PlayerId:    player_id,
		PlayerName:  name,
		PlayerLevel: level,
		PlayerHead:  head,
		Zaned:       zan,
		Charm:       charm,
		BeFocusNum:  be_focus_num,
		Cats:        cats,
		Gender:      gender,
		FashionIds:  fashion_ids,
	}
	this.Send(uint16(msg_client_message.S2CSpaceDataResponse_ProtoID), response)
	log.Trace("Player %v get player %v space data %v", this.Id, player_id, response)
	return 1
}

func (this *Player) get_space_cat_data(cat_id int32) (cat_name string, cat_table_id, coin_ability, match_ability, explore_ability, ouqi, err_code int32) {
	if !this.db.Cats.HasIndex(cat_id) {
		err_code = int32(msg_client_message.E_ERR_CAT_NOT_FOUND)
		log.Error("Player %v have not cat %v", this.Id, cat_id)
		return
	}
	if !this.db.MyPictureDatas.HasIndex(cat_id) {
		err_code = int32(msg_client_message.E_ERR_SPACE_NOT_HAVE_CAT_PICTURE)
		log.Error("Player %v cat %v not have picture", this.Id, cat_id)
		return
	}
	cat_name, _ = this.db.Cats.GetNick(cat_id)
	cat_table_id, _ = this.db.Cats.GetCfgId(cat_id)
	coin_ability, _ = this.db.Cats.GetCoinAbility(cat_id)
	match_ability, _ = this.db.Cats.GetMatchAbility(cat_id)
	explore_ability, _ = this.db.Cats.GetExploreAbility(cat_id)
	ouqi = this.db.Cats.CalcOuqi(cat_id)
	return
}

func (this *Player) space_set_gender(gender int32) int32 {
	gen := this.db.SpaceCommon.GetGender()
	if gen > 0 {
		log.Error("Player %v space is already set gender", this.Id)
		return int32(msg_client_message.E_ERR_SPACE_ALREADY_SET_GENDER)
	}

	if gender != 1 && gender != 2 {
		log.Error("Player %v set gender type %v invalid", this.Id, gender)
		return -1
	}

	this.db.SpaceCommon.SetGender(gender)

	response := &msg_client_message.S2CSpaceSetGenderResponse{
		Gender: gender,
	}
	this.Send(uint16(msg_client_message.S2CSpaceSetGenderResponse_ProtoID), response)
	log.Trace("Player %v space set gender to %v", this.Id, gender)
	return 1
}

const (
	FASHION_EQUIP_TOTAL_TYPE = 4
)

func (this *Player) space_fashion_save(fashion_ids []int32) int32 {
	if fashion_ids == nil || len(fashion_ids) == 0 {
		this.db.SpaceCommon.SetFashionIds([]int32{})
	} else {
		gender := this.db.SpaceCommon.GetGender()
		var fids_map = make(map[int32]int32)
		for _, fid := range fashion_ids {
			fashion := item_table_mgr.Map[fid]
			if fashion == nil {
				log.Error("Player %v save fashion id %v not found", this.Id, fid)
				return int32(msg_client_message.E_ERR_SPACE_FASHION_TABLE_ID_NOT_FOUND)
			}
			if fashion.Gender != gender {
				log.Error("Player %v save fashion id %v not suitable to gender %v", this.Id, fid, gender)
				return int32(msg_client_message.E_ERR_SPACE_FASHION_GENDER_NOT_MATCH)
			}
			if fids_map[fid] > 0 {
				log.Error("Player %v save fashion ids %v has duplicate id %v", this.Id, fashion_ids, fid)
			}
			if this.GetItemResourceValue(fid) < 1 {
				log.Error("Player %v not found item %v to fashion", this.Id, fid)
				return int32(msg_client_message.E_ERR_ITEM_NOT_FOUND)
			}
			fids_map[fid] = fid
		}
		this.db.SpaceCommon.SetFashionIds(fashion_ids)
	}
	response := &msg_client_message.S2CSpaceFashionSaveResponse{
		FashionIds: fashion_ids,
	}
	this.Send(uint16(msg_client_message.S2CSpaceFashionSaveResponse_ProtoID), response)
	log.Trace("Player %v space fashion %v saved", this.Id, fashion_ids)
	return 1
}

func (this *Player) space_fashion_data() int32 {
	response := &msg_client_message.S2CSpaceFashionDataResponse{
		Gender:     this.db.SpaceCommon.GetGender(),
		FashionIds: this.db.SpaceCommon.GetFashionIds(),
	}
	this.Send(uint16(msg_client_message.S2CSpaceFashionDataResponse_ProtoID), response)
	log.Trace("Player %v space fashion data %v", this.Id, response)
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
	return p.my_picture_set(req.GetCatId(), req.GetIsCancel())
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

func C2SSpaceSetGenderHandler(p *Player, msg_data []byte) int32 {
	var req msg_client_message.C2SSpaceSetGenderRequest
	err := proto.Unmarshal(msg_data, &req)
	if err != nil {
		log.Error("unmarshal msg failed %v", err.Error())
		return -1
	}
	return p.space_set_gender(req.GetGender())
}

func C2SSpaceFashionSaveHandler(p *Player, msg_data []byte) int32 {
	var req msg_client_message.C2SSpaceFashionSaveRequest
	err := proto.Unmarshal(msg_data, &req)
	if err != nil {
		log.Error("unmarshal msg failed %v", err.Error())
		return -1
	}
	return p.space_fashion_save(req.FashionIds)
}

func C2SSpaceFashionDataHandler(p *Player, msg_data []byte) int32 {
	var req msg_client_message.C2SSpaceFashionDataRequest
	err := proto.Unmarshal(msg_data, &req)
	if err != nil {
		log.Error("unmarshal msg failed %v", err.Error())
		return -1
	}
	return p.space_fashion_data()
}
