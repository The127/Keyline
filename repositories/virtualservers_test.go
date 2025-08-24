package repositories

import (
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestVirtualServerFilter(t *testing.T) {
	// arrange
	f := NewVirtualServerFilter()
	id := uuid.New()
	name := "name"

	// act
	f = f.Id(id)
	f = f.Name(name)

	// assert
	assert.Equal(t, &id, f.id)
	assert.Equal(t, &name, f.name)
}
