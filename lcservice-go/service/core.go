package service

import (
	"bytes"
	"fmt"
	"math"
	"reflect"
	"sort"
	"sync/atomic"
	"time"

	lc "github.com/refractionPOINT/go-limacharlie/limacharlie"
)

const (
	PROTOCOL_VERSION = 1
)

type CoreService struct {
	desc Descriptor

	callsInProgress uint32
	startedAt       int64

	cbMap map[string]ServiceCallback
}

type lcRequest struct {
	Version  int     `json:"version" msgpack:"version"`
	JWT      string  `json:"jwt" msgpack:"jwt"`
	OID      string  `json:"oid" msgpack:"oid"`
	MsgID    string  `json:"mid" msgpack:"mid"`
	Deadline float64 `json:"deadline" msgpack:"deadline"`
	Type     string  `json:"etype" msgpack:"etype"`
	Data     Dict    `json:"data" msgpack:"data"`
}

func NewService(descriptor Descriptor) (*CoreService, error) {
	if err := descriptor.IsValid(); err != nil {
		return nil, err
	}

	cs := &CoreService{
		desc:      descriptor,
		startedAt: time.Now().Unix(),
	}
	// Initialize some of the values we prefer to be ready.
	if cs.desc.DetectionsSubscribed == nil {
		cs.desc.DetectionsSubscribed = []string{}
	}
	if cs.desc.RequestParameters == nil {
		cs.desc.RequestParameters = map[string]RequestParamDef{}
	}
	cs.cbMap = cs.buildCallbackMap()

	return cs, nil
}

func (cs *CoreService) Init() error {
	return nil
}

func (cs *CoreService) GetSecretKey() []byte {
	return []byte(cs.desc.SecretKey)
}

type handlerResolver interface {
	getType() string
	parse(requestEvent RequestEvent) (Dict, error)
	get(requestEvent RequestEvent) ServiceCallback
	preHandlerHook(request Request) (string, error)
}

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

func (r *requestHandlerResolver) preHandlerHook(request Request) (string, error) {
	return "", nil
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
		if c.desc.IsDebug {
			c.desc.Log("command_name not found in data")
		}
		return nil
	}
	if c.desc.IsDebug {
		c.desc.Log(fmt.Sprintf("looking for handler for '%s'", commandName))
	}
	for _, commandHandler := range c.commandsDesc.Descriptors {
		if commandName == commandHandler.Name {
			return commandHandler.Handler
		}
	}
	if c.desc.IsDebug {
		c.desc.Log(fmt.Sprintf("no handler found for '%s'", commandName))
	}
	return nil
}

func (r *commandHandlerResolver) preHandlerHook(request Request) (string, error) {
	rid, err := request.GetRoomID()
	if err != nil {
		return "", err
	}

	// Test compat, ignore if no SDK.
	if request.Org == nil {
		return "", nil
	}

	mid, err := request.Org.Comms().Room(rid).Post(lc.NewMessage{
		Type:    lc.CommsMessageTypes.CommandAck,
		Content: request.Event.Data,
	})
	if err != nil {
		return "", err
	}
	return mid, nil
}

func (cs *CoreService) Log(log string) {
	if cs.desc.IsDebug {
		cs.desc.Log(log)
	}
}

func (cs *CoreService) LogError(errStr string) {
	if cs.desc.IsDebug {
		cs.desc.LogCritical(errStr)
	}
}

func (cs *CoreService) processGenericRequest(data Dict, resolver handlerResolver) Response {
	atomic.AddUint32(&cs.callsInProgress, 1)
	defer func() {
		atomic.AddUint32(&cs.callsInProgress, ^uint32(0))
	}()

	// Parse the request format.
	req := lcRequest{}
	if err := DictToStruct(data, &req); err != nil {
		return NewErrorResponse(fmt.Errorf("invalid format: %v", err))
	}

	// Check we can work with this version of the protocol.
	if req.Version > PROTOCOL_VERSION {
		return NewErrorResponse(fmt.Errorf("unsupported version (> %d)", PROTOCOL_VERSION))
	}

	if cs.desc.IsDebug {
		cs.desc.Log(fmt.Sprintf("REQ (%s): %s => %+v", req.MsgID, req.Type, req.Data))
	}

	// Check if we're still within the deadline.
	deadline := time.Time{}
	if req.Deadline != 0 {
		deadline := time.Unix(int64(math.Trunc(req.Deadline)), 0)
		if time.Now().After(deadline) {
			cs.LogError("deadline exceeded")
			return NewErrorResponse(fmt.Errorf("deadline exceeded"))
		}
	}

	serviceRequest := Request{
		Refs:     RequestRefs{},
		OID:      req.OID,
		Deadline: deadline,
		Event: RequestEvent{
			Type: req.Type,
			ID:   req.MsgID,
			Data: req.Data,
		},
	}
	var err error
	parsedData, err := resolver.parse(serviceRequest.Event)
	if err != nil {
		cs.LogError(err.Error())
		return NewErrorResponse(err)
	}
	serviceRequest.Event.Data = parsedData

	handler := resolver.get(serviceRequest.Event)
	if handler == nil {
		cs.LogError(fmt.Sprintf("resolver not implemented for '%s'", serviceRequest.Event))
		return NewErrorResponse(fmt.Errorf("not implemented"))
	}

	// health request will not be providing a jwt - if you want an org provide an oid and a jwt
	if req.OID != "" && req.JWT != "" {
		// Create an SDK instance.
		if serviceRequest.Org, err = lc.NewOrganizationFromClientOptions(lc.ClientOptions{
			OID: req.OID,
			JWT: req.JWT,
		}, cs); err != nil {
			cs.LogError(err.Error())
			return NewErrorResponse(err)
		}
	}

	ackMid, err := resolver.preHandlerHook(serviceRequest)
	if err != nil {
		return NewErrorResponse(err)
	}
	if ackMid != "" {
		serviceRequest.Refs.AckMID = ackMid
	}

	// Send it.
	resp := handler(serviceRequest)
	if cs.desc.IsDebug {
		cs.desc.Log(fmt.Sprintf("REQ (%s) result: err(%s)", req.MsgID, resp.Error))
	}
	return resp
}

