package client

import (
	"fmt"
	"net/http"

	"golang.org/x/oauth2"
)

type OIDCRoundTripper struct {
	next        http.RoundTripper
	tokenSource oauth2.TokenSource
}

func NewOIDCRoundTripper(next http.RoundTripper, tokenSource oauth2.TokenSource) *OIDCRoundTripper {
	return &OIDCRoundTripper{
		next:        next,
		tokenSource: tokenSource,
	}
}

func (rt *OIDCRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	// Get a valid access token (auto-refresh handled by oauth2)
	token, err := rt.tokenSource.Token()
	if err != nil {
		return nil, fmt.Errorf("getting OIDC token: %w", err)
	}

	req = req.Clone(req.Context())
	req.Header.Set("Authorization", "Bearer "+token.AccessToken)

	return rt.next.RoundTrip(req)
}
