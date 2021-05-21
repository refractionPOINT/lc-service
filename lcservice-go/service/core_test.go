package service

import (
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
