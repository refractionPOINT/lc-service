package service

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"reflect"
	"runtime"
	"strconv"
	"strings"

	"gopkg.in/yaml.v2"

	"github.com/google/uuid"
	lc "github.com/refractionPOINT/go-limacharlie/limacharlie"
)

const (
	interactiveRuleTemplate = `
%s:
  namespace: managed
  detect:
    op: and
    rules:
      - op: starts with
        path: routing/investigation_id
        value: %s
      - op: is
        not: true
        path: routing/event_type
        value: CLOUD_NOTIFICATION
  respond:
    - action: report
      name: __%s`
)

type InteractiveService struct {
	cs *CoreService

	// Rule used to get responses back.
	detectionName   string
	interactiveRule map[string]lc.CoreDRRule

	// Original user-defined callbacks
	// that we need to overload.
	originalOnDetection    ServiceCallback
	originalOnOrgPer1H     ServiceCallback
	originalOnOrgInstall   ServiceCallback
	originalOnOrgUninstall ServiceCallback

	interactiveCallbacks map[string]InteractiveCallback
}

type InteractiveRequest struct {
	Org            *lc.Organization
	OID            string
	SID            string
	Event          Dict
	Job            *Job
	Context        Dict
	ServiceRequest Request
}

func (r InteractiveRequest) GetFromContext(key string) (interface{}, error) {
	dataValue, found := r.Context[key]
	if !found {
		return "", fmt.Errorf("key '%s' not found", key)
	}
	return dataValue, nil
}

func (r InteractiveRequest) GetStringFromContext(key string) (string, error) {
	dataValue, err := r.GetFromContext(key)
	if err != nil {
		return "", err
	}
	value, ok := dataValue.(string)
	if !ok {
		return "", fmt.Errorf("key '%s' is not a string", key)
	}
	return value, nil
}

