package servers

import (
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
	process(cf.svc, w, r)
}
