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
%s:
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

type InteractiveService struct {
	cs *coreService

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

type TrackedTaskingOptions struct {
	Context Dict
	JobID   string
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

func (is *InteractiveService) Init() error {
	return is.cs.Init()
}

func (is *InteractiveService) ProcessRequest(data map[string]interface{}, sig string) (Response, bool) {
	return is.cs.ProcessRequest(data, sig)
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

func (is *InteractiveService) onOrgPer1H(r Request) Response {
	is.applyInteractiveRule(r.Org)

	return is.originalOnOrgPer1H(r)
}

func (is *InteractiveService) onOrgInstall(r Request) Response {
	is.applyInteractiveRule(r.Org)

	return is.originalOnOrgInstall(r)
}

func (is *InteractiveService) onOrgUninstall(r Request) Response {
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

func (is *InteractiveService) TrackedTasking(sensor *lc.Sensor, task string, opts TrackedTaskingOptions, cb InteractiveCallback) error {
	cbHash := is.getCbHash(cb)
	if _, ok := is.interactiveCallbacks[cbHash]; !ok {
		panic(fmt.Sprintf("tracked sensor task callback not registered: %v", cbHash))
	}
	serialCtx, err := json.Marshal(interactiveContext{
		CallbackID: cbHash,
		JobID:      opts.JobID,
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
