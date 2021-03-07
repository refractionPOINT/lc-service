package main

import (
	"net/http"
	"os"

	srv "github.com/refractionPOINT/lc-service/lcservice-go/servers"
	svc "github.com/refractionPOINT/lc-service/lcservice-go/service"
)

type templateService struct {
}

func (s *templateService) onOrgInstall(request svc.Request) svc.Response {
	return svc.Response{}
}

func (s *templateService) onOrgUninstall(request svc.Request) svc.Response {
	return svc.Response{}
}

func main() {
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

	sr := srv.NewStandalone(sv, 80)
	if err := sr.Init(); err != nil {
		panic(err)
	}
	if err := sr.Start(); err != nil && err != http.ErrServerClosed {
		panic(err)
	}
}
