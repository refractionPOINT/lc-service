package servers

import (
	"encoding/json"
	"net/http"

	svc "github.com/refractionPOINT/lc-service/lcservice-go/service"
)

func isRequestNotImplemented(resp interface{}) bool {
	svcResp, ok := resp.(svc.Response)
	if !ok {
		return false
	}
	return svcResp.Error == svc.ErrNotImplemented.Error
}

func handleResponse(resp interface{}, isAccepted bool, w http.ResponseWriter) {
	if !isAccepted {
		w.WriteHeader(401)
		return
	}
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		w.WriteHeader(500)
		return
	}
}

func process(service Service, w http.ResponseWriter, r *http.Request) {
	d := map[string]interface{}{}
	if err := json.NewDecoder(r.Body).Decode(&d); err != nil {
		w.WriteHeader(400)
		return
	}
	sig := r.Header.Get("lc-svc-sig")

	requestTypeValue, ok := d["request_type"]
	if !ok || requestTypeValue == "request" {
		// it's not there we assume it's a regular request
		resp, isAccepted := service.ProcessRequest(d, sig)
		handleResponse(resp, isAccepted, w)
		return
	}
	if requestTypeValue == "command" {
		resp, isAccepted := service.ProcessCommand(d, sig)
		handleResponse(resp, isAccepted, w)
		return
	}
}
