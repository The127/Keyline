package repositories

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestSessionFilter(t *testing.T) {
	// arrange
	f := NewSessionFilter()
	id := uuid.New()
	userId := uuid.New()
	virtualServerId := uuid.New()

	// act
	f = f.Id(id)
	f = f.UserId(userId)
	f = f.VirtualServerId(virtualServerId)

	// assert
	assert.Equal(t, &id, f.id)
	assert.Equal(t, &userId, f.userId)
	assert.Equal(t, &virtualServerId, f.virtualServerId)
}
