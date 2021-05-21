package request

import (
	"github.com/refractionPOINT/lc-service/lcservice-go/common"
	"github.com/refractionPOINT/lc-service/lcservice-go/service/acker"
)

type HandlerGetter interface {
	GetHandler(eventType string) (common.ServiceCallback, bool)
}

type requestHandlerResolver struct {
	handlerGetter HandlerGetter
}

func NewResolver(handlerGetter HandlerGetter) requestHandlerResolver {
	return requestHandlerResolver{
		handlerGetter: handlerGetter,
	}
}

func (r *requestHandlerResolver) GetType() string {
	return "request"
}

func (r *requestHandlerResolver) Parse(requestEvent common.RequestEvent) (common.Dict, error) {
	return requestEvent.Data, nil
}

func (r *requestHandlerResolver) Get(requestEvent common.RequestEvent) common.ServiceCallback {
	// Unlike the Python implementation, we will not perform validation
	// of the incoming parameters based on the schema in the Descriptor.
	// Instead we will leave that task to the user by using `DictToStruct`
	// to facilitate Marshaling and validation.
	// TODO revisit this, maybe we can at least validate part of it.

	// Get the relevant handler.
	handler, found := r.handlerGetter.GetHandler(requestEvent.Type)
	if !found {
		return nil
	}
	return handler
}

func (r *requestHandlerResolver) PreHandlerHook(request common.Request, acker acker.RequestAcker) error {
	return nil
}