func (r InteractiveRequest) GetEnumValueFromContext(key string, requestParams RequestParams) (string, error) {
	paramDef, found := requestParams[key]
	if !found {
		return "", fmt.Errorf("key '%s' is not an expected parameter", key)
	}
	if paramDef.Type != RequestParamTypes.Enum {
		return "", fmt.Errorf("key '%s' is not of enum type", key)
	}
	enumValue, err := r.GetStringFromContext(key)
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

func (r InteractiveRequest) GetIntFromContext(key string) (int, error) {
	dataValue, err := r.GetFromContext(key)
	if err != nil {
		return 0, err
	}
	value, ok := dataValue.(int)
	if !ok {
		return 0, fmt.Errorf("key '%s' is not an integer", key)
	}
	return value, nil
}

func (r InteractiveRequest) GetBoolFromContext(key string) (bool, error) {
	dataValue, err := r.GetFromContext(key)
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

func (r InteractiveRequest) GetUUIDFromContext(key string) (uuid.UUID, error) {
	strValue, err := r.GetStringFromContext(key)
	if err != nil {
		return uuid.UUID{}, err
	}
	uuidValue, err := uuid.Parse(strValue)
	if err != nil {
		return uuid.UUID{}, fmt.Errorf("could not parse uuid from key '%s'", key)
	}
	return uuidValue, nil
}

type InteractiveCallback = func(InteractiveRequest) Response

// Canonical format for context passing
// between Services and LimaCharlie.
type interactiveContext struct {
	CallbackID string `json:"cb" msgpack:"cb"`
	JobID      string `json:"j,omitempty" msgpack:"j,omitempty"`
	SessionID  string `json:"s,omitempty" msgpack:"s,omitempty"`
	Context    Dict   `json:"c" msgpack:"c"`
}

type inboundDetection struct {
	Detect  Dict `json:"detect" msgpack:"detect"`
	Routing struct {
		InvestigationID string `json:"investigation_id" msgpack:"investigation_id"`
		SensorID        string `json:"sid,omitempty" msgpack:"sid,omitempty"`
	} `json:"routing" msgpack:"routing"`
}

type TrackedTaskingOptions struct {
	Context   Dict
	JobID     string
	SessionID string
}

func NewInteractiveService(descriptor Descriptor, callbacks []InteractiveCallback) (is *InteractiveService, err error) {
	is = &InteractiveService{}

	// Install a D&R rule and a Detection subscription.
	is.detectionName = fmt.Sprintf("svc-%s-ex", descriptor.Name)
	is.interactiveRule = map[string]lc.CoreDRRule{}
	templatedRule := fmt.Sprintf(interactiveRuleTemplate, is.detectionName, is.detectionName, is.detectionName)
	if err := yaml.Unmarshal([]byte(templatedRule), &is.interactiveRule); err != nil {
		panic(fmt.Sprintf("error parsing interactive rule (%v): %s", err, templatedRule))
	}
	descriptor.DetectionsSubscribed = append(descriptor.DetectionsSubscribed, fmt.Sprintf("__%s", is.detectionName))

	// Overload a few callbacks.
	is.originalOnDetection = descriptor.Callbacks.OnDetection
	is.originalOnOrgPer1H = descriptor.Callbacks.OnOrgPer1H
	is.originalOnOrgInstall = descriptor.Callbacks.OnOrgInstall
	is.originalOnOrgUninstall = descriptor.Callbacks.OnOrgUninstall
	descriptor.Callbacks.OnDetection = is.onDetection
	descriptor.Callbacks.OnOrgPer1H = is.onOrgPer1H
	descriptor.Callbacks.OnOrgInstall = is.onOrgInstall
	descriptor.Callbacks.OnOrgUninstall = is.onOrgUninstall

	is.cs, err = NewService(descriptor)
	if err != nil {
		return nil, err
	}

	// Compute the callbacks.
	is.interactiveCallbacks = map[string]InteractiveCallback{}
	is.registerInteractiveCallbacks(callbacks)

	return is, err
}

func (is *InteractiveService) registerInteractiveCallbacks(callbacks []InteractiveCallback) {
	for _, cb := range callbacks {
		is.interactiveCallbacks[is.getCbHash(cb)] = cb
	}
}

func (is *InteractiveService) RegisterCommand(cmdDescriptor CommandDescriptor, interactiveCb ...InteractiveCallback) error {
	if err := is.cs.desc.addCommand(cmdDescriptor); err != nil {
		return err
	}
	if len(interactiveCb) == 0 {
		return nil
	}

	is.registerInteractiveCallbacks([]InteractiveCallback{interactiveCb[0]})
	return nil
}

func (is *InteractiveService) Init() error {
	return is.cs.Init()
}

func (is *InteractiveService) GetSecretKey() []byte {
	return []byte(is.cs.desc.SecretKey)
}

func (is *InteractiveService) ProcessRequest(data map[string]interface{}) Response {
	return is.cs.ProcessRequest(data)
}

func (is *InteractiveService) ProcessCommand(commandArguments map[string]interface{}) Response {
	return is.cs.ProcessCommand(commandArguments)
}

func (is *InteractiveService) getCbHash(cb interface{}) string {
	name := runtime.FuncForPC(reflect.ValueOf(cb).Pointer()).Name()
	h := md5.Sum([]byte(fmt.Sprintf("%s/%s", is.cs.desc.SecretKey, name)))
	return hex.EncodeToString(h[:])[:8]
}

func (is *InteractiveService) onDetection(r Request) Response {
	detection := inboundDetection{}
	// Get the basic headers we use to tell if this is
	// for the interactive service, or the user.
	if err := DictToStruct(r.Event.Data, &detection); err != nil {
		// Pass through to user.
		return is.originalOnDetection(r)
	}
	// Check the routing investigation ID to see if it's for us.
	if !strings.HasPrefix(detection.Routing.InvestigationID, is.detectionName) {
		// Pass through to user.
		return is.originalOnDetection(r)
	}
	ic, isICValid := parseInteractiveContext(detection.Routing.InvestigationID)
	if !isICValid {
		// Pass through to user.
		return is.originalOnDetection(r)
	}
	req := InteractiveRequest{
		Org:            r.Org,
		OID:            r.OID,
		SID:            detection.Routing.SensorID,
		Event:          detection.Detect,
		Context:        ic.Context,
		ServiceRequest: r,
	}

	if ic.JobID != "" {
		req.Job = NewJob(ic.JobID)
	}

	// Get the right callback.
	if ic.CallbackID == "" {
		is.cs.desc.LogCritical(fmt.Sprintf("received interactive callback without callbackID: %s", detection.Routing.InvestigationID))
		return is.originalOnDetection(r)
	}

	cb, ok := is.interactiveCallbacks[ic.CallbackID]
	if !ok {
		is.cs.desc.LogCritical(fmt.Sprintf("received interactive callback with unknown callbackID: %s", detection.Routing.InvestigationID))
		return is.originalOnDetection(r)
	}
	if cb == nil {
		return NewErrorResponse(fmt.Errorf("not implemented"))
	}
	return cb(req)
}

func parseInteractiveContext(invID string) (interactiveContext, bool) {
	ic := interactiveContext{}
	components := strings.SplitN(invID, "/", 2)
	if len(components) != 2 {
		return ic, false
	}
	if err := json.Unmarshal([]byte(components[1]), &ic); err != nil {
		return ic, false
	}
	return ic, true
}

func (is *InteractiveService) onOrgPer1H(r Request) Response {
	if err := is.applyInteractiveRule(r.Org); err != nil {
		is.cs.desc.LogCritical(fmt.Sprintf("onOrgPer1H.applyInteractiveRule: %v", err))
	}

	if is.originalOnOrgPer1H == nil {
		return NewErrorResponse(fmt.Errorf("not implemented"))
	}

	return is.originalOnOrgPer1H(r)
}

func (is *InteractiveService) onOrgInstall(r Request) Response {
	if err := is.applyInteractiveRule(r.Org); err != nil {
		is.cs.desc.LogCritical(fmt.Sprintf("onOrgInstall.applyInteractiveRule: %v", err))
	}

	if is.originalOnOrgInstall == nil {
		return NewErrorResponse(fmt.Errorf("not implemented"))
	}

	return is.originalOnOrgInstall(r)
}

func (is *InteractiveService) onOrgUninstall(r Request) Response {
	if err := is.removeInteractiveRule(r.Org); err != nil {
		is.cs.desc.LogCritical(fmt.Sprintf("onOrgUninstall.removeInteractiveRule: %v", err))
	}

	if is.originalOnOrgUninstall == nil {
		return NewErrorResponse(fmt.Errorf("not implemented"))
	}

	// Remove interactive rules
	return is.originalOnOrgUninstall(r)
}

func (is *InteractiveService) applyInteractiveRule(org *lc.Organization) error {
	c := lc.OrgConfig{
		DRRules: is.interactiveRule,
	}
	if _, err := org.SyncPush(c, lc.SyncOptions{
		SyncDRRules: true,
	}); err != nil {
		is.cs.desc.LogCritical(fmt.Sprintf("error syncing interactive rule: %v", err))
		return err
	}
	return nil
}

func (is *InteractiveService) removeInteractiveRule(org *lc.Organization) error {
	if err := org.DRRuleDelete(is.detectionName); err != nil {
		is.cs.desc.LogCritical(fmt.Sprintf("error removing interactive rule: %v", err))
		return err
	}
	return nil
}

func (is *InteractiveService) TrackedTasking(sensor *lc.Sensor, task string, opts TrackedTaskingOptions, cb InteractiveCallback) error {
	cbHash := is.getCbHash(cb)
	if _, ok := is.interactiveCallbacks[cbHash]; !ok {
		panic(fmt.Sprintf("tracked sensor task callback not registered: %v", cbHash))
	}
	serialCtx, err := json.Marshal(interactiveContext{
		CallbackID: cbHash,
		JobID:      opts.JobID,
		SessionID:  opts.SessionID,
		Context:    opts.Context,
	})
	if err != nil {
		return err
	}

	if err := sensor.Task(task, lc.TaskingOptions{
		InvestigationID:      is.detectionName,
		InvestigationContext: string(serialCtx),
	}); err != nil {
		return err
	}
	return nil
}

// LC.Logger Interface Compatibility
func (is *InteractiveService) Fatal(msg string) {
	if is.cs.desc.LogCritical == nil {
		return
	}
	is.cs.desc.LogCritical(msg)
}
func (is *InteractiveService) Error(msg string) {
	if is.cs.desc.LogCritical == nil {
		return
	}
	is.cs.desc.LogCritical(msg)
}
func (is *InteractiveService) Warn(msg string) {
	if is.cs.desc.LogCritical == nil {
		return
	}
	is.cs.desc.LogCritical(msg)
}
func (is *InteractiveService) Info(msg string) {
	if is.cs.desc.Log == nil {
		return
	}
	is.cs.desc.Log(msg)
}
func (is *InteractiveService) Debug(msg string) {
	if is.cs.desc.Log == nil {
		return
	}
	is.cs.desc.Log(msg)
}
func (is *InteractiveService) Trace(msg string) {
	if is.cs.desc.Log == nil {
		return
	}
	is.cs.desc.Log(msg)
}
