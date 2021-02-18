package service

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	lc "github.com/refractionPOINT/go-limacharlie/limacharlie"
)

const (
	PROTOCOL_VERSION = 1
)

type coreService struct {
	desc Descriptor
}

type lcRequest struct {
	Version  int                    `json:"version"`
	JWT      string                 `json:"jwt"`
	OID      string                 `json:"oid"`
	MsgID    string                 `json:"mid"`
	Deadline int64                  `json:"deadline"`
	Type     string                 `json:"etype"`
	Data     map[string]interface{} `json:"data"`
}

var ErrNotImplemented = NewErrorResponse("not implemented")

func NewService(descriptor Descriptor) (*coreService, error) {
	cs := &coreService{
		desc: descriptor,
	}
	return cs, nil
}

func (cs *coreService) Init() error {
	return nil
}

func (cs *coreService) ProcessRequest(data map[string]interface{}, sig string) (response interface{}, isAccepted bool) {
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
			Data: map[string]interface{}{"error": fmt.Sprintf("unsupported version (> %s)", PROTOCOL_VERSION)},
		}, true
	}

	if cs.desc.IsDebug {
		cs.desc.Log(fmt.Sprintf("REQ (%s): %s => %+v", req.MsgID, req.Type, req.Data))
	}

	// Check if we're still within the deadline.
	deadline := time.Unix(req.Deadline, 0)
	if time.Now().After(deadline) {
		return NewErrorResponse("deadline exceeded"), true
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
	if serviceRequest.Org, err = lc.NewOrganization(lc.ClientOptions{
		OID: req.OID,
		JWT: req.JWT,
	}); err != nil {
		return NewErrorResponse(err.Error()), true
	}

	// Send it.
	resp := handler(serviceRequest)

	return resp, true
}

func (cs *coreService) verifyOrigin(data map[string]interface{}, sig string) bool {
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
	switch reqType {
	case "health":
		return cs.cbHealth, true
	case "org_install":
		return cs.desc.OnOrgInstall, cs.desc.OnOrgInstall != nil
	case "org_uninstall":
		return cs.desc.OnOrgUninstall, cs.desc.OnOrgUninstall != nil
	case "request":
		return cs.desc.OnRequest, cs.desc.OnRequest != nil
	default:
		return nil, false
	}
}

func (cs *coreService) cbHealth(r Request) Response {
	return Response{}
}
