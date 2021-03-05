package service

type CommandsDescriptor struct {
	Descriptors []commandDescriptor `json:"commands"`
}

var commandAllowedRequestParamTypes = map[string]struct{}{
	RequestParamTypeString: {},
	RequestParamTypeEnum:   {},
	RequestParamTypeInt:    {},
	RequestParamTypeBool:   {},
}

type CommandNamespace = string
type CommandName = string
type commandDescriptor struct {
	Name    CommandName `json:"name"`
	Args    Dict        `json:"args"`
	handler ServiceCallback
}
