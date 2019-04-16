package main

import (
	//"mm_server/libs/log"
	//"mm_server/libs/socket"
	"mm_server/proto/gen_go/client_message"
	//"github.com/golang/protobuf/proto"
)

func (this *Player) guide_data() {
	response := &msg_client_message.S2CGuideDataResponse{
		Data: this.db.GuideData.GetData(),
	}
	this.Send(uint16(msg_client_message.S2CGuideDataResponse_ProtoID), response)
}
