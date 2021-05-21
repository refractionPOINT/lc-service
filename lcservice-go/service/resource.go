package service

import (
	"github.com/refractionPOINT/lc-service/lcservice-go/common"
)

type ResourceRequest = common.RequestEvent
type ResourceResponse = common.ResourceResponse

func NewResourceFromData(category string, data []byte) *ResourceResponse {
	return common.NewResourceFromData(category, data)
}
