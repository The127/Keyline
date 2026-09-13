package client

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/suite"
)

type ServiceUserTokenSourceSuite struct {
	suite.Suite
}

func TestServiceUserTokenSourceSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(ServiceUserTokenSourceSuite))
}

func (s *ServiceUserTokenSourceSuite) TestLogsInWithTheIssuerAdvertisedByTheServer() {
	// arrange
	publicKey, privateKeyPem := s.newKeyPair()
	keyline := s.newFakeKeyline(publicKey, 300)
	defer keyline.server.Close()

	testee := NewClient(keyline.server.URL, "test", WithServiceUser(privateKeyPem, "kid-1", "svc", "my-app")).Oidc()

	// act
	_, err := testee.BeginDeviceFlow(s.T().Context(), "my-app", "openid")

	// assert
	s.Require().NoError(err)
	s.Equal("Bearer issued-token", keyline.authorization)
}

func (s *ServiceUserTokenSourceSuite) TestLogsInThroughABareCustomHttpClient() {
	// arrange
	publicKey, privateKeyPem := s.newKeyPair()
	keyline := s.newFakeKeyline(publicKey, 300)
	defer keyline.server.Close()

	testee := NewClient(keyline.server.URL, "test", WithClient(&http.Client{}), WithServiceUser(privateKeyPem, "kid-1", "svc", "my-app")).Oidc()

	// act
	_, err := testee.BeginDeviceFlow(s.T().Context(), "my-app", "openid")

	// assert
	s.Require().NoError(err)
	s.Equal("Bearer issued-token", keyline.authorization)
}

func (s *ServiceUserTokenSourceSuite) TestAsksForTheIssuerOnEveryLogin() {
	// arrange
	publicKey, privateKeyPem := s.newKeyPair()
	keyline := s.newFakeKeyline(publicKey, 0)
	defer keyline.server.Close()

	testee := NewClient(keyline.server.URL, "test", WithServiceUser(privateKeyPem, "kid-1", "svc", "my-app")).Oidc()
	_, err := testee.BeginDeviceFlow(s.T().Context(), "my-app", "openid")
	s.Require().NoError(err)

	// act
	_, err = testee.BeginDeviceFlow(s.T().Context(), "my-app", "openid")

	// assert
	s.Require().NoError(err)
	s.Equal(2, keyline.logins)
	s.Equal(2, keyline.discoveries)
}

func (s *ServiceUserTokenSourceSuite) TestHandsOutTheAccessTokenWithoutATransport() {
	// arrange
	publicKey, privateKeyPem := s.newKeyPair()
	keyline := s.newFakeKeyline(publicKey, 300)
	defer keyline.server.Close()

	testee := NewServiceUserTokenSource(keyline.server.URL, "test", privateKeyPem, "kid-1", "svc", "my-app")

	// act
	token, err := testee.Token()

	// assert
	s.Require().NoError(err)
	s.Equal("issued-token", token.AccessToken)
	s.Equal(1, keyline.logins)
}

func (s *ServiceUserTokenSourceSuite) TestToleratesATrailingSlashOnTheKeylineURL() {
	// arrange
	publicKey, privateKeyPem := s.newKeyPair()
	keyline := s.newFakeKeyline(publicKey, 300)
	defer keyline.server.Close()

	testee := NewServiceUserTokenSource(keyline.server.URL+"/", "test", privateKeyPem, "kid-1", "svc", "my-app")

	// act
	token, err := testee.Token()

	// assert
	s.Require().NoError(err)
	s.Equal("issued-token", token.AccessToken)
}

type fakeKeyline struct {
	server        *httptest.Server
	authorization string
	discoveries   int
	logins        int
}

func (s *ServiceUserTokenSourceSuite) newFakeKeyline(publicKey ed25519.PublicKey, expiresIn int) *fakeKeyline {
	const advertisedIssuer = "https://public.example/oidc/test"
	keyline := &fakeKeyline{}

	keyline.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/oidc/test/.well-known/openid-configuration":
			keyline.discoveries++
			_ = json.NewEncoder(w).Encode(map[string]string{"issuer": advertisedIssuer})

		case "/oidc/test/token":
			keyline.logins++
			s.NoError(r.ParseForm())
			s.Equal("urn:ietf:params:oauth:grant-type:jwt-bearer", r.Form.Get("grant_type"))
			s.Equal("my-app", r.Form.Get("client_id"))
			s.Equal("openid profile email", r.Form.Get("scope"))

			claims := jwt.MapClaims{}
			token, err := jwt.ParseWithClaims(r.Form.Get("assertion"), claims, func(*jwt.Token) (any, error) { return publicKey, nil })
			if !s.NoError(err) {
				return
			}
			s.Equal("kid-1", token.Header["kid"])
			s.Equal("svc", claims["iss"])
			s.Equal("svc", claims["sub"])
			s.Equal(advertisedIssuer, claims["aud"])
			s.NotEmpty(claims["jti"])
			s.NotNil(claims["exp"])

			_ = json.NewEncoder(w).Encode(map[string]any{"access_token": "issued-token", "expires_in": expiresIn})

		case "/oidc/test/device":
			keyline.authorization = r.Header.Get("Authorization")
			_ = json.NewEncoder(w).Encode(map[string]any{})

		default:
			s.Failf("unexpected request", "%s %s", r.Method, r.URL.Path)
		}
	}))

	return keyline
}

func (s *ServiceUserTokenSourceSuite) newKeyPair() (ed25519.PublicKey, string) {
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	s.Require().NoError(err)

	der, err := x509.MarshalPKCS8PrivateKey(privateKey)
	s.Require().NoError(err)

	return publicKey, string(pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: der}))
}
