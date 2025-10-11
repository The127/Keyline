package mediator

import (
	"Keyline/internal/logging"
	"Keyline/utils"
	"context"
	"reflect"
)

type Mediator interface {
	Send(ctx context.Context, request any, requestType reflect.Type, responseType reflect.Type) (any, error)
	SendEvent(ctx context.Context, evt any, eventType reflect.Type) error
}

type mediator struct {
	handlers      map[reflect.Type]handlerInfo
	behaviours    []behaviourInfo
	eventHandlers map[reflect.Type][]eventHandlerInfo
}

type eventHandlerInfo struct {
	eventType        reflect.Type
	eventHandlerFunc func(ctx context.Context, evt any) error
}

type EventHandlerFunc[TEvent any] func(ctx context.Context, evt TEvent) error

func RegisterEventHandler[TEvent any](m *mediator, eventHandler EventHandlerFunc[TEvent]) {
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

type Next func() error

type behaviourInfo struct {
	requestType   reflect.Type
	behaviourFunc func(ctx context.Context, request any, next Next) error
}

type handlerInfo struct {
	requestType  reflect.Type
	responseType reflect.Type
	handlerFunc  func(ctx context.Context, request any) (any, error)
}

type HandlerFunc[TRequest any, TResponse any] func(ctx context.Context, request TRequest) (TResponse, error)

func NewMediator() *mediator {
	return &mediator{
		handlers:      make(map[reflect.Type]handlerInfo),
		behaviours:    make([]behaviourInfo, 0),
		eventHandlers: make(map[reflect.Type][]eventHandlerInfo),
	}
}

type BehaviourFunc[TRequest any] func(ctx context.Context, request TRequest, next Next) error

func RegisterBehaviour[TRequest any](m *mediator, behaviour BehaviourFunc[TRequest]) {
	requestType := utils.TypeOf[TRequest]()

	m.behaviours = append(m.behaviours, behaviourInfo{
		requestType: requestType,
		behaviourFunc: func(ctx context.Context, request any, next Next) error {
			return behaviour(ctx, request.(TRequest), next)
		},
	})
}

func RegisterHandler[TRequest any, TResponse any](m *mediator, handler HandlerFunc[TRequest, TResponse]) {
	m.handlers[utils.TypeOf[TRequest]()] = handlerInfo{
		requestType:  utils.TypeOf[TRequest](),
		responseType: utils.TypeOf[TResponse](),
		handlerFunc: func(ctx context.Context, request any) (any, error) {
			return handler(ctx, request.(TRequest))
		},
	}
}

func SendEvent[TEvent any](ctx context.Context, m Mediator, evt TEvent) error {
	eventType := utils.TypeOf[TEvent]()
	return m.SendEvent(ctx, evt, eventType)
}

func (m *mediator) SendEvent(ctx context.Context, evt any, eventType reflect.Type) error {
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

func Send[TResponse any](ctx context.Context, m Mediator, request any) (TResponse, error) {
	requestType := reflect.TypeOf(request)
	response, err := m.Send(ctx, request, requestType, utils.TypeOf[TResponse]())
	if response == nil {
		response = utils.Zero[TResponse]()
	}
	return response.(TResponse), err
}

func (m *mediator) Send(ctx context.Context, request any, requestType reflect.Type, responseType reflect.Type) (any, error) {
	info, ok := m.handlers[requestType]
	if !ok {
		logging.Logger.Fatalf("Could not find any registered handler for %s", requestType.Name())
	}

	if info.responseType != responseType {
		logging.Logger.Fatalf("Wrong response type %s was used for request %s, expected response type %s",
			responseType.Name(),
			requestType.Name(),
			info.responseType.Name())
	}

	var step Next
	var response any
	var err error

	step = func() error {
		response, err = info.handlerFunc(ctx, request)
		return err
	}

	behaviours := m.getBehaviours(requestType)

	for i := len(behaviours) - 1; i >= 0; i-- {
		behaviour := behaviours[i]
		prev := step
		step = func() error {
			return behaviour.behaviourFunc(ctx, request, prev)
		}
	}

	err = step()
	if err != nil {
		return nil, err
	}

	return response, err
}

func (m *mediator) getBehaviours(requestType reflect.Type) []behaviourInfo {
	result := make([]behaviourInfo, 0)

	for _, behaviour := range m.behaviours {
		if requestType.AssignableTo(behaviour.requestType) {
			result = append(result, behaviour)
		}
	}

	return result
}
