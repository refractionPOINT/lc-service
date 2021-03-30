package servers

import (
	"bytes"
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
	})
	a.NoError(err)
	a.NoError(s.AddCommandHandler("testMe", svc.Dict{"arg0": "arg0 description"}, testMeCB))

	data := svc.Dict{
		"etype": "health",
		"data":  svc.Dict{},
	}
	dataBytes, err := json.Marshal(data)
	a.NoError(err)
	recorder := httptest.NewRecorder()
	req := httptest.NewRequest("", "http://test-url.com", bytes.NewReader(dataBytes))
	req.Header.Add("lc-svc-sig", "c8ef520e8d1047e137696e59d445b6cfd078b6c377d9af9dbf9ee37ccefdacb5")

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
				"arg0": "arg0 description",
			},
			"name": "testMe",
		},
	}, respCommands)

}
