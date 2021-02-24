package service

type CommandNamespace = string
type CommandName = string

type Command interface {
	GetNamespace() CommandNamespace
	GetName() CommandName
	GetRequestParams() RequestParams
	IsValid() bool
}
