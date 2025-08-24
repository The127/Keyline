package repositories

import (
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestFoo(t *testing.T) {
	// arrange
	f := NewCredentialFilter()
	userId := uuid.New()

	// act
	f = f.UserId(userId)

	// assert
	assert.Equal(t, &userId, f.userId)
}
