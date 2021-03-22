package servers

import (
	"encoding/json"
	"net/http"

	svc "github.com/refractionPOINT/lc-service/lcservice-go/service"
)

func encodeResponse(resp svc.Response, w http.ResponseWriter) {
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}

func handleResponse(resp svc.Response, isAccepted bool, w http.ResponseWriter) {
	if !isAccepted {
		w.WriteHeader(http.StatusPreconditionFailed)
		return
	}
	if resp.Error != "" {
		w.WriteHeader(http.StatusBadRequest)
		encodeResponse(resp, w)
		return
	}

	w.WriteHeader(http.StatusOK)
	encodeResponse(resp, w)
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
