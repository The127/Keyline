package handlers

import (
	"testing"

	"github.com/The127/Keyline/internal/repositories"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestUserinfoClaims_ProfileAndEmailScopes(t *testing.T) {
	t.Parallel()

	user := repositories.NewUser("alice", "Alice Example", "alice@example.com", uuid.New())
	user.SetEmailVerified(true)

	claims := userinfoClaims(user, []string{"openid", "profile", "email"})

	assert.Equal(t, map[string]any{
		"email":              "alice@example.com",
		"email_verified":     true,
		"name":               "Alice Example",
		"preferred_username": "alice",
	}, claims)
}

func TestUserinfoClaims_OnlyEmailScope(t *testing.T) {
	t.Parallel()

	user := repositories.NewUser("alice", "Alice Example", "alice@example.com", uuid.New())

	claims := userinfoClaims(user, []string{"openid", "email"})

	assert.Equal(t, map[string]any{
		"email":          "alice@example.com",
		"email_verified": false,
	}, claims)
}

func TestUserinfoClaims_OpenidOnly(t *testing.T) {
	t.Parallel()

	user := repositories.NewUser("alice", "Alice Example", "alice@example.com", uuid.New())

	claims := userinfoClaims(user, []string{"openid"})

	assert.Empty(t, claims)
}
