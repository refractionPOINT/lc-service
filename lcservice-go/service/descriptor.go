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
type RequestParamType = string
type RequestParamDef struct {
	Type        RequestParamType `json:"type"`
	Description string           `json:"desc"`
	IsRequired  bool             `json:"is_required"`

	// Only for "enum" Type
	Values []string `json:"values"`
}
type RequestParams = map[RequestParamName]RequestParamDef

const (
	RequestParamTypeString = "str"
	RequestParamTypeEnum   = "enum"
	RequestParamTypeInt    = "int"
	RequestParamTypeBool   = "bool"
)

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

	// commands
	commands commandsDescriptor
}

// Optional callbacks available.
type DescriptorCallbacks struct {
	OnOrgInstall   ServiceCallback `json:"org_install"`
	OnOrgUninstall ServiceCallback `json:"org_uninstall"`

	OnDetection       ServiceCallback `json:"detection"`
	OnRequest         ServiceCallback `json:"request"`
	OnGetResource     ServiceCallback `json:"get_resource"`
	OnDeploymentEvent ServiceCallback `json:"deployment_event"`
	OnLogEvent        ServiceCallback `json:"log_event"`

	// Called once per Org per X time.
	OnOrgPer1H  ServiceCallback `json:"org_per_1h"`
	OnOrgPer3H  ServiceCallback `json:"org_per_3h"`
	OnOrgPer12H ServiceCallback `json:"org_per_12h"`
	OnOrgPer24H ServiceCallback `json:"org_per_24h"`
	OnOrgPer7D  ServiceCallback `json:"org_per_7d"`
	OnOrgPer30D ServiceCallback `json:"org_per_30d"`

	// Called once per X time.
	OnOncePer1H  ServiceCallback `json:"once_per_1h"`
	OnOncePer3H  ServiceCallback `json:"once_per_3h"`
	OnOncePer12H ServiceCallback `json:"once_per_12h"`
	OnOncePer24H ServiceCallback `json:"once_per_24h"`
	OnOncePer7D  ServiceCallback `json:"once_per_7d"`
	OnOncePer30D ServiceCallback `json:"once_per_30d"`

	// Called once per sensor per X time.
	OnSensorPer1H  ServiceCallback `json:"sensor_per_1h"`
	OnSensorPer3H  ServiceCallback `json:"sensor_per_3h"`
	OnSensorPer12H ServiceCallback `json:"sensor_per_12h"`
	OnSensorPer24H ServiceCallback `json:"sensor_per_24h"`
	OnSensorPer7D  ServiceCallback `json:"sensor_per_7d"`
	OnSensorPer30D ServiceCallback `json:"sensor_per_30d"`

	OnNewSensor ServiceCallback `json:"new_sensor"`

	OnServiceError ServiceCallback `json:"service_error"`
}
