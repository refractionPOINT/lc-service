package command

import (
	"github.com/refractionPOINT/lc-service/lcservice-go/common"

	lc "github.com/refractionPOINT/go-limacharlie/limacharlie"
)

type requestAcker struct{}

func NewRequestAcker() requestAcker {
	return requestAcker{}
}

func (r requestAcker) Ack(req common.Request) error {
	rid, err := req.GetRoomID()
	if err != nil {
		return err
	}
	cid, err := req.GetCommandID()
	if err != nil {
		return err
	}

	if _, err := req.Org.Comms().Room(rid).Post(lc.NewMessage{
		Type: lc.CommsMessageTypes.CommandAck,
		Content: common.Dict{
			"cid": cid,
		},
	}); err != nil {
		return err
	}
	return nil
}
