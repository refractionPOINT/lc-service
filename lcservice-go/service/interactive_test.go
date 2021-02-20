package service

import (
	"fmt"
	"testing"
)

func TestInteractiveHealth(t *testing.T) {
	params := map[string]RequestParamDef{"p1": {
		Type:        "enum",
		Description: "ddd",
		IsRequired:  true,
		Values:      []string{"v1", "v2"},
	}}
	s, err := NewInteractiveService(Descriptor{
		Name:        "testService",
		SecretKey:   testSecretKey,
		Log:         func(m string) { fmt.Println(m) },
		LogCritical: func(m string) { fmt.Println(m) },
		Callbacks: DescriptorCallbacks{
			OnOrgUninstall: func(r Request) Response {
				return Response{}
			},
		},
		DetectionsSubscribed: []string{"d1", "d2"},
		RequestParameters:    params,
	}, []InteractiveCallback{})
	if err != nil {
		t.Errorf("NewInteractiveService: %v", err)
	}

	testData := makeRequest(lcRequest{
		Version:  1,
		JWT:      "",
		OID:      "",
		MsgID:    "",
		Deadline: 0,
		Type:     "health",
		Data:     Dict{},
	})

	sig := computeSig(testData)

	resp, isAccepted := s.ProcessRequest(testData, sig)
	if !isAccepted {
		t.Error("valid sig not accepted")
	}
	r := resp.(Response)
	r.Data["start_time"] = 0

	if !compareResponses(r, Response{
		IsSuccess: true,
		Data: Dict{
			"version":           1,
			"calls_in_progress": 1,
			"start_time":        0,
			"mtd": Dict{
				"request_params":       params,
				"detect_subscriptions": []string{"d1", "d2", "svc-testService-ex"},
				"callbacks":            []string{"detection", "health", "org_install", "org_per_1h", "org_uninstall"},
			},
		},
	}) {
		t.Errorf("unexpected: %+v", r)
	}
}
