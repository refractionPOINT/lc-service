package servers

type Service interface {
	Init() error
	ProcessRequest(data map[string]interface{}, sig string) (response interface{}, isAccepted bool)
	ProcessCommand(commandArguments map[string]interface{}, sig string) (response interface{}, isAccepted bool)
}
