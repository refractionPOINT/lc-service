package service

import "encoding/json"

type Dict = map[string]interface{}

func DictToStruct(d Dict, s interface{}) error {
	b, err := json.Marshal(d)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(b, s); err != nil {
		return err
	}
	return nil
}

func NewErrorResponse(err string) Response {
	return Response{Data: Dict{"error": err}}
}

func NewRetriableResponse(err string) Response {
	return Response{IsRetriable: true, Data: Dict{"error": err}}
}
