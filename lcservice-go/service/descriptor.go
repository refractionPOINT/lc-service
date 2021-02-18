package service

import (
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
	Data map[string]interface{}
}

type Response struct {
	IsSuccess   bool                   `json:"success"`
	IsRetriable bool                   `json:"retry,omitempty"`
	Error       string                 `json:"error,omitempty"`
	Data        map[string]interface{} `json:"data"`
	Jobs        []*Job                 `json:"jobs,omitempty"`
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

	// General purpose
	Log         func(msg string)
	LogCritical func(msg string)

	// Callbacks
	OnOrgInstall   ServiceCallback
	OnOrgUninstall ServiceCallback
	OnRequest      ServiceCallback
}
