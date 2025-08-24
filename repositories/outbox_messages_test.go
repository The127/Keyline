package repositories

import (
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestOutboxFilter(t *testing.T) {
	// arrange
	f := NewOutboxMessageFilter()
	id := uuid.New()

	// act
	f = f.Id(id)

	// assert
	assert.Equal(t, &id, f.id)
}
