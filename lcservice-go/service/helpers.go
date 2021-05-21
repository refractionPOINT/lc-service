package service

import (
	"github.com/refractionPOINT/lc-service/lcservice-go/common"
)

type Dict = common.Dict

func DictToStruct(d Dict, s interface{}) error {
	return common.DictToStruct(d, s)
}

func NewErrorResponse(err error) Response {
	return Response{Error: err.Error()}
}

func NewRetriableResponse(err error) Response {
	resp := NewErrorResponse(err)
	resp.IsRetriable = true
	return resp
}
