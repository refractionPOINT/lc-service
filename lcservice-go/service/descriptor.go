package service

import (
	"errors"
	"fmt"
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

type RequestEvent struct {
	Type string
	ID   string
	Data Dict
}

type Response struct {
	IsSuccess   bool   `json:"success" msgpack:"success"`
	IsRetriable bool   `json:"retry,omitempty" msgpack:"retry,omitempty"`
	Error       string `json:"error,omitempty" msgpack:"error,omitempty"`
	Data        Dict   `json:"data" msgpack:"data"`
	Jobs        []*Job `json:"jobs,omitempty" msgpack:"jobs,omitempty"`
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
	Type        RequestParamType `json:"type" msgpack:"type"`
	Description string           `json:"desc" msgpack:"desc"`
	IsRequired  bool             `json:"is_required" msgpack:"is_required"`

	// Only for "enum" Type
	Values []string `json:"values,omitempty" msgpack:"values,omitempty"`
}
type RequestParams = map[RequestParamName]RequestParamDef

func (r *RequestParamDef) isValid() error {
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

func requestParamsIsValid(params RequestParams) error {
	for paramName, paramDef := range params {
		if paramName == "" {
			return fmt.Errorf("parameter name is empty")
		}
		if err := paramDef.isValid(); err != nil {
			return err
		}
	}
	return nil
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

	// Commands
	Commands CommandsDescriptor
}

// Optional callbacks available.
type DescriptorCallbacks struct {
	OnOrgInstall   ServiceCallback `json:"org_install" msgpack:"org_install"`
	OnOrgUninstall ServiceCallback `json:"org_uninstall" msgpack:"org_uninstall"`

	OnDetection       ServiceCallback `json:"detection" msgpack:"detection"`
	OnRequest         ServiceCallback `json:"request" msgpack:"request"`
	OnGetResource     ServiceCallback `json:"get_resource" msgpack:"get_resource"`
	OnDeploymentEvent ServiceCallback `json:"deployment_event" msgpack:"deployment_event"`
	OnLogEvent        ServiceCallback `json:"log_event" msgpack:"log_event"`

	// Called once per Org per X time.
	OnOrgPer1H  ServiceCallback `json:"org_per_1h" msgpack:"org_per_1h"`
	OnOrgPer3H  ServiceCallback `json:"org_per_3h" msgpack:"org_per_3h"`
	OnOrgPer12H ServiceCallback `json:"org_per_12h" msgpack:"org_per_12h"`
	OnOrgPer24H ServiceCallback `json:"org_per_24h" msgpack:"org_per_24h"`
	OnOrgPer7D  ServiceCallback `json:"org_per_7d" msgpack:"org_per_7d"`
	OnOrgPer30D ServiceCallback `json:"org_per_30d" msgpack:"org_per_30d"`

	// Called once per X time.
	OnOncePer1H  ServiceCallback `json:"once_per_1h" msgpack:"once_per_1h"`
	OnOncePer3H  ServiceCallback `json:"once_per_3h" msgpack:"once_per_3h"`
	OnOncePer12H ServiceCallback `json:"once_per_12h" msgpack:"once_per_12h"`
	OnOncePer24H ServiceCallback `json:"once_per_24h" msgpack:"once_per_24h"`
	OnOncePer7D  ServiceCallback `json:"once_per_7d" msgpack:"once_per_7d"`
	OnOncePer30D ServiceCallback `json:"once_per_30d" msgpack:"once_per_30d"`

	// Called once per sensor per X time.
	OnSensorPer1H  ServiceCallback `json:"sensor_per_1h" msgpack:"sensor_per_1h"`
	OnSensorPer3H  ServiceCallback `json:"sensor_per_3h" msgpack:"sensor_per_3h"`
	OnSensorPer12H ServiceCallback `json:"sensor_per_12h" msgpack:"sensor_per_12h"`
	OnSensorPer24H ServiceCallback `json:"sensor_per_24h" msgpack:"sensor_per_24h"`
	OnSensorPer7D  ServiceCallback `json:"sensor_per_7d" msgpack:"sensor_per_7d"`
	OnSensorPer30D ServiceCallback `json:"sensor_per_30d" msgpack:"sensor_per_30d"`

	OnNewSensor ServiceCallback `json:"new_sensor" msgpack:"new_sensor"`

	OnServiceError ServiceCallback `json:"service_error" msgpack:"service_error"`
}

func (d Descriptor) IsValid() error {
	commandNames := map[string]struct{}{}
	for _, command := range d.Commands.Descriptors {
		if command.Name == "" {
			return errors.New("command name cannot be empty")
		}
		if command.Description == "" {
			return fmt.Errorf("command '%s' description is empty", command.Name)
		}
		if err := requestParamsIsValid(command.Args); err != nil {
			return err
		}
		if _, ok := commandNames[command.Name]; ok {
			return fmt.Errorf("command %s implemented more than once", command.Name)
		}
		commandNames[command.Name] = struct{}{}
		if command.Handler == nil {
			return fmt.Errorf("command %s has a nil handler", command.Name)
		}
	}
	return nil
}
