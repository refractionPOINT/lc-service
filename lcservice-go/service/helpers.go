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

func StructToDict(s interface{}, d *Dict) error {
	b, err := json.Marshal(s)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(b, d); err != nil {
		return err
	}
	return nil
}

func NewErrorResponse(err error) Response {
	return Response{Error: err.Error()}
}

func NewRetriableResponse(err error) Response {
	resp := NewErrorResponse(err)
	resp.IsRetriable = true
	return resp
}
