package service

import "encoding/json"

func DictToStruct(d map[string]interface{}, s interface{}) error {
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
	return Response{Data: map[string]interface{}{"error": err}}
}

func NewRetriableResponse(err string) Response {
	return Response{IsRetriable: true, Data: map[string]interface{}{"error": err}}
}
