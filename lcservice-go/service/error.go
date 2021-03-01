package service

import "fmt"

const (
	errNotFound             = "NOT_FOUND"
	errMissingCommandName   = "MISSING_COMMAND_NAME"
	errBadArgumentValueType = "BAD_ARGUMENT_VALUE_TYPE"
	errUnsupportedType      = "UNSUPPORTED_TYPE"
)

type serviceError struct {
	code string
	data Dict
}

func newServiceError(code string) *serviceError {
	return &serviceError{code: code}
}

func (e *serviceError) Error() string {
	errStr := fmt.Sprintf("lcservice-go: %s", e.code)
	if e.data != nil {
		errStr = fmt.Sprintf("%s - context: %v", errStr, e.data)
	}
	return errStr
}

func (e *serviceError) withData(data Dict) {
	e.data = data
}
