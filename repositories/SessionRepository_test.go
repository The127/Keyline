package repositories

import (
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestSessionFilter(t *testing.T) {
	// arrange
	f := NewSessionFilter()
	userId := uuid.New()
	virtualServerId := uuid.New()

	// act
	f = f.UserId(userId)
	f = f.VirtualServerId(virtualServerId)

	// assert
	assert.Equal(t, &userId, f.userId)
	assert.Equal(t, &virtualServerId, f.virtualServerId)
}
