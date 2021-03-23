package service

import (
	"encoding/json"
	"fmt"
	"testing"
)

func TestInteractive(t *testing.T) {
	params := map[string]RequestParamDef{"p1": {
		Type:        "enum",
		Description: "ddd",
		IsRequired:  true,
		Values:      []string{"v1", "v2"},
	}}
	testCB := func(r InteractiveRequest) Response {
		if _, ok := r.Event["a"]; ok {
			return Response{IsSuccess: true, Data: Dict{"yes": 1}}
		}
		return Response{IsSuccess: false}
	}
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
	}, []InteractiveCallback{testCB})
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
	resp.Data["start_time"] = 0

	if !compareResponses(resp, Response{
		IsSuccess: true,
		Data: Dict{
			"version":           1,
			"calls_in_progress": 1,
			"start_time":        0,
			"mtd": Dict{
				"request_params":       params,
				"detect_subscriptions": []string{"d1", "d2", "svc-testService-ex"},
				"callbacks":            []string{"detection", "health", "org_install", "org_per_1h", "org_uninstall"},
				"commands":             Dict{},
			},
		},
	}) {
		t.Errorf("unexpected: %+v", resp)
	}

	// Make a request to callback.
	cbHash := s.getCbHash(testCB)
	iContext, err := json.Marshal(interactiveContext{
		CallbackID: cbHash,
		Context: Dict{
			"some": "ctx",
		},
	})
	if err != nil {
		t.Errorf("json.Marshal: %v", err)
	}
	testData = makeRequest(lcRequest{
		Version:  1,
		JWT:      "",
		OID:      "",
		MsgID:    "",
		Deadline: 0,
		Type:     "detection",
		Data: Dict{
			"detect": Dict{
				"a": "yes",
			},
			"routing": Dict{
				"investigation_id": fmt.Sprintf("%s/%s", s.detectionName, iContext),
			},
		},
	})
	sig = computeSig(testData)

	resp, isAccepted = s.ProcessRequest(testData, sig)
	if !isAccepted {
		t.Error("valid sig not accepted")
	}

	if !compareResponses(resp, Response{
		IsSuccess: true,
		Data: Dict{
			"yes": 1,
		},
	}) {
		t.Errorf("unexpected: %+v", resp)
	}
}
