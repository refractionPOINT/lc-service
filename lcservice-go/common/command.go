package common

type CommandsDescriptor struct {
	Descriptors []CommandDescriptor `json:"commands" msgpack:"commands"`
}

type CommandName = string
type CommandParams = RequestParams
type CommandDescriptor struct {
	Name        CommandName     `json:"name" msgpack:"name"`
	Description string          `json:"desc" msgpack:"desc"`
	Args        CommandParams   `json:"args" msgpack:"args"`
	Handler     ServiceCallback `json:"-" msgpack:"-"`
}
