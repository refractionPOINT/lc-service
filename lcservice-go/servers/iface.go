package servers

import svc "github.com/refractionPOINT/lc-service/lcservice-go/service"

type Service interface {
	Init() error
	ProcessRequest(data map[string]interface{}, sig string) (response svc.Response, isAccepted bool)
	ProcessCommand(commandArguments map[string]interface{}, sig string) (response svc.Response, isAccepted bool)
}
