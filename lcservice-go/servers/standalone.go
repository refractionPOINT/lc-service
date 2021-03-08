package servers

import (
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
	process(sa.svc, w, r)
}
