package servers

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"

	svc "github.com/refractionPOINT/lc-service/lcservice-go/service"
)

const (
	testSecretKey = "abc"
)

func TestAuth(t *testing.T) {
	testMeCB := func(req svc.Request) svc.Response {
		return svc.Response{IsSuccess: true}
	}

	a := assert.New(t)
	s, err := svc.NewService(svc.Descriptor{
		SecretKey:   testSecretKey,
		Log:         func(m string) { fmt.Println(m) },
		LogCritical: func(m string) { fmt.Printf("critial: %s\n", m) },
		IsDebug:     true,
		Commands: svc.CommandsDescriptor{
			Descriptors: []svc.CommandDescriptor{
				{
					Name: "testMe",
					Args: svc.CommandParams{
						"arg0": svc.RequestParamDef{
							Type:        svc.RequestParamTypes.String,
							Description: "arg0 description",
						},
					},
					Handler:     testMeCB,
					Description: "test command",
				},
			},
		},
	})
	a.NoError(err)

	data := svc.Dict{
		"etype": "health",
		"data":  svc.Dict{},
	}
	dataBytes, err := json.Marshal(data)
	a.NoError(err)
	recorder := httptest.NewRecorder()
	req := httptest.NewRequest("", "http://test-url.com", bytes.NewReader(dataBytes))

	// Set an invalid signature.
	req.Header.Add("lc-svc-sig", "aaaabbbbcccc")

	cf := NewCloudFunction(s)
	cf.Process(recorder, req)

	resp := recorder.Result()
	a.Equal(http.StatusUnauthorized, resp.StatusCode)
}

func TestProcess(t *testing.T) {
	testMeCB := func(req svc.Request) svc.Response {
		return svc.Response{IsSuccess: true}
	}

	a := assert.New(t)
	s, err := svc.NewService(svc.Descriptor{
		SecretKey:   testSecretKey,
		Log:         func(m string) { fmt.Println(m) },
		LogCritical: func(m string) { fmt.Printf("critial: %s\n", m) },
		IsDebug:     true,
		Commands: svc.CommandsDescriptor{
			Descriptors: []svc.CommandDescriptor{
				{
					Name: "testMe",
					Args: svc.CommandParams{
						"arg0": svc.RequestParamDef{
							Type:        svc.RequestParamTypes.String,
							Description: "arg0 description",
						},
					},
					Handler:     testMeCB,
					Description: "test command",
				},
			},
		},
	})
	a.NoError(err)

	data := svc.Dict{
		"etype": "health",
		"data":  svc.Dict{},
	}
	dataBytes, err := json.Marshal(data)
	a.NoError(err)
	recorder := httptest.NewRecorder()
	req := httptest.NewRequest("", "http://test-url.com", bytes.NewReader(dataBytes))
	req.Header.Add("lc-svc-sig", computeSig(dataBytes))

	cf := NewCloudFunction(s)
	cf.Process(recorder, req)

	resp := recorder.Result()
	a.Equal(http.StatusOK, resp.StatusCode)
	bytes, err := ioutil.ReadAll(resp.Body)
	a.NoError(err)
	respDict := svc.Dict{}
	a.NoError(json.Unmarshal(bytes, &respDict))
	a.Equal(true, respDict["success"])

	rawRespData := respDict["data"]
	respData := rawRespData.(svc.Dict)

	rawRespMTD := respData["mtd"]
	respMTD := rawRespMTD.(svc.Dict)

	rawRespCallbacks := respMTD["callbacks"]
	respCallbacks := rawRespCallbacks.([]interface{})
	a.Equal([]interface{}{"health"}, respCallbacks)

	rawRespCommands := respMTD["commands"]
	respCommands := rawRespCommands.(svc.Dict)
	a.Equal(map[string]interface{}{
		"testMe": svc.Dict{
			"args": svc.Dict{
				"arg0": svc.Dict{
					"desc":        "arg0 description",
					"is_required": false,
					"type":        "str",
					"index":       float64(0),
				},
			},
			"name": "testMe",
			"desc": "test command",
		},
	}, respCommands)
}

func computeSig(data []byte) string {
	mac := hmac.New(sha256.New, []byte(testSecretKey))
	if _, err := mac.Write(data); err != nil {
		return ""
	}
	jsonCompatSig := []byte(hex.EncodeToString(mac.Sum(nil)))
	return string(jsonCompatSig)
}
