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
