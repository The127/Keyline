package client

import (
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/oauth2"
)

// ServiceUserTokenSource implements oauth2.TokenSource via Keyline's RFC 8693
// token exchange, signing a short-lived JWT with a service user's Ed25519 private key.
type ServiceUserTokenSource struct {
	KeylineURL    string
	VirtualServer string
	PrivKeyPEM    string
	Kid           string
	Username      string
	Application   string

	mu     sync.Mutex
	cached *oauth2.Token
}

func (s *ServiceUserTokenSource) Token() (*oauth2.Token, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.cached != nil && s.cached.Valid() {
		return s.cached, nil
	}

	block, _ := pem.Decode([]byte(s.PrivKeyPEM))
	if block == nil {
		return nil, fmt.Errorf("failed to decode private key PEM")
	}
	rawKey, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("parsing private key: %w", err)
	}

	claims := jwt.MapClaims{
		"aud":    s.Application,
		"iss":    s.Username,
		"sub":    s.Username,
		"scopes": "openid profile email",
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodEdDSA, claims)
	tok.Header["kid"] = s.Kid

	signed, err := tok.SignedString(rawKey)
	if err != nil {
		return nil, fmt.Errorf("signing JWT: %w", err)
	}

	resp, err := http.PostForm(
		fmt.Sprintf("%s/oidc/%s/token", s.KeylineURL, s.VirtualServer),
		url.Values{
			"grant_type":         {"urn:ietf:params:oauth:grant-type:token-exchange"},
			"subject_token":      {signed},
			"subject_token_type": {"urn:ietf:params:oauth:token-type:access_token"},
		},
	)
	if err != nil {
		return nil, fmt.Errorf("token exchange request: %w", err)
	}
	defer resp.Body.Close() //nolint:errcheck

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("token exchange returned %d", resp.StatusCode)
	}

	var body struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return nil, fmt.Errorf("decoding token response: %w", err)
	}

	s.cached = &oauth2.Token{
		AccessToken: body.AccessToken,
		Expiry:      time.Now().Add(time.Duration(body.ExpiresIn) * time.Second),
	}
	return s.cached, nil
}
