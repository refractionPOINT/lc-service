package service

type CommandsDescriptor struct {
	Descriptors []CommandDescriptor `json:"commands"`
}

type CommandName = string
type CommandParams = RequestParams
type CommandDescriptor struct {
	Name        CommandName     `json:"name"`
	Description string          `json:"desc"`
	Args        CommandParams   `json:"args"`
	Handler     ServiceCallback `json:"-"`
}
