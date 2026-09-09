package claimsMapping

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultMapping_ProfileScopeAddsPreferredUsername(t *testing.T) {
	t.Parallel()

	claims := defaultMapping(Params{
		Roles:    []string{"system:admin"},
		Username: "alice",
		Scopes:   []string{"openid", "profile"},
	})

	assert.Equal(t, "alice", claims["preferred_username"])
	assert.Equal(t, []string{"system:admin"}, claims["roles"])
}

func TestDefaultMapping_WithoutProfileScopeOmitsPreferredUsername(t *testing.T) {
	t.Parallel()

	claims := defaultMapping(Params{
		Username: "alice",
		Scopes:   []string{"openid", "email"},
	})

	assert.NotContains(t, claims, "preferred_username")
}

func TestCustomScript_CanReadUsernameAndScopes(t *testing.T) {
	t.Parallel()

	script := `({ user: username, hasProfile: scopes.includes("profile") })`
	claims, err := (&claimsMapper{}).runCustomClaimsMappingScript(&script, Params{
		Username: "alice",
		Scopes:   []string{"openid", "profile"},
	})

	require.NoError(t, err)
	assert.Equal(t, "alice", claims["user"])
	assert.Equal(t, true, claims["hasProfile"])
}