func (cs *CoreService) ProcessCommand(data Dict) Response {
	return cs.processGenericRequest(data, &commandHandlerResolver{commandsDesc: &cs.desc.Commands, desc: &cs.desc})
}

func (cs *CoreService) ProcessRequest(data Dict) Response {
	return cs.processGenericRequest(data, &requestHandlerResolver{cs: cs})
}

func lcCompatibleJSONMarshal(d []byte) []byte {
	/*
		dataIn: {"key0":{},"key1":42.24,"key2":"value2","jwt":null}
		compat: {"key0": {}, "key1": 42.24, "key2": "value2", "jwt": null}
	*/
	// replace '":' -> '": '
	res := bytes.ReplaceAll(d, []byte(`":`), []byte(`": `))
	// replace ',"' -> ', "'
	res = bytes.ReplaceAll(res, []byte(`,"`), []byte(`, "`))
	return res
}

func (cs *CoreService) getHandler(reqType string) (ServiceCallback, bool) {
	cb, ok := cs.cbMap[reqType]
	return cb, ok
}

func (cs *CoreService) cbHealth(r Request) Response {
	cbSupported := []string{}
	for k := range cs.cbMap {
		cbSupported = append(cbSupported, k)
	}
	sort.StringSlice(cbSupported).Sort()

	commandsSupported := make(map[string]CommandDescriptor, len(cs.desc.Commands.Descriptors))
	for _, commandDescriptor := range cs.desc.Commands.Descriptors {
		commandsSupported[commandDescriptor.Name] = commandDescriptor
	}

	return Response{
		IsSuccess: true,
		Data: Dict{
			"version":           PROTOCOL_VERSION,
			"start_time":        cs.startedAt,
			"calls_in_progress": cs.callsInProgress,
			"mtd": Dict{
				"detect_subscriptions": cs.desc.DetectionsSubscribed,
				"callbacks":            cbSupported,
				"request_params":       cs.desc.RequestParameters,
				"commands":             commandsSupported,
			},
		},
	}
}

func (cs *CoreService) buildCallbackMap() map[string]ServiceCallback {
	cb := cs.desc.Callbacks
	t := reflect.TypeOf(cb)

	// Already include some static callbacks provided
	// by the coreService.
	cbMap := map[string]ServiceCallback{
		"health": cs.cbHealth,
	}

	for i := 0; i < t.NumField(); i++ {
		v := reflect.ValueOf(cb).Field(i)
		if v.IsNil() {
			continue
		}
		f := t.Field(i)
		cbName, ok := f.Tag.Lookup("json")
		if !ok {
			panic("callback with unknown name")
		}
		cbMap[cbName] = v.Interface().(ServiceCallback)
	}
	return cbMap
}

// LC.Logger Interface Compatibility
func (cs CoreService) Fatal(msg string) {
	if cs.desc.LogCritical == nil {
		return
	}
	cs.desc.LogCritical(msg)
}
func (cs CoreService) Error(msg string) {
	if cs.desc.LogCritical == nil {
		return
	}
	cs.desc.LogCritical(msg)
}
func (cs CoreService) Warn(msg string) {
	if cs.desc.LogCritical == nil {
		return
	}
	cs.desc.LogCritical(msg)
}
func (cs CoreService) Info(msg string) {
	if cs.desc.Log == nil {
		return
	}
	cs.desc.Log(msg)
}
func (cs CoreService) Debug(msg string) {
	if cs.desc.Log == nil {
		return
	}
	cs.desc.Log(msg)
}
func (cs CoreService) Trace(msg string) {
	if cs.desc.Log == nil {
		return
	}
	cs.desc.Log(msg)
}
