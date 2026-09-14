package repositories

import (
	"testing"

	"github.com/The127/Keyline/utils"

	"github.com/stretchr/testify/require"
)

func TestValidateRedirectUriAcceptsWebAndNativeAppUris(t *testing.T) {
	t.Parallel()

	for _, uri := range []string{
		"https://app.example.com/callback",
		"http://localhost:8000/callback",
		"http://127.0.0.1:8000/callback",
		"com.example.app:/callback",
	} {
		t.Run(uri, func(t *testing.T) {
			// act
			err := ValidateRedirectUri(uri)

			// assert
			require.NoError(t, err)
		})
	}
}

func TestValidateRedirectUriRefusesOtherUris(t *testing.T) {
	t.Parallel()

	for _, uri := range []string{
		"javascript:alert(document.domain)",
		"JavaScript:alert(document.domain)",
		"data:text/html,<script>alert(document.domain)</script>",
		"vbscript:msgbox(1)",
		"file:///etc/passwd",
		"myapp://callback",
		"/relative/callback",
		"https://app.example.com/callback#fragment",
		"https://:443/callback",
		"com.example.app:",
		"",
	} {
		t.Run(uri, func(t *testing.T) {
			// act
			err := ValidateRedirectUri(uri)

			// assert
			require.ErrorIs(t, err, utils.ErrHttpBadRequest)
		})
	}
}
