package service

import (
	"errors"
	"fmt"
)

type CommandsDescriptor struct {
	Descriptors []CommandDescriptor `json:"commands" msgpack:"commands"`
}

func (d CommandsDescriptor) isValid() error {
	commandNames := map[string]struct{}{}
	for _, command := range d.Descriptors {
		if err := command.isValid(); err != nil {
			return err
		}
		if _, ok := commandNames[command.Name]; ok {
			return fmt.Errorf("command %s implemented more than once", command.Name)
		}
		commandNames[command.Name] = struct{}{}
	}
	return nil
}

type CommandName = string
type CommandParams = RequestParams
type CommandDescriptor struct {
	Name        CommandName     `json:"name" msgpack:"name"`
	Description string          `json:"desc" msgpack:"desc"`
	Args        CommandParams   `json:"args" msgpack:"args"`
	Handler     ServiceCallback `json:"-" msgpack:"-"`
}

func (d CommandDescriptor) isValid() error {
	if d.Name == "" {
		return errors.New("command name cannot be empty")
	}
	if d.Description == "" {
		return fmt.Errorf("command '%s' description is empty", d.Name)
	}
	if err := requestParamsIsValid(d.Args); err != nil {
		return err
	}
	if d.Handler == nil {
		return fmt.Errorf("command %s has a nil handler", d.Name)
	}
	return nil
}
