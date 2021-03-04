package servers

import (
	"encoding/json"
	"net/http"
)

type CloudFunction struct {
	svc Service
}

func NewCloudFunction(svc Service) *CloudFunction {
	return &CloudFunction{
		svc: svc,
	}
}

func (cf *CloudFunction) Init() error {
	return cf.svc.Init()
}

func (cf *CloudFunction) Process(w http.ResponseWriter, r *http.Request) {
	d := map[string]interface{}{}
	if err := json.NewDecoder(r.Body).Decode(&d); err != nil {
		w.WriteHeader(400)
		return
	}
	s := r.Header.Get("lc-svc-sig")

	resp, isAccepted := cf.svc.ProcessRequest(d, s)
	if !isAccepted {
		w.WriteHeader(401)
		return
	}
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		w.WriteHeader(500)
		return
	}
}
