package service

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"reflect"
	"sort"
	"sync/atomic"
	"time"

	lc "github.com/refractionPOINT/go-limacharlie/limacharlie"
)

const (
	PROTOCOL_VERSION = 1
)

type coreService struct {
	desc Descriptor

	callsInProgress uint32
	startedAt       int64

	cbMap map[string]ServiceCallback
}

type lcRequest struct {
	Version  int    `json:"version"`
	JWT      string `json:"jwt"`
	OID      string `json:"oid"`
	MsgID    string `json:"mid"`
	Deadline int64  `json:"deadline"`
	Type     string `json:"etype"`
	Data     Dict   `json:"data"`
}

var ErrNotImplemented = NewErrorResponse("not implemented")

func NewService(descriptor Descriptor) (*coreService, error) {
	cs := &coreService{
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

func (cs *coreService) Init() error {
	return nil
}

func (cs *coreService) ProcessCommand(data Dict, sig string) (interface{}, bool) {
	atomic.AddUint32(&cs.callsInProgress, 1)
	defer func() {
		atomic.AddUint32(&cs.callsInProgress, ^uint32(0))
	}()
	// Validate the HMAC signature.
	if !cs.verifyOrigin(data, sig) {
		// This is the only special case where
		// we return isAccepted = false to tell
		// the parent that the signature is
		// specifically invalid.
		return nil, false
	}

	// Parse the request format.
	req := lcRequest{}
	if err := DictToStruct(data, &req); err != nil {
		return Response{
			Error: fmt.Sprintf("invalid format: %v", err),
		}, true
	}

	// Check we can work with this version of the protocol.
	if req.Version > PROTOCOL_VERSION {
		return Response{
			Data: Dict{"error": fmt.Sprintf("unsupported version (> %d)", PROTOCOL_VERSION)},
		}, true
	}

	if cs.desc.IsDebug {
		cs.desc.Log(fmt.Sprintf("REQ (%s): %s => %+v", req.MsgID, req.Type, req.Data))
	}

	// Check if we're still within the deadline.
	deadline := time.Time{}
	if req.Deadline != 0 {
		deadline := time.Unix(req.Deadline, 0)
		if time.Now().After(deadline) {
			return NewErrorResponse("deadline exceeded"), true
		}
	}

	desc, found := cs.getCommandDescriptor(req.Type)
	if !found {
		return ErrNotImplemented, true
	}

	// do args validation
	parseState := desc.parse(req.Data)
	if parseState.err != nil {
		return Response{
			Error: parseState.err.Error(),
		}, false
	}

	request := Request{
		OID:      req.OID,
		Deadline: deadline,
		Event: RequestEvent{
			Type: req.Type,
			ID:   req.MsgID,
			Data: parseState.args,
		},
	}
	resp := desc.handler(request)
	return resp, true
}

func (cs *coreService) ProcessRequest(data Dict, sig string) (response interface{}, isAccepted bool) {
	atomic.AddUint32(&cs.callsInProgress, 1)
	defer func() {
		atomic.AddUint32(&cs.callsInProgress, ^uint32(0))
	}()
	// Validate the HMAC signature.
	var err error
	if !cs.verifyOrigin(data, sig) {
		// This is the only special case where
		// we return isAccepted = false to tell
		// the parent that the signature is
		// specifically invalid.
		return nil, false
	}

	// Parse the request format.
	req := lcRequest{}
	if err := DictToStruct(data, &req); err != nil {
		return Response{
			Error: fmt.Sprintf("invalid format: %v", err),
		}, true
	}

	// Check we can work with this version of the protocol.
	if req.Version > PROTOCOL_VERSION {
		return Response{
			Data: Dict{"error": fmt.Sprintf("unsupported version (> %d)", PROTOCOL_VERSION)},
		}, true
	}

	if cs.desc.IsDebug {
		cs.desc.Log(fmt.Sprintf("REQ (%s): %s => %+v", req.MsgID, req.Type, req.Data))
	}

	// Check if we're still within the deadline.
	deadline := time.Time{}
	if req.Deadline != 0 {
		deadline := time.Unix(req.Deadline, 0)
		if time.Now().After(deadline) {
			return NewErrorResponse("deadline exceeded"), true
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

	// Unlike the Python implementation, we will not perform validation
	// of the incoming parameters based on the schema in the Descriptor.
	// Instead we will leave that task to the user by using `DictToStruct`
	// to facilitate Marshaling and validation.
	// TODO revisit this, maybe we can at least validate part of it.

	// Get the relevant handler.
	handler, ok := cs.getHandler(req.Type)
	if !ok {
		return ErrNotImplemented, true
	}

	// Create an SDK instance.
	if serviceRequest.Org, err = lc.NewOrganizationFromClientOptions(lc.ClientOptions{
		OID: req.OID,
		JWT: req.JWT,
	}, cs); err != nil {
		return NewErrorResponse(err.Error()), true
	}

	// Send it.
	resp := handler(serviceRequest)

	return resp, true
}

func (cs *coreService) verifyOrigin(data Dict, sig string) bool {
	d, err := json.Marshal(data)
	if err != nil {
		cs.desc.LogCritical(fmt.Sprintf("verifyOrigin.json.Marshal: %v", err))
		return false
	}
	mac := hmac.New(sha256.New, []byte(cs.desc.SecretKey))
	if _, err := mac.Write(d); err != nil {
		cs.desc.LogCritical(fmt.Sprintf("verifyOrigin.hmac.Write: %v", err))
		return false
	}
	expected := mac.Sum(nil)
	return hmac.Equal([]byte(hex.EncodeToString(expected)), []byte(sig))
}

func (cs *coreService) getHandler(reqType string) (ServiceCallback, bool) {
	cb, ok := cs.cbMap[reqType]
	return cb, ok
}

func (cs *coreService) getCommandDescriptor(reqType string) (commandDescriptor, bool) {
	for _, commandHandler := range cs.desc.Commands.descriptors {
		if reqType == commandHandler.name {
			return commandHandler, true
		}
	}
	return commandDescriptor{}, false
}

func (cs *coreService) cbHealth(r Request) Response {
	cbSupported := []string{}
	for k := range cs.cbMap {
		cbSupported = append(cbSupported, k)
	}
	sort.StringSlice(cbSupported).Sort()
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
			},
		},
	}
}

func (cs *coreService) buildCallbackMap() map[string]ServiceCallback {
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
func (cs coreService) Fatal(msg string) {
	if cs.desc.LogCritical == nil {
		return
	}
	cs.desc.LogCritical(msg)
}
func (cs coreService) Error(msg string) {
	if cs.desc.LogCritical == nil {
		return
	}
	cs.desc.LogCritical(msg)
}
func (cs coreService) Warn(msg string) {
	if cs.desc.LogCritical == nil {
		return
	}
	cs.desc.LogCritical(msg)
}
func (cs coreService) Info(msg string) {
	if cs.desc.Log == nil {
		return
	}
	cs.desc.Log(msg)
}
func (cs coreService) Debug(msg string) {
	if cs.desc.Log == nil {
		return
	}
	cs.desc.Log(msg)
}
func (cs coreService) Trace(msg string) {
	if cs.desc.Log == nil {
		return
	}
	cs.desc.Log(msg)
}
