package service

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCommandArgumentIsRequired(t *testing.T) {
	a := assert.New(t)
	arg := argumentTV{
		key:        "foo",
		IsRequired: false,
	}
	a.NoError(arg.isValid(Dict{"bar": ""}))

	expectedErr := newServiceError(errNotFound)
	expectedErr.withData(Dict{
		"isRequired": true,
		"key":        "foo",
	})
	arg.IsRequired = true
	a.EqualError(arg.isValid(Dict{"bar": ""}), expectedErr.Error())

	arg.t = "qwert"
	expectedErr = newServiceError(errUnsupportedType)
	supportedTypes := make([]string, 0, len(commandAllowedRequestParamTypes))
	for k := range commandAllowedRequestParamTypes {
		supportedTypes = append(supportedTypes, k)
	}
	expectedErr.withData(Dict{
		"actual":   "qwert",
		"expected": []string{"bool", "enum", "flag", "int", "str"},
	})
	a.EqualError(arg.isValid(Dict{"foo": ":D"}), expectedErr.Error())
}

func TestCommandArgumentTypeValueBool(t *testing.T) {
	a := assert.New(t)
	arg := argumentTV{
		key: "foo",
		t:   RequestParamTypeBool,
	}
	a.NoError(arg.isValid(Dict{"foo": "1"}))
	a.NoError(arg.isValid(Dict{"foo": "False"}))
	a.NoError(arg.isValid(Dict{"foo": "true"}))
}

func TestCommandArgumentTypeEnum(t *testing.T) {
	a := assert.New(t)
	arg := argumentTV{
		key:        "foo",
		t:          RequestParamTypeEnum,
		enumValues: []string{"VALUE_0", "2", "VALUE_3"},
	}
	a.NoError(arg.isValid(Dict{"foo": "VALUE_0"}))
	a.NoError(arg.isValid(Dict{"foo": "VALUE_3"}))
	a.NoError(arg.isValid(Dict{"foo": "2"}))

	expectedErr := newServiceError(errNotFound)
	expectedErr.withData(Dict{
		"expected": []string{"VALUE_0", "2", "VALUE_3"},
		"actual":   "2",
	})
	a.EqualError(arg.isValid(Dict{"foo": 2}), expectedErr.Error())

	expectedErr = newServiceError(errNotFound)
	expectedErr.withData(Dict{
		"expected": []string{"VALUE_0", "2", "VALUE_3"},
		"actual":   "VALUE_1",
	})
	a.EqualError(arg.isValid(Dict{"foo": "VALUE_1"}), expectedErr.Error())
}

func TestCommandArgumentTypeFlag(t *testing.T) {
	a := assert.New(t)
	arg := argumentTV{
		key: "foo",
		t:   "flag",
	}

	expectedErr := newServiceError(errBadArgumentValueType)
	expectedErr.withData(Dict{
		"message":  "a flag does not accept value",
		"expected": "<nil>",
	})
	a.EqualError(arg.isValid(Dict{"foo": 2}), expectedErr.Error())
	a.NoError(arg.isValid(Dict{"foo": ""}))
}

func TestCommandArgumentTypeString(t *testing.T) {
	a := assert.New(t)
	arg := argumentTV{
		key: "foo",
		t:   "str",
	}
	a.NoError(arg.isValid(Dict{"foo": "1"}))
	a.NoError(arg.isValid(Dict{"foo": "bar"}))

	expectedErr := newServiceError(errBadArgumentValueType)
	expectedErr.withData(Dict{
		"actual":       1,
		"expectedType": "str",
	})
	a.EqualError(arg.isValid(Dict{"foo": 1}), expectedErr.Error())
}

func TestCommandArgumentTypeInt(t *testing.T) {
	a := assert.New(t)
	arg := argumentTV{
		key: "foo",
		t:   "int",
	}
	a.NoError(arg.isValid(Dict{"foo": "1"}))
	a.NoError(arg.isValid(Dict{"foo": "0"}))

	expectedErr := newServiceError(errBadArgumentValueType)
	expectedErr.withData(Dict{
		"actual":       "bar",
		"expectedType": "int",
	})
	a.EqualError(arg.isValid(Dict{"foo": "bar"}), expectedErr.Error())
}
