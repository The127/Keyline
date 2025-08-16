package mediator

import (
	"Keyline/logging"
	"Keyline/utils"
	"context"
	"reflect"
)

type Mediator struct {
	handlers      map[reflect.Type]handlerInfo
	behaviours    map[reflect.Type][]behaviourInfo
	eventHandlers map[reflect.Type][]eventHandlerInfo
}

type eventHandlerInfo struct {
	eventType        reflect.Type
	eventHandlerFunc func(ctx context.Context, evt any) error
}

type EventHandlerFunc[TEvent any] func(ctx context.Context, evt TEvent) error

func RegisterEventHandler[TEvent any](m *Mediator, eventHandler EventHandlerFunc[TEvent]) {
	eventType := utils.TypeOf[TEvent]()

	eventHandlers, ok := m.eventHandlers[eventType]
	if !ok {
		eventHandlers = []eventHandlerInfo{}
	}

	eventHandlers = append(eventHandlers, eventHandlerInfo{
		eventType: eventType,
		eventHandlerFunc: func(ctx context.Context, evt any) error {
			return eventHandler(ctx, evt.(TEvent))
		},
	})

	m.eventHandlers[eventType] = eventHandlers
}

type Next func()

type behaviourInfo struct {
	requestType   reflect.Type
	behaviourFunc func(ctx context.Context, request any, next Next)
}

type handlerInfo struct {
	requestType  reflect.Type
	responseType reflect.Type
	handlerFunc  func(ctx context.Context, request any) (any, error)
}

type HandlerFunc[TRequest any, TResponse any] func(ctx context.Context, request TRequest) (TResponse, error)

func NewMediator() *Mediator {
	return &Mediator{
		handlers:      make(map[reflect.Type]handlerInfo),
		behaviours:    make(map[reflect.Type][]behaviourInfo),
		eventHandlers: make(map[reflect.Type][]eventHandlerInfo),
	}
}

type BehaviourFunc[TRequest any] func(ctx context.Context, request TRequest, next Next)

func RegisterBehaviour[TRequest any](m *Mediator, behaviour BehaviourFunc[TRequest]) {
	requestType := utils.TypeOf[TRequest]()

	behaviours, ok := m.behaviours[requestType]
	if !ok {
		behaviours = make([]behaviourInfo, 0)
	}

	behaviours = append(behaviours, behaviourInfo{
		requestType: requestType,
		behaviourFunc: func(ctx context.Context, request any, next Next) {
			behaviour(ctx, request.(TRequest), next)
		},
	})

	m.behaviours[requestType] = behaviours
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

func SendEvent[TEvent any](ctx context.Context, m *Mediator, evt TEvent) error {
	eventType := utils.TypeOf[TEvent]()

	eventHandlers, ok := m.eventHandlers[eventType]
	if !ok {
		return nil
	}

	for _, eventHandler := range eventHandlers {
		err := eventHandler.eventHandlerFunc(ctx, evt)
		if err != nil {
			return err
		}
	}

	return nil
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

	var step Next
	var response any
	var err error

	step = func() {
		response, err = info.handlerFunc(ctx, request)
	}

	behaviours, ok := m.behaviours[requestType]
	if !ok {
		behaviours = make([]behaviourInfo, 0)
	}

	for i := len(behaviours) - 1; i >= 0; i-- {
		behaviour := behaviours[i]
		prev := step
		step = func() {
			behaviour.behaviourFunc(ctx, request, prev)
		}
	}

	step()

	return response.(TResponse), err
}
