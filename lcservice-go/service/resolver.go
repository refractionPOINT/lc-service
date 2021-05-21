package service

import (
	"fmt"

	lc "github.com/refractionPOINT/go-limacharlie/limacharlie"
)

type requestHandlerResolver struct {
	cs *CoreService
}

func (r *requestHandlerResolver) getType() string {
	return "request"
}

func (r *requestHandlerResolver) parse(requestEvent RequestEvent) (Dict, error) {
	return requestEvent.Data, nil
}

func (r *requestHandlerResolver) get(requestEvent RequestEvent) ServiceCallback {
	// Unlike the Python implementation, we will not perform validation
	// of the incoming parameters based on the schema in the Descriptor.
	// Instead we will leave that task to the user by using `DictToStruct`
	// to facilitate Marshaling and validation.
	// TODO revisit this, maybe we can at least validate part of it.

	// Get the relevant handler.
	handler, found := r.cs.getHandler(requestEvent.Type)
	if !found {
		return nil
	}
	return handler
}

func (r *requestHandlerResolver) preHandlerHook(request Request) error {
	return nil
}

type Logger interface {
	Log(str string)
}

type commandHandlerResolver struct {
	commandsDesc *CommandsDescriptor
	desc         *Descriptor
}

func (r *commandHandlerResolver) getType() string {
	return "command"
}

func (c *commandHandlerResolver) parse(requestEvent RequestEvent) (Dict, error) {
	// TODO here we might want to
	// 1. filter request argument that we want to send to the command handler
	// 2. revalidate what we received
	return requestEvent.Data, nil
}

func (c *commandHandlerResolver) get(requestEvent RequestEvent) ServiceCallback {
	commandName, found := requestEvent.Data["command_name"]
	if !found {
		c.desc.Log("command_name not found in data")
		return nil
	}
	c.desc.Log(fmt.Sprintf("looking for handler for '%s'", commandName))

	for _, commandHandler := range c.commandsDesc.Descriptors {
		if commandName == commandHandler.Name {
			return commandHandler.Handler
		}
	}
	c.desc.Log(fmt.Sprintf("no handler found for '%s'", commandName))
	return nil
}

func (r *commandHandlerResolver) preHandlerHook(request Request) error {
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
		Content: Dict{
			"cid": cid,
		},
	}); err != nil {
		return err
	}
	return nil
}
