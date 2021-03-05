package servers

import (
	"encoding/json"
	"fmt"
	"net/http"
)

type standalone struct {
	svc Service
	srv *http.Server
}

func NewStandalone(svc Service, port uint16) *standalone {
	sa := &standalone{
		svc: svc,
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/", sa.process)
	sa.srv = &http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: mux,
	}
	return sa
}

func (sa *standalone) Init() error {
	return sa.svc.Init()
}

func (sa *standalone) Start() error {
	return sa.srv.ListenAndServe()
}

func (sa *standalone) process(w http.ResponseWriter, r *http.Request) {
	d := map[string]interface{}{}
	if err := json.NewDecoder(r.Body).Decode(&d); err != nil {
		w.WriteHeader(400)
		return
	}
	s := r.Header.Get("lc-svc-sig")

	resp, isAccepted := sa.svc.ProcessRequest(d, s)
	if !isAccepted {
		w.WriteHeader(401)
		return
	}
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		w.WriteHeader(500)
		return
	}
}
