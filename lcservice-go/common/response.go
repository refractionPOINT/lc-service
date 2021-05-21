package common

type Response struct {
	IsSuccess   bool   `json:"success" msgpack:"success"`
	IsRetriable bool   `json:"retry,omitempty" msgpack:"retry,omitempty"`
	Error       string `json:"error,omitempty" msgpack:"error,omitempty"`
	Data        Dict   `json:"data" msgpack:"data"`
	Jobs        []*Job `json:"jobs,omitempty" msgpack:"jobs,omitempty"`
}
