package acker

import "github.com/refractionPOINT/lc-service/lcservice-go/common"

type RequestAcker interface {
	Ack(req common.Request) error
}

type NoopAcker struct{}

func (a NoopAcker) Ack(req common.Request) error {
	return nil
}
