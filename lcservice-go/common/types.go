package common

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

type ServiceCallback = func(Request) Response
