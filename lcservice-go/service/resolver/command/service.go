package command

import (
	"fmt"

	lc "github.com/refractionPOINT/go-limacharlie/limacharlie"
	"github.com/refractionPOINT/lc-service/lcservice-go/common"
)

type Logger interface {
	Log(log string)
}

type CommandsDescriptorsGetter interface {
	GetCommandsDescriptor() common.CommandsDescriptor
}

type commandHandlerResolver struct {
	cmdsDescriptorsGetter CommandsDescriptorsGetter
	logger                Logger
}

func NewService(cmdsDescriptorsGetter CommandsDescriptorsGetter, logger Logger) commandHandlerResolver {
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
	commandName, found := requestEvent.Data["command_name"]
	if !found {
		c.logger.Log("command_name not found in data")
		return nil
	}
	c.logger.Log(fmt.Sprintf("looking for handler for '%s'", commandName))

	for _, commandHandler := range c.cmdsDescriptorsGetter.GetCommandsDescriptor().Descriptors {
		if commandName == commandHandler.Name {
			return commandHandler.Handler
		}
	}
	c.logger.Log(fmt.Sprintf("no handler found for '%s'", commandName))
	return nil
}

func (r *commandHandlerResolver) PreHandlerHook(request common.Request) error {
	rid, err := request.GetRoomID()
	if err != nil {
		return err
	}
	cid, err := request.GetCommandID()
	if err != nil {
		return err
	}

	// Test compat, ignore if no SDK.
	if request.Org == nil {
		return nil
	}

	if _, err := request.Org.Comms().Room(rid).Post(lc.NewMessage{
		Type: lc.CommsMessageTypes.CommandAck,
		Content: common.Dict{
			"cid": cid,
		},
	}); err != nil {
		return err
	}
	return nil
}
