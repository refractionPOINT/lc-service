package service

type CommandsDescriptor struct {
	Descriptors []CommandDescriptor `json:"commands"`
}

var commandAllowedRequestParamTypes = map[string]struct{}{
	RequestParamTypeString: {},
	RequestParamTypeEnum:   {},
	RequestParamTypeInt:    {},
	RequestParamTypeBool:   {},
}

type CommandName = string
type CommandDescriptor struct {
	Name    CommandName     `json:"name"`
	Args    Dict            `json:"args"`
	Handler ServiceCallback `json:"-"`
}
