package service

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"reflect"
	"runtime"
	"strings"

	"gopkg.in/yaml.v2"

	lc "github.com/refractionPOINT/go-limacharlie/limacharlie"
)

const (
	interactiveRuleTemplate = `
namespace: replicant
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

type interactiveService struct {
	cs *coreService

	// Rule used to get responses back.
	detectionName   string
	interactiveRule Dict

	// Original user-defined callbacks
	// that we need to overload.
	originalOnDetection    ServiceCallback
	originalOnOrgPer1H     ServiceCallback
	originalOnOrgInstall   ServiceCallback
	originalOnOrgUninstall ServiceCallback

	interactiveCallbacks map[string]InteractiveCallback
}

type InteractiveRequest struct {
	Org     *lc.Organization
	OID     string
	Event   Dict
	Job     *Job
	Context Dict
}

type InteractiveCallback = func(InteractiveRequest) Response

// Canonical format for context passing
// between Services and LimaCharlie.
type interactiveContext struct {
	SessionID  string `json:"si"`
	CallbackID string `json:"cb"`
	JobID      string `json:"j"`
	Context    Dict   `json:"c"`
}

type inboundDetection struct {
	Detect  Dict `json:"detect"`
	Routing struct {
		InvestigationID string `json:"investigation_id"`
	} `json:"routing"`
}

func NewInteractiveService(descriptor Descriptor, callbacks []InteractiveCallback) (is *interactiveService, err error) {
	is = &interactiveService{}

	// Install a D&R rule and a Detection subscription.
	is.detectionName = fmt.Sprintf("svc-%s-ex", descriptor.Name)
	if err := yaml.Unmarshal([]byte(fmt.Sprintf(interactiveRuleTemplate, is.detectionName, is.detectionName)), &is.interactiveRule); err != nil {
		panic("error parsing interactive rule")
	}
	descriptor.DetectionsSubscribed = append(descriptor.DetectionsSubscribed, is.detectionName)

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
	for _, cb := range callbacks {
		is.interactiveCallbacks[is.getCbHash(cb)] = cb
	}

	return is, err
}

func (is *interactiveService) Init() error {
	return is.cs.Init()
}
func (is *interactiveService) ProcessRequest(data map[string]interface{}, sig string) (response interface{}, isAccepted bool) {
	return is.cs.ProcessRequest(data, sig)
}

func (is *interactiveService) getCbHash(cb interface{}) string {
	name := runtime.FuncForPC(reflect.ValueOf(cb).Pointer()).Name()
	h := md5.Sum([]byte(fmt.Sprintf("%s/%s", is.cs.desc.SecretKey, name)))
	return hex.EncodeToString(h[:])[:8]
}

func (is *interactiveService) onDetection(r Request) Response {
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
		Org:     r.Org,
		OID:     r.OID,
		Event:   detection.Detect,
		Context: ic.Context,
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

func (is *interactiveService) onOrgPer1H(r Request) Response {
	is.applyInteractiveRule()

	return is.originalOnOrgPer1H(r)
}

func (is *interactiveService) onOrgInstall(r Request) Response {
	is.applyInteractiveRule()

	return is.originalOnOrgInstall(r)
}

func (is *interactiveService) onOrgUninstall(r Request) Response {
	// Remove interactive rules

	return is.originalOnOrgUninstall(r)
}

func (is *interactiveService) applyInteractiveRule() {
	// TODO apply rules
}
