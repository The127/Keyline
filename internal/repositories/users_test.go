package repositories

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestUserFilter(t *testing.T) {
	// arrange
	f := NewUserFilter()
	id := uuid.New()
	username := "username"
	virtualServerId := uuid.New()

	// act
	f = f.Id(id)
	f = f.Username(username)
	f = f.VirtualServerId(virtualServerId)

	// assert
	assert.Equal(t, &id, f.id)
	assert.Equal(t, &username, f.username)
	assert.Equal(t, &virtualServerId, f.virtualServerId)
}
