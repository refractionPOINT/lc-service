package common

import (
	"fmt"
	"time"

	lc "github.com/refractionPOINT/go-limacharlie/limacharlie"
)

type RequestEvent struct {
	Type string
	ID   string
	Data Dict
}


func (re RequestEvent) AsResourceRequest() (ResourceRequest, error) {
	rr := ResourceRequest{}
	srr := singleResourceRequest{}
	mrr := multiResourceRequest{}
	if err := DictToStruct(re.Data, &srr); err == nil {
		rr.inIncludeData = srr.IsWithData
		rr.isSingleRes = true
		rr.ResourceNames = []string{srr.Name}
	} else if err := DictToStruct(re.Data, &mrr); err == nil {
		rr.inIncludeData = mrr.IsWithData
		rr.isSingleRes = false
		rr.ResourceNames = mrr.Names
	}
	return rr, nil
}

// Input/Output from Service callbacks.
type Request struct {
	Org      *lc.Organization
	OID      string
	Deadline time.Time
	Event    RequestEvent
}

func (r Request) GetRoomID() (string, error) {
	eventRID, found := r.Event.Data["rid"]
	if !found {
		return "", fmt.Errorf("missing rid (roomID)")
	}
	rid, ok := eventRID.(string)
	if !ok {
		return "", fmt.Errorf("rid is not a string")
	}
	return rid, nil
}

func (r Request) GetCommandID() (string, error) {
	eventCID, found := r.Event.Data["cid"]
	if !found {
		return "", fmt.Errorf("missing cid (commandID)")
	}
	cid, ok := eventCID.(string)
	if !ok {
		return "", fmt.Errorf("cid is not a string")
	}
	return cid, nil
}

func (r Request) GetSessionID() (string, error) {
	eventSSID, found := r.Event.Data["ssid"]
	if !found {
		return "", fmt.Errorf("missing ssid (sessionID)")
	}
	ssid, ok := eventSSID.(string)
	if !ok {
		return "", fmt.Errorf("ssid is not a string")
	}
	return ssid, nil
}

var RequestParamTypes = struct {
	String RequestParamType
	Enum   RequestParamType
	Int    RequestParamType
	Bool   RequestParamType
	UUID   RequestParamType
}{
	String: "str",
	Enum:   "enum",
	Int:    "int",
	Bool:   "bool",
	UUID:   "uuid",
}

var SupportedRequestParamTypes = map[string]struct{}{
	RequestParamTypes.String: {},
	RequestParamTypes.Enum:   {},
	RequestParamTypes.Int:    {},
	RequestParamTypes.Bool:   {},
	RequestParamTypes.UUID:   {},
}

// LimaCharlie Service Request formats.
// These parameter definitions are only used
// to provide the LimaCharlie cloud with an
// expected list of parameters. Actual validation
// should be done at runtime. You may use the
// helper function `DictToStruct` for this purpose.
type RequestParamName = string
type RequestParamType = string
type RequestParamDef struct {
	Type        RequestParamType `json:"type" msgpack:"type"`
	Description string           `json:"desc" msgpack:"desc"`
	IsRequired  bool             `json:"is_required" msgpack:"is_required"`

	// Only for "enum" Type
	Values []string `json:"values,omitempty" msgpack:"values,omitempty"`
}

func (r *RequestParamDef) IsValid() error {
	if r.Description == "" {
		return fmt.Errorf("parameter description is empty")
	}
	if _, ok := SupportedRequestParamTypes[r.Type]; !ok {
		return fmt.Errorf("parameter type '%v' is not supported (%v)", r.Type, SupportedRequestParamTypes)
	}
	if r.Type == RequestParamTypes.Enum && len(r.Values) == 0 {
		return fmt.Errorf("parameter type is enum but no values provided")
	}
	if r.Type != RequestParamTypes.Enum && len(r.Values) != 0 {
		return fmt.Errorf("paramter type is not enum but has values provided")
	}
	return nil
}

type RequestParams = map[RequestParamName]RequestParamDef
