package mediator

import (
	"context"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestRequestHandlerGetsCalled(t *testing.T) {
	// arrange
	m := NewMediator()
	RegisterHandler(m, func(ctx context.Context, request string) (string, error) {
		return "foo", nil
	})

	// act
	response, err := Send[string](t.Context(), m, "bar")

	// assert
	assert.NoError(t, err)
	assert.Equal(t, "foo", response)
}

func TestBehaviourCalled(t *testing.T) {
	// arrange
	m := NewMediator()
	RegisterHandler(m, func(ctx context.Context, request string) (string, error) {
		return "foo", nil
	})

	behaviourCalled := false
	RegisterBehaviour(m, func(ctx context.Context, request string, next Next) {
		behaviourCalled = true
		next()
	})

	// act
	response, err := Send[string](t.Context(), m, "bar")

	// assert
	assert.NoError(t, err)
	assert.Equal(t, "foo", response)
	assert.True(t, behaviourCalled)
}

func TestEventHandlerGetsCalled(t *testing.T) {
	// arrange
	m := NewMediator()
	evtHandlerCalled := false
	RegisterEventHandler(m, func(ctx context.Context, evt string) error {
		evtHandlerCalled = true
		return nil
	})

	// act
	err := SendEvent(t.Context(), m, "foo")

	// assert
	assert.NoError(t, err)
	assert.True(t, evtHandlerCalled)
}
