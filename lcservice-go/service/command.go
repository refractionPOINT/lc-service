package service

type CommandNamespace = string
type CommandName = string
type Command struct {
	Namespace     CommandNamespace `json:"namespace"`
	Name          CommandName      `json:"name"`
	RequestParams RequestParams    `json:"params"`
}

func NewCommand(namespace CommandNamespace) *Command {
	return &Command{Namespace: namespace}
}

func (c *Command) WithName(name CommandName) *Command {
	c.Name = name
	return c
}

func (c *Command) WithParam(name RequestParamName, desc string, paramType RequestParamType, required bool, values []string) *Command {
	paramValues := []string{}
	if paramType == RequestParamTypeEnum {
		paramValues = values
	}

	c.RequestParams[name] = RequestParamDef{
		Type:        paramType,
		Description: desc,
		IsRequired:  required,
		Values:      paramValues,
	}
	return c
}

func (c *Command) WithParamEnumOptional(name RequestParamName, desc string, values []string) *Command {
	return c.WithParam(name, desc, RequestParamTypeEnum, false, values)
}

func (c *Command) WithParamEnumRequired(name RequestParamName, desc string, values []string) *Command {
	return c.WithParam(name, desc, RequestParamTypeEnum, true, values)
}

func (c *Command) WithParamOptional(name RequestParamName, desc string, paramType RequestParamType) *Command {
	return c.WithParam(name, desc, paramType, false, nil)
}

func (c *Command) WithParamRequired(name RequestParamName, desc string, paramType RequestParamType) *Command {
	return c.WithParam(name, desc, paramType, true, nil)
}
