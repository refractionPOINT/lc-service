package service

import (
	"reflect"
	"time"

	lc "github.com/refractionPOINT/go-limacharlie/limacharlie"
)

// Input/Output from Service callbacks.
type Request struct {
	Org      *lc.Organization
	OID      string
	Deadline time.Time
	Event    RequestEvent
}

type RequestEvent struct {
	Type string
	ID   string
	Data Dict
}

type Response struct {
	IsSuccess   bool   `json:"success"`
	IsRetriable bool   `json:"retry,omitempty"`
	Error       string `json:"error,omitempty"`
	Data        Dict   `json:"data"`
	Jobs        []*Job `json:"jobs,omitempty"`
}

type ServiceCallback = func(Request) Response

// LimaCharlie Service Request formats.
// These parameter definitions are only used
// to provide the LimaCharlie cloud with an
// expected list of parameters. Actual validation
// should be done at runtime. You may use the
// helper function `DictToStruct` for this purpose.
type RequestParamName = string
type RequestParamDef struct {
	Type        string `json:"type"`
	Description string `json:"desc"`
	IsRequired  bool   `json:"is_required"`

	// Only for "enum" Type
	Values []string `json:"values"`
}
type RequestParams = map[RequestParamName]RequestParamDef

type Descriptor struct {
	// Basic info
	Name      string
	SecretKey string
	IsDebug   bool

	// Supported requests
	RequestParameters map[RequestParamName]RequestParamDef

	// Detections to subscribe to
	DetectionsSubscribed []string

	// General purpose
	Log         func(msg string)
	LogCritical func(msg string)

	// Callbacks
	Callbacks DescriptorCallbacks
}

type DescriptorCallbacks struct {
	OnOrgInstall   ServiceCallback `json:"org_install"`
	OnOrgUninstall ServiceCallback `json:"org_uninstall"`
	OnRequest      ServiceCallback `json:"request"`
}

func (cb DescriptorCallbacks) getSupported() []string {
	t := reflect.TypeOf(cb)

	// Already include some static callbacks provided
	// by the coreService.
	names := []string{
		"health",
	}

	for i := 0; i < t.NumField(); i++ {
		if reflect.ValueOf(cb).Field(i).IsNil() {
			continue
		}
		f := t.Field(i)
		cbName, ok := f.Tag.Lookup("json")
		if !ok {
			panic("callback with unknown name")
		}
		names = append(names, cbName)
	}
	return names
}
