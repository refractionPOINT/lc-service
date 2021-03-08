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

func process(service Service, w http.ResponseWriter, r *http.Request) {
	d := map[string]interface{}{}
	if err := json.NewDecoder(r.Body).Decode(&d); err != nil {
		w.WriteHeader(400)
		return
	}
	sig := r.Header.Get("lc-svc-sig")

	resp, isAccepted := service.ProcessRequest(d, sig)
	if !isAccepted {
		w.WriteHeader(401)
		return
	}

	if isRequestNotImplemented(resp) {
		resp, isAccepted = service.ProcessCommand(d, sig)
	}

	if err := json.NewEncoder(w).Encode(resp); err != nil {
		w.WriteHeader(500)
		return
	}
}
