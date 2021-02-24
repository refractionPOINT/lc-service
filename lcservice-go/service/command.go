package service

type CommandNamespace string
type CommandName string
type Command struct {
	Namespace     CommandNamespace `json:"namespace"`
	Name          CommandName      `json:"name"`
	RequestParams RequestParams    `json:"params"`
}

func NewCommand(namespace CommandNamespace) *Command {
	return &Command{Namespace: namespace}
}
