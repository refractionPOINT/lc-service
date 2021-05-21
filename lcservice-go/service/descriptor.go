package service

import (
	"errors"
	"fmt"

	"github.com/refractionPOINT/lc-service/lcservice-go/common"
)

type Request = common.Request
type RequestEvent = common.RequestEvent
type Response = common.Response

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

type ServiceCallback = common.ServiceCallback
type RequestParamName = common.RequestParamName
type RequestParamType = common.RequestParamType
type RequestParamDef = common.RequestParamDef
type RequestParams = common.RequestParams

func requestParamsIsValid(params RequestParams) error {
	for paramName, paramDef := range params {
		if paramName == "" {
			return fmt.Errorf("parameter name is empty")
		}
		if err := paramDef.IsValid(); err != nil {
			return err
		}
	}
	return nil
}

var RequestParamTypes = common.RequestParamTypes
var SupportedRequestParamTypes = common.SupportedRequestParamTypes

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
