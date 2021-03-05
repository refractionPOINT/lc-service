package service

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
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

func TestResourceConvertion(t *testing.T) {
	// Single resource request
	r, err := RequestEvent{
		Data: Dict{
			"resource":        "test1",
			"is_include_data": true,
		},
	}.AsResourceRequest()
	if err != nil {
		t.Errorf("AsResourceRequest: %v", err)
	}
	if !r.inIncludeData {
		t.Errorf("wrong isIncludeData: %+v", r)
	}
	if !r.isSingleRes {
		t.Errorf("wrong isSingleRes: %+v", r)
	}
	if len(r.ResourceNames) != 1 || r.ResourceNames[0] != "test1" {
		t.Errorf("wrong ResourceNames: %+v", r)
	}

	// Multi resource request
	r, err = RequestEvent{
		Data: Dict{
			"resource":        []string{"test1", "test2"},
			"is_include_data": true,
		},
	}.AsResourceRequest()
	if err != nil {
		t.Errorf("AsResourceRequest: %v", err)
	}
	if !r.inIncludeData {
		t.Errorf("wrong isIncludeData: %+v", r)
	}
	if r.isSingleRes {
		t.Errorf("wrong isSingleRes: %+v", r)
	}
	if len(r.ResourceNames) != 2 || r.ResourceNames[0] != "test1" || r.ResourceNames[1] != "test2" {
		t.Errorf("wrong ResourceNames: %+v", r)
	}
}

func TestResourceResponse(t *testing.T) {
	resData := []byte("thisisatest")
	h := sha256.Sum256(resData)
	resHash := hex.EncodeToString(h[:])
	resEncoded := base64.StdEncoding.EncodeToString(resData)

	r := NewResourceFromData("lookup", resData)
	if r == nil {
		t.Error("bad load")
	}
	if r.Category != "lookup" {
		t.Errorf("wrong cat: %+v", r)
	}
	if r.Data != resEncoded {
		t.Errorf("wrong dat: %+v", r)
	}
	if r.Hash != resHash {
		t.Errorf("wrong hash: %+v", r)
	}
}

func TestResourceSupply(t *testing.T) {
	// Single resource
	r, err := RequestEvent{
		Data: Dict{
			"resource":        []string{"test1", "test2"},
			"is_include_data": true,
		},
	}.AsResourceRequest()

	if err != nil {
		t.Errorf("AsResourceRequest: %v", err)
	}

	s := map[string]*ResourceResponse{
		"test1": NewResourceFromData("lookup", []byte("data1")),
		"test2": NewResourceFromData("lookup", []byte("data2")),
	}

	resp := r.SupplyResponse(s)
	if fmt.Sprintf("%+v", resp) != `{IsSuccess:true IsRetriable:false Error: Data:map[resources:map[test1:map[hash:5b41362bc82b7f3d56edc5a306db22105707d01ff4819e26faef9724a2d406c9 res_cat:lookup res_data:ZGF0YTE=] test2:map[hash:d98cf53e0c8b77c14a96358d5b69584225b4bb9026423cbc2f7b0161894c402c res_cat:lookup res_data:ZGF0YTI=]]] Jobs:[]}` {
		t.Errorf("unexpected supply: %+v", resp)
	}

	// Multi resource
	r, err = RequestEvent{
		Data: Dict{
			"resource":        "test1",
			"is_include_data": true,
		},
	}.AsResourceRequest()

	if err != nil {
		t.Errorf("AsResourceRequest: %v", err)
	}

	s = map[string]*ResourceResponse{
		"test1": NewResourceFromData("lookup", []byte("data1")),
	}

	resp = r.SupplyResponse(s)
	if fmt.Sprintf("%+v", resp) != `{IsSuccess:true IsRetriable:false Error: Data:map[hash:5b41362bc82b7f3d56edc5a306db22105707d01ff4819e26faef9724a2d406c9 res_cat:lookup res_data:ZGF0YTE=] Jobs:[]}` {
		t.Errorf("unexpected supply: %+v", resp)
	}
}

func TestCommand(t *testing.T) {
	a := assert.New(t)
	testCommandOneCB := func(req Request) Response {
		return Response{IsSuccess: true, Data: Dict{"from": "cbOne"}}
	}
	testCommandTwoCB := func(req Request) Response {
		return Response{IsSuccess: true, Data: Dict{"from": "cbTwo"}}
	}
	s, err := NewService(Descriptor{
		SecretKey:   testSecretKey,
		Log:         func(m string) { fmt.Println(m) },
		LogCritical: func(m string) { fmt.Println(m) },
	})
	a.NoError(s.AddCommandHandler("commandOne", Dict{}, testCommandOneCB))
	a.NoError(s.AddCommandHandler("commandTwo", Dict{}, testCommandTwoCB))
	a.NoError(err)
	a.NotNil(s)

	testData := makeRequest(lcRequest{
		Version: 1,
		Type:    "commandOne",
	})
	sig := computeSig(testData)
	resp, accepted := s.ProcessCommand(testData, sig)
	a.True(accepted)
	r := resp.(Response)
	a.Equal(Dict{"from": "cbOne"}, r.Data)

	testData = makeRequest(lcRequest{
		Version: 1,
		Type:    "commandTwo",
	})
	sig = computeSig(testData)
	resp, accepted = s.ProcessCommand(testData, sig)
	a.True(accepted)
	r = resp.(Response)
	a.Equal(Dict{"from": "cbTwo"}, r.Data)

	a.Error(s.AddCommandHandler("", Dict{}, testCommandOneCB))
	a.Error(s.AddCommandHandler("commandOne", Dict{}, testCommandOneCB))
	a.Error(s.AddCommandHandler("commandThree", Dict{}, nil))
	a.NoError(s.AddCommandHandler("commandThree", Dict{}, testCommandOneCB))
}
