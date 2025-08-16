package repositories

import (
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestRoleFilter(t *testing.T) {
	// arrange
	f := NewRoleFilter()
	id := uuid.New()
	name := "name"

	// act
	f = f.Id(id)
	f = f.Name(name)

	// assert
	assert.Equal(t, &id, f.id)
	assert.Equal(t, &name, f.name)
}
