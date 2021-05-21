package service

type requestHandlerResolver struct {
	cs *CoreService
}

func (r *requestHandlerResolver) GetType() string {
	return "request"
}

func (r *requestHandlerResolver) Parse(requestEvent RequestEvent) (Dict, error) {
	return requestEvent.Data, nil
}

func (r *requestHandlerResolver) Get(requestEvent RequestEvent) ServiceCallback {
	// Unlike the Python implementation, we will not perform validation
	// of the incoming parameters based on the schema in the Descriptor.
	// Instead we will leave that task to the user by using `DictToStruct`
	// to facilitate Marshaling and validation.
	// TODO revisit this, maybe we can at least validate part of it.

	// Get the relevant handler.
	handler, found := r.cs.getHandler(requestEvent.Type)
	if !found {
		return nil
	}
	return handler
}

func (r *requestHandlerResolver) PreHandlerHook(request Request) error {
	return nil
}
