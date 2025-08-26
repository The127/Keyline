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
	_type := CredentialType("type")

	// act
	f = f.UserId(userId)
	f = f.Type(_type)

	// assert
	assert.Equal(t, &userId, f.userId)
	assert.Equal(t, &_type, f._type)
}
