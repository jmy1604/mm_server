package main

import (
	"mm_server/libs/log"
	"mm_server/proto/gen_go/client_message"

	"github.com/golang/protobuf/proto"
)

func (this *Player) load_surface_data() int32 {
	var msg msg_client_message.S2CSurfaceDataResponse
	data := this.db.Surface.GetData()
	err := proto.Unmarshal(data, &msg)
	if err != nil {
		log.Error("Player %v load surface data error %v", err.Error())
		return -1
	}

	for _, d := range msg.GetData() {
		if this.surface_data == nil {
			this.surface_data = make(map[int32]map[int32]int32)
		}
		x := d.GetX()
		y := d.GetY()
		if this.surface_data[x] == nil {
			this.surface_data[x] = make(map[int32]int32)
		}
		this.surface_data[x][y] = d.GetCfgId()
	}

	return 1
}

func (this *Player) save_surface_data() int32 {
	this.surface_data_locker.RLock()
	defer this.surface_data_locker.RUnlock()

	if this.surface_data == nil {
		return 0
	}
	var msg msg_client_message.S2CSurfaceDataResponse
	for x, d := range this.surface_data {
		if d == nil {
			continue
		}
		for y, bid := range d {
			msg.Data = append(msg.Data, &msg_client_message.BuildingInfo{
				CfgId: bid,
				X:     x,
				Y:     y,
			})
		}
	}
	data, err := proto.Marshal(&msg)
	if err != nil {
		log.Error("Player %v save surface err %v", this.Id, err.Error())
		return -1
	}
	this.db.Surface.SetData(data)
	return 1
}

func (this *Player) get_surface_data() []*msg_client_message.BuildingInfo {
	this.surface_data_locker.RLock()
	defer this.surface_data_locker.RUnlock()

	var data []*msg_client_message.BuildingInfo
	if this.surface_data != nil {
		for x, d := range this.surface_data {
			if d == nil {
				continue
			}
			for y, bid := range d {
				data = append(data, &msg_client_message.BuildingInfo{
					CfgId: bid,
					X:     x,
					Y:     y,
				})
			}

		}
	}

	return data
}

func (this *Player) send_surface_data() int32 {
	data := this.get_surface_data()
	this.Send(uint16(msg_client_message.S2CSurfaceDataResponse_ProtoID), &msg_client_message.S2CSurfaceDataResponse{
		Data: data,
	})
	log.Trace("Player %v get surface data %v", this.Id, data)
	return 1
}

func (this *Player) surface_update(update_data, remove_data []*msg_client_message.BuildingInfo) int32 {
	this.surface_data_locker.Lock()

	var updated bool
	// 更新的地板
	for _, d := range update_data {
		if this.surface_data == nil {
			this.surface_data = make(map[int32]map[int32]int32)
		}
		x := d.GetX()
		if this.surface_data[x] == nil {
			this.surface_data[x] = make(map[int32]int32)
		}
		y := d.GetY()
		v := this.surface_data[x][y]
		if v != d.GetCfgId() {
			this.surface_data[x][y] = d.GetCfgId()
			updated = true
		}
	}
	// 删除的地板
	for _, d := range remove_data {
		x := d.GetX()
		if this.surface_data[x] != nil {
			y := d.GetY()
			delete(this.surface_data[x], y)
			updated = true
		}
	}

	this.surface_data_locker.Unlock()

	// 保存
	if updated {
		res := this.save_surface_data()
		if res < 0 {
			return res
		}
	}
	this.Send(uint16(msg_client_message.S2CSurfaceUpdateResponse_ProtoID), &msg_client_message.S2CSurfaceUpdateResponse{})
	log.Trace("Player %v updated surface", this.Id)
	return 1
}

func C2SSurfaceDataHandler(p *Player, msg_data []byte) int32 {
	var req msg_client_message.C2SSurfaceDataRequest
	err := proto.Unmarshal(msg_data, &req)
	if err != nil {
		log.Error("unmarshal msg failed %v", err.Error())
		return -1
	}
	return p.send_surface_data()
}

func C2SSurfaceUpdateHandler(p *Player, msg_data []byte) int32 {
	var req msg_client_message.C2SSurfaceUpdateRequest
	err := proto.Unmarshal(msg_data, &req)
	if err != nil {
		log.Error("unmarshal msg failed %v", err.Error())
		return -1
	}
	return p.surface_update(req.GetUpdateData(), req.GetRemoveData())
}
