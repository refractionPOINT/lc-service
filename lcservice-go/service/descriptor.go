package service

import (
	"fmt"
	"strconv"
	"time"

	"github.com/google/uuid"
	lc "github.com/refractionPOINT/go-limacharlie/limacharlie"
)

// Input/Output from Service callbacks.
type Request struct {
	Refs     RequestRefs
	Org      *lc.Organization
	OID      string
	Deadline time.Time
	Event    RequestEvent
}

func (r Request) Get(key string) (interface{}, error) {
	dataValue, found := r.Event.Data[key]
	if !found {
		return "", fmt.Errorf("key '%s' not found", key)
	}
	return dataValue, nil
}

func (r Request) GetString(key string) (string, error) {
	dataValue, err := r.Get(key)
	if err != nil {
		return "", err
	}
	value, ok := dataValue.(string)
	if !ok {
		return "", fmt.Errorf("key '%s' is not a string", key)
	}
	return value, nil
}

func (r Request) GetEnumValue(key string, requestParams RequestParams) (string, error) {
	paramDef, found := requestParams[key]
	if !found {
		return "", fmt.Errorf("key '%s' is not an expected parameter", key)
	}
	if paramDef.Type != RequestParamTypes.Enum {
		return "", fmt.Errorf("key '%s' is not of enum type", key)
	}
	enumValue, err := r.GetString(key)
	if err != nil {
		return "", err
	}

	for _, value := range paramDef.Values {
		if value == enumValue {
			return enumValue, nil
		}
	}
	return "", fmt.Errorf("value '%s' is not a valid enum value for key '%s'", enumValue, key)
}

func (r Request) GetInt(key string) (int, error) {
	dataValue, err := r.Get(key)
	if err != nil {
		return 0, err
	}
	value, ok := dataValue.(int)
	if !ok {
		return 0, fmt.Errorf("key '%s' is not an integer", key)
	}
	return value, nil
}

func (r Request) GetBool(key string) (bool, error) {
	dataValue, err := r.Get(key)
	if err != nil {
		return false, err
	}
	value, ok := dataValue.(bool)
	if ok {
		return value, nil
	}
	strValue, ok := dataValue.(string)
	if ok {
		if boolValue, err := strconv.ParseBool(strValue); err == nil {
			return boolValue, nil
		}
	}
	return false, fmt.Errorf("key '%s' is not a boolean", key)
}

func (r Request) GetUUID(key string) (uuid.UUID, error) {
	strValue, err := r.GetString(key)
	if err != nil {
		return uuid.UUID{}, err
	}
	uuidValue, err := uuid.Parse(strValue)
	if err != nil {
		return uuid.UUID{}, fmt.Errorf("could not parse uuid from key '%s'", key)
	}
	return uuidValue, nil
}

func (r Request) GetRoomID() (string, error) {
	return r.GetString("rid")
}

func (r Request) GetCommandID() (string, error) {
	return r.GetString("cid")
}

func (r Request) GetSessionID() (string, error) {
	return r.GetString("ssid")
}

func (r Request) GetAckMessageID() string {
	return r.Refs.AckMessageID
}

type RequestRefs struct {
	AckMessageID string
}

type RequestEvent struct {
	Refs RequestRefs
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

func MakeErrorResponse(err error) Response {
	return Response{
		IsSuccess: false,
		Error:     err.Error(),
	}
}

func MakeErrorResponseFromString(errStr string) Response {
	return MakeErrorResponse(fmt.Errorf(errStr))
}

func MakeRetriableErrorResponse(err error) Response {
	return Response{
		IsSuccess:   false,
		Error:       err.Error(),
		IsRetriable: true,
	}
}

func MakeSuccessResponse(data ...Dict) Response {
	var d Dict
	if len(data) > 0 {
		d = data[0]
	}
	return Response{
		IsSuccess: true,
		Data:      d,
	}
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

	// Optional index for parameter ordering
	Index int `json:"index" msgpack:"index"`
}
type RequestParams map[RequestParamName]RequestParamDef

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
	SID    RequestParamType
}{
	String: "str",
	Enum:   "enum",
	Int:    "int",
	Bool:   "bool",
	UUID:   "uuid",
	SID:    "sid",
}

var SupportedRequestParamTypes = map[string]struct{}{
	RequestParamTypes.String: {},
	RequestParamTypes.Enum:   {},
	RequestParamTypes.Int:    {},
	RequestParamTypes.Bool:   {},
	RequestParamTypes.UUID:   {},
	RequestParamTypes.SID:    {},
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
	return d.Commands.isValid()
}

func (d *Descriptor) addCommand(cmdDescriptor CommandDescriptor) error {
	newCommandsDescriptor := CommandsDescriptor{
		Descriptors: append(d.Commands.Descriptors, cmdDescriptor),
	}
	if err := newCommandsDescriptor.isValid(); err != nil {
		return err
	}
	d.Commands = newCommandsDescriptor
	return nil
}
