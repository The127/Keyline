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
	"github.com/google/uuid"
	"golang.org/x/oauth2"
)

// ServiceUserTokenSource implements oauth2.TokenSource via Keyline's RFC 7523
// JWT bearer grant, signing a short-lived JWT with a service user's Ed25519 private key.
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

	now := time.Now()
	claims := jwt.MapClaims{
		"aud": fmt.Sprintf("%s/oidc/%s", s.KeylineURL, s.VirtualServer),
		"iss": s.Username,
		"sub": s.Username,
		"iat": now.Unix(),
		"exp": now.Add(time.Minute).Unix(),
		"jti": uuid.NewString(),
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodEdDSA, claims)
	tok.Header["kid"] = s.Kid

	signed, err := tok.SignedString(rawKey)
	if err != nil {
		return nil, fmt.Errorf("signing JWT: %w", err)
	}

	httpClient := &http.Client{Timeout: 10 * time.Second}
	resp, err := httpClient.PostForm(
		fmt.Sprintf("%s/oidc/%s/token", s.KeylineURL, s.VirtualServer),
		url.Values{
			"grant_type": {"urn:ietf:params:oauth:grant-type:jwt-bearer"},
			"assertion":  {signed},
			"client_id":  {s.Application},
			"scope":      {"openid profile email"},
		},
	)
	if err != nil {
		return nil, fmt.Errorf("token request: %w", err)
	}
	defer resp.Body.Close() //nolint:errcheck

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("token endpoint returned %d", resp.StatusCode)
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
