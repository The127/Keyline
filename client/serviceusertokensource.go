package client

import (
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/oauth2"
)

// WithServiceUser authenticates every request as the given service user through Keyline's RFC 7523 JWT bearer grant.
func WithServiceUser(privateKeyPem string, kid string, username string, application string) TransportOptions {
	return func(transport *Transport) {
		tokenSource := newServiceUserTokenSource(privateKeyPem, kid, username, application, func(endpoint string) string {
			return oidcUrl(transport.baseURL, transport.virtualServer, endpoint)
		})
		WithOidc(tokenSource)(transport)
	}
}

// NewServiceUserTokenSource logs the given service user in through Keyline's RFC 7523 JWT bearer grant and hands out the access token itself.
//
// WithServiceUser is the way to call Keyline's own API as a service user. This constructor exists for
// programs that need the token as a value, because they present it to another service that trusts
// Keyline as its issuer. Both share one login so a change to the grant reaches every caller.
func NewServiceUserTokenSource(keylineURL string, virtualServer string, privateKeyPem string, kid string, username string, application string) oauth2.TokenSource {
	return newServiceUserTokenSource(privateKeyPem, kid, username, application, func(endpoint string) string {
		return oidcUrl(keylineURL, virtualServer, endpoint)
	})
}

func newServiceUserTokenSource(privateKeyPem string, kid string, username string, application string, oidcEndpoint func(endpoint string) string) *serviceUserTokenSource {
	return &serviceUserTokenSource{
		oidcEndpoint:  oidcEndpoint,
		privateKeyPem: privateKeyPem,
		kid:           kid,
		username:      username,
		application:   application,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
			CheckRedirect: func(*http.Request, []*http.Request) error {
				return http.ErrUseLastResponse
			},
		},
	}
}

type serviceUserTokenSource struct {
	oidcEndpoint  func(endpoint string) string
	privateKeyPem string
	kid           string
	username      string
	application   string
	httpClient    *http.Client

	mu     sync.Mutex
	cached *oauth2.Token
}

func (tokenSource *serviceUserTokenSource) Token() (*oauth2.Token, error) {
	tokenSource.mu.Lock()
	defer tokenSource.mu.Unlock()

	if tokenSource.cached != nil && tokenSource.cached.Valid() {
		return tokenSource.cached, nil
	}

	issuer, err := tokenSource.discoverIssuer()
	if err != nil {
		return nil, err
	}

	signed, err := tokenSource.signAssertion(issuer)
	if err != nil {
		return nil, err
	}

	resp, err := tokenSource.httpClient.PostForm(
		tokenSource.oidcEndpoint("/token"),
		url.Values{
			"grant_type": {"urn:ietf:params:oauth:grant-type:jwt-bearer"},
			"assertion":  {signed},
			"client_id":  {tokenSource.application},
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

	tokenSource.cached = &oauth2.Token{
		AccessToken: body.AccessToken,
		Expiry:      time.Now().Add(time.Duration(body.ExpiresIn) * time.Second),
	}
	return tokenSource.cached, nil
}

func (tokenSource *serviceUserTokenSource) discoverIssuer() (string, error) {
	resp, err := tokenSource.httpClient.Get(tokenSource.oidcEndpoint("/.well-known/openid-configuration"))
	if err != nil {
		return "", fmt.Errorf("discovery request: %w", err)
	}
	defer resp.Body.Close() //nolint:errcheck

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("discovery endpoint returned %d", resp.StatusCode)
	}

	var body struct {
		Issuer string `json:"issuer"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return "", fmt.Errorf("decoding discovery response: %w", err)
	}
	if body.Issuer == "" {
		return "", fmt.Errorf("discovery response has no issuer")
	}

	return body.Issuer, nil
}

func (tokenSource *serviceUserTokenSource) signAssertion(issuer string) (string, error) {
	block, _ := pem.Decode([]byte(tokenSource.privateKeyPem))
	if block == nil {
		return "", fmt.Errorf("failed to decode private key PEM")
	}
	rawKey, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return "", fmt.Errorf("parsing private key: %w", err)
	}

	now := time.Now()
	claims := jwt.MapClaims{
		"aud": issuer,
		"iss": tokenSource.username,
		"sub": tokenSource.username,
		"iat": now.Unix(),
		"exp": now.Add(time.Minute).Unix(),
		"jti": uuid.NewString(),
	}
	assertion := jwt.NewWithClaims(jwt.SigningMethodEdDSA, claims)
	assertion.Header["kid"] = tokenSource.kid

	signed, err := assertion.SignedString(rawKey)
	if err != nil {
		return "", fmt.Errorf("signing assertion: %w", err)
	}
	return signed, nil
}

func oidcUrl(baseURL string, virtualServer string, endpoint string) string {
	return fmt.Sprintf("%s/oidc/%s%s", strings.TrimSuffix(baseURL, "/"), virtualServer, endpoint)
}
