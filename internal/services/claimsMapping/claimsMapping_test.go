package claimsMapping

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultMapping_OnlyMapsRoles(t *testing.T) {
	t.Parallel()

	claims := defaultMapping(Params{
		Roles:            []string{"system:admin"},
		ApplicationRoles: []string{"editor"},
		Username:         "alice",
		Scopes:           []string{"openid", "profile", "email"},
	})

	assert.Equal(t, map[string]any{
		"roles":             []string{"system:admin"},
		"application_roles": []string{"editor"},
	}, claims)
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
