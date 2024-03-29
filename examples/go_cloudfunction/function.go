package service

import (
	"net/http"
	"os"

	srv "github.com/refractionPOINT/lc-service/lcservice-go/servers"
	svc "github.com/refractionPOINT/lc-service/lcservice-go/service"
)

type templateService struct {
}

var cf *srv.CloudFunction

func init() {
	tSvc := templateService{}
	sv, err := svc.NewService(svc.Descriptor{
		SecretKey: os.Getenv("SHARED_SECRET"),
		Callbacks: svc.DescriptorCallbacks{
			OnOrgInstall:   tSvc.onOrgInstall,
			OnOrgUninstall: tSvc.onOrgUninstall,
		},
	})
	if err != nil {
		panic(err)
	}

	cf = srv.NewCloudFunction(sv)
	if err := cf.Init(); err != nil {
		panic(err)
	}
}

func (s *templateService) onOrgInstall(request svc.Request) svc.Response {
	return svc.Response{}
}

func (s *templateService) onOrgUninstall(request svc.Request) svc.Response {
	return svc.Response{}
}

func ServiceMain(w http.ResponseWriter, r *http.Request) {
	cf.Process(w, r)
}
