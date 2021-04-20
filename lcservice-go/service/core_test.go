package service

import (
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

	resp := s.ProcessRequest(testData)
	resp.Data["start_time"] = 0

	if !compareResponses(resp, Response{
		IsSuccess: true,
		Data: Dict{
			"version":           1,
			"calls_in_progress": 1,
			"start_time":        0,
			"mtd": Dict{
				"request_params":       params,
				"detect_subscriptions": []string{"d1", "d2"},
				"callbacks":            []string{"health", "org_uninstall"},
				"commands":             Dict{},
			},
		},
	}) {
		t.Errorf("unexpected: %+v", resp)
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
		IsDebug:     true,
		Commands: CommandsDescriptor{
			Descriptors: []CommandDescriptor{
				{
					Name:        "commandOne",
					Args:        CommandParams{},
					Handler:     testCommandOneCB,
					Description: "cmd1",
				},
				{
					Name:        "commandTwo",
					Args:        CommandParams{},
					Handler:     testCommandTwoCB,
					Description: "cmd2",
				},
			},
		},
	})
	a.NoError(err)
	a.NotNil(s)

	testData := makeRequest(lcRequest{
		Version: 1,
		Type:    "command",
		Data: Dict{
			"command_name": "commandOne",
			"rid":          "123",
			"cid":          "456",
		},
	})
	resp := s.ProcessCommand(testData)
	a.Empty(resp.Error)
	a.Equal(Dict{"from": "cbOne"}, resp.Data)

	testData = makeRequest(lcRequest{
		Version: 1,
		Type:    "command",
		Data: Dict{
			"command_name": "commandTwo",
			"rid":          "123",
			"cid":          "456",
		},
	})
	resp = s.ProcessCommand(testData)
	a.Empty(resp.Error)
	a.Equal(Dict{"from": "cbTwo"}, resp.Data)

	d := Descriptor{
		Commands: CommandsDescriptor{
			Descriptors: []CommandDescriptor{
				{
					Name:    "",
					Args:    CommandParams{},
					Handler: testCommandOneCB,
				},
			},
		},
	}
	a.Error(d.IsValid())

	d = Descriptor{
		Commands: CommandsDescriptor{
			Descriptors: []CommandDescriptor{
				{
					Name:    "commandOne",
					Args:    CommandParams{},
					Handler: testCommandOneCB,
				},
				{
					Name:    "commandOne",
					Args:    CommandParams{},
					Handler: testCommandOneCB,
				},
			},
		},
	}
	a.Error(d.IsValid())

	d = Descriptor{
		Commands: CommandsDescriptor{
			Descriptors: []CommandDescriptor{
				{
					Name:    "commandOne",
					Args:    CommandParams{},
					Handler: nil,
				},
			},
		},
	}
	a.Error(d.IsValid())
}
