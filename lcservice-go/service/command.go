package service

import (
	"sort"
	"strconv"
)

type CommandNamespace = string
type CommandName = string

var commandAllowedRequestParamTypes = map[string]struct{}{
	RequestParamTypeString: {},
	RequestParamTypeEnum:   {},
	RequestParamTypeInt:    {},
	RequestParamTypeBool:   {},
	RequestParamTypeFlag:   {},
}

type command struct {
	namespace CommandNamespace
	name      CommandName
	args      []commandArgument
}

type commandArgument interface {
	isRequired() bool
	parse(data Dict) error
}

type argumentTV struct {
	key string
	t   string

	IsRequired bool
	enumValues []string
}

func (atv argumentTV) isRequired() bool {
	return atv.IsRequired
}

func (atv argumentTV) isValid(data Dict) error {
	value, found := data[atv.key]
	if !found && atv.IsRequired {
		err := newServiceError(errNotFound)
		err.withData(Dict{
			"isRequired": atv.IsRequired,
			"key":        atv.key,
		})
		return err
	}
	if !found && !atv.IsRequired {
		return nil
	}

	switch atv.t {
	case RequestParamTypeFlag:
		if value != "" {
			err := newServiceError(errBadArgumentValueType)
			err.withData(Dict{
				"message":  "a flag does not accept value",
				"expected": "<nil>",
			})
			return err
		}
		return nil
	case RequestParamTypeBool:
		_, parseErr := strconv.ParseBool(value.(string))
		if parseErr == nil {
			return nil
		}
		err := newServiceError(errBadArgumentValueType)
		err.withData(Dict{
			"expectedType": RequestParamTypeBool,
			"actual":       value,
		})
		return err
	case RequestParamTypeEnum:
		for _, v := range atv.enumValues {
			if v == value {
				return nil
			}
		}
		err := newServiceError(errNotFound)
		err.withData(Dict{
			"expected": atv.enumValues,
			"actual":   value,
		})
		return err
	case RequestParamTypeString:
		_, ok := value.(string)
		if ok {
			return nil
		}
		err := newServiceError(errBadArgumentValueType)
		err.withData(Dict{
			"actual":       value,
			"expectedType": RequestParamTypeString,
		})
		return err
	case RequestParamTypeInt:
		_, parseErr := strconv.ParseInt(value.(string), 10, 32)
		if parseErr == nil {
			return nil
		}
		err := newServiceError(errBadArgumentValueType)
		err.withData(Dict{
			"actual":       value,
			"expectedType": RequestParamTypeInt,
		})
		return err
	}

	err := newServiceError(errUnsupportedType)
	supportedTypes := make([]string, 0, len(commandAllowedRequestParamTypes))
	for k := range commandAllowedRequestParamTypes {
		supportedTypes = append(supportedTypes, k)
	}
	sort.Strings(supportedTypes)
	err.withData(Dict{
		"actual":   atv.t,
		"expected": supportedTypes,
	})
	return err
}

type CommandsDescriptor struct {
	commands []command
}
