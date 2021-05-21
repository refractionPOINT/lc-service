package command

import (
	"fmt"

	"github.com/refractionPOINT/lc-service/lcservice-go/common"
	"github.com/refractionPOINT/lc-service/lcservice-go/service/acker"
)

type Logger interface {
	Log(log string)
}

type CommandsDescriptorsGetter interface {
	GetCommandDescriptors() []common.CommandDescriptor
}

type CommandAcker interface {
	Ack(req common.Request) error
}

type commandHandlerResolver struct {
	cmdsDescriptorsGetter CommandsDescriptorsGetter
	logger                Logger
}

func NewResolver(cmdsDescriptorsGetter CommandsDescriptorsGetter, logger Logger) commandHandlerResolver {
	return commandHandlerResolver{
		cmdsDescriptorsGetter: cmdsDescriptorsGetter,
		logger:                logger,
	}
}

func (r *commandHandlerResolver) GetType() string {
	return "command"
}

func (c *commandHandlerResolver) Parse(requestEvent common.RequestEvent) (common.Dict, error) {
	// TODO here we might want to
	// 1. filter request argument that we want to send to the command handler
	// 2. revalidate what we received
	return requestEvent.Data, nil
}

func (c *commandHandlerResolver) Get(requestEvent common.RequestEvent) common.ServiceCallback {
	commandName, err := requestEvent.GetCommandName()
	if err != nil {
		c.logger.Log(fmt.Sprintf("command_name: %v", err))
		return nil
	}
	c.logger.Log(fmt.Sprintf("looking for handler for '%s'", commandName))

	for _, commandHandler := range c.cmdsDescriptorsGetter.GetCommandDescriptors() {
		if commandName == commandHandler.Name {
			return commandHandler.Handler
		}
	}
	c.logger.Log(fmt.Sprintf("no handler found for '%s'", commandName))
	return nil
}

func (r *commandHandlerResolver) PreHandlerHook(request common.Request, reqAcker acker.RequestAcker) error {
	// Test compat, ignore if no SDK.
	if request.Org == nil {
		return nil
	}

	return reqAcker.Ack(request)
}
