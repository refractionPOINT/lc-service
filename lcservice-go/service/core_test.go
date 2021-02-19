package service

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"testing"
)

const (
	testSecretKey = "abc"
)

func TestAuth(t *testing.T) {
	s, err := NewService(Descriptor{
		SecretKey: testSecretKey,
		Callbacks: DescriptorCallbacks{
			OnOrgInstall: func(r Request) Response {
				return Response{}
			},
			OnOrgUninstall: func(r Request) Response {
				return Response{}
			},
		},
	})
	if err != nil {
		t.Errorf("NewService: %v", err)
	}

	testData := Dict{
		"a": "a",
		"A": "c",
		"b": "b",
	}

	if _, isAccepted := s.ProcessRequest(testData, "nope"); isAccepted {
		t.Error("invalid sig accepted")
	}

	sig := computeSig(testData)

	if _, isAccepted := s.ProcessRequest(testData, sig); !isAccepted {
		t.Error("valid sig not accepted")
	}
}

func computeSig(data Dict) string {
	d, err := json.Marshal(data)
	if err != nil {
		panic(err)
	}
	mac := hmac.New(sha256.New, []byte(testSecretKey))
	if _, err := mac.Write(d); err != nil {
		panic(err)
	}
	expected := mac.Sum(nil)
	return hex.EncodeToString(expected)
}

func TestHealth(t *testing.T) {
	params := map[string]RequestParamDef{"p1": {
		Type:        "enum",
		Description: "ddd",
		IsRequired:  true,
		Values:      []string{"v1", "v2"},
	}}
	s, err := NewService(Descriptor{
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
	})
	if err != nil {
		t.Errorf("NewService: %v", err)
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
				"detect_subscriptions": []string{"d1", "d2"},
				"callbacks":            []string{"health", "org_uninstall"},
			},
		},
	}) {
		t.Errorf("unexpected: %+v", r)
	}
}

func makeRequest(r lcRequest) Dict {
	b, err := json.Marshal(r)
	if err != nil {
		panic("invalid request for json")
	}
	d := Dict{}
	if err := json.Unmarshal(b, &d); err != nil {
		panic("invalid json for request")
	}
	return d
}

func compareResponses(r1 Response, r2 Response) bool {
	b1, err := json.Marshal(r1)
	if err != nil {
		panic("invalid Response for json")
	}
	b2, err := json.Marshal(r2)
	if err != nil {
		panic("invalid Response for json")
	}
	isSame := string(b1) == string(b2)
	if !isSame {
		fmt.Println(string(b1))
		fmt.Println(string(b2))
	}
	return isSame
}
