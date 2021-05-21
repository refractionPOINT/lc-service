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
	"github.com/refractionPOINT/lc-service/lcservice-go/common"
	"github.com/refractionPOINT/lc-service/lcservice-go/service/resolver/command"
	"github.com/refractionPOINT/lc-service/lcservice-go/service/resolver/request"
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
	GetType() string
	Parse(requestEvent RequestEvent) (Dict, error)
	Get(requestEvent RequestEvent) ServiceCallback
	PreHandlerHook(request Request) error
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
	cs.Log(fmt.Sprintf("Processing started for '%s' => %+v", resolver.GetType(), data))

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
		OID:      req.OID,
		Deadline: deadline,
		Event: RequestEvent{
			Type: req.Type,
			ID:   req.MsgID,
			Data: req.Data,
		},
	}
	var err error
	parsedData, err := resolver.Parse(serviceRequest.Event)
	if err != nil {
		cs.LogError(err.Error())
		return NewErrorResponse(err)
	}
	serviceRequest.Event.Data = parsedData

	handler := resolver.Get(serviceRequest.Event)
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

	if err := resolver.PreHandlerHook(serviceRequest); err != nil {
		return NewErrorResponse(err)
	}

	// Send it.
	resp := handler(serviceRequest)
	if cs.desc.IsDebug {
		cs.desc.Log(fmt.Sprintf("REQ (%s) result: err(%s)", req.MsgID, resp.Error))
	}
	return resp
}

func (cs *CoreService) ProcessCommand(data Dict) Response {
	resolver := command.NewService(cs.desc, cs)
	return cs.processGenericRequest(data, &resolver)
}

func (cs *CoreService) ProcessRequest(data Dict) Response {
	resolver := request.NewService(cs)
	return cs.processGenericRequest(data, &resolver)
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

func (cs *CoreService) GetHandler(reqType string) (common.ServiceCallback, bool) {
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
