package service

type commandsDescriptor struct {
	Descriptors []commandDescriptor `json:"commands"`
}

var commandAllowedRequestParamTypes = map[string]struct{}{
	RequestParamTypeString: {},
	RequestParamTypeEnum:   {},
	RequestParamTypeInt:    {},
	RequestParamTypeBool:   {},
}

type CommandName = string
type commandDescriptor struct {
	Name    CommandName `json:"name"`
	Args    Dict        `json:"args"`
	handler ServiceCallback
}
