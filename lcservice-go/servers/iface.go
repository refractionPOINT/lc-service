package servers

import svc "github.com/refractionPOINT/lc-service/lcservice-go/service"

type Service interface {
	Init() error
	ProcessRequest(data map[string]interface{}) svc.Response
	ProcessCommand(commandArguments map[string]interface{}) svc.Response
	GetSecretKey() []byte
}
