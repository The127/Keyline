package mediator

import (
	"Keyline/logging"
	"Keyline/utils"
	"context"
	"reflect"
)

type Mediator struct {
	handlers map[reflect.Type]handlerInfo
}

type handlerInfo struct {
	requestType  reflect.Type
	responseType reflect.Type
	handlerFunc  func(ctx context.Context, request any) (any, error)
}

type HandlerFunc[TRequest any, TResponse any] func(ctx context.Context, request TRequest) (TResponse, error)

func NewMediator() *Mediator {
	return &Mediator{
		handlers: make(map[reflect.Type]handlerInfo),
	}
}

func RegisterHandler[TRequest any, TResponse any](m *Mediator, handler HandlerFunc[TRequest, TResponse]) {
	m.handlers[utils.TypeOf[TRequest]()] = handlerInfo{
		requestType:  utils.TypeOf[TRequest](),
		responseType: utils.TypeOf[TResponse](),
		handlerFunc: func(ctx context.Context, request any) (any, error) {
			return handler(ctx, request.(TRequest))
		},
	}
}

func Send[TResponse any](ctx context.Context, m *Mediator, request any) (TResponse, error) {
	requestType := reflect.TypeOf(request)

	info, ok := m.handlers[requestType]
	if !ok {
		logging.Logger.Fatalf("Could not find any registered handler for %s", requestType.Name())
	}

	responseType := utils.TypeOf[TResponse]()
	if info.responseType != responseType {
		logging.Logger.Fatalf("Wrong response type %s was used for request %s, expected response type %s",
			responseType.Name(),
			requestType.Name(),
			info.responseType.Name())
	}

	response, err := info.handlerFunc(ctx, request)
	return response.(TResponse), err
}
