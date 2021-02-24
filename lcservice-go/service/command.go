package service

import (
	"fmt"
)

type CommandNamespace = string
type CommandName = string
type Command struct {
	Namespace     CommandNamespace `json:"namespace"`
	Name          CommandName      `json:"name"`
	RequestParams RequestParams    `json:"params"`
}

func NewCommand(namespace CommandNamespace) (*Command, error) {
	return &Command{Namespace: namespace}, nil
}

func (c *Command) WithName(name CommandName) error {
	c.Name = name
	return nil
}

func (c *Command) AddParam(name RequestParamName, desc string, paramType RequestParamType, required bool, values []string) error {
	if _, found := allowedRequestParamTypes[paramType]; !found {
		return fmt.Errorf("unsupported request parameter type %s", paramType)
	}
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
	return nil
}

func (c *Command) AddParamEnumOptional(name RequestParamName, desc string, values []string) error {
	return c.AddParam(name, desc, RequestParamTypeEnum, false, values)
}

func (c *Command) AddParamEnumRequired(name RequestParamName, desc string, values []string) error {
	return c.AddParam(name, desc, RequestParamTypeEnum, true, values)
}

func (c *Command) AddParamOptional(name RequestParamName, desc string, paramType RequestParamType) error {
	return c.AddParam(name, desc, paramType, false, nil)
}

func (c *Command) AddParamRequired(name RequestParamName, desc string, paramType RequestParamType) error {
	return c.AddParam(name, desc, paramType, true, nil)
}
