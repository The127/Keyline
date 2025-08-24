package repositories

import (
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestApplicationFilter(t *testing.T) {
	// arrange
	f := NewApplicationFilter()
	name := "name"
	id := uuid.New()
	virtualServerId := uuid.New()

	// act
	f = f.Name(name)
	f = f.Id(id)
	f = f.VirtualServerId(virtualServerId)

	// assert
	assert.Equal(t, &name, f.name)
	assert.Equal(t, &id, f.id)
	assert.Equal(t, &virtualServerId, f.virtualServerId)
}
