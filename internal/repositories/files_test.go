package repositories

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestFileFilter(t *testing.T) {
	// arrange
	f := NewFileFilter()
	id := uuid.New()

	// act
	f = f.Id(id)

	// assert
	assert.Equal(t, &id, f.id)
}
