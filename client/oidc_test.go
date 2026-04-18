package client

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/The127/Keyline/api"
	"github.com/stretchr/testify/suite"
)

type OidcClientSuite struct {
	suite.Suite
}

func TestOidcClientSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(OidcClientSuite))
}

func (s *OidcClientSuite) TestBeginDeviceFlow_HappyPath() {
	expected := api.DeviceAuthorizationResponse{
		DeviceCode:              "device-code-abc",
		UserCode:                "ABCD-EFGH",
		VerificationUri:         "http://localhost/oidc/test/activate",
		VerificationUriComplete: "http://localhost/oidc/test/activate?user_code=ABCD-EFGH",
		ExpiresIn:               600,
		Interval:                5,
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.Equal(http.MethodPost, r.Method)
		s.Equal("/oidc/test/device", r.URL.Path)
		s.Equal("application/x-www-form-urlencoded", r.Header.Get("Content-Type"))

		s.NoError(r.ParseForm())
		s.Equal("my-app", r.Form.Get("client_id"))
		s.Equal("openid profile", r.Form.Get("scope"))

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(expected)
	}))
	defer server.Close()

	testee := NewClient(server.URL, "test").Oidc()

	result, err := testee.BeginDeviceFlow(s.T().Context(), "my-app", "openid profile")

	s.Require().NoError(err)
	s.Equal(expected, result)
}

func (s *OidcClientSuite) TestBeginDeviceFlow_RejectsUnknownApp() {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"error":             "invalid_client",
			"error_description": "application not found",
		})
	}))
	defer server.Close()

	testee := NewClient(server.URL, "test").Oidc()

	_, err := testee.BeginDeviceFlow(s.T().Context(), "nonexistent", "openid")

	s.Require().Error(err)
}

func (s *OidcClientSuite) TestPollDeviceToken_HappyPath() {
	expected := api.DeviceTokenResponse{
		TokenType:    "Bearer",
		IdToken:      "id-token-value",
		AccessToken:  "access-token-value",
		RefreshToken: "refresh-token-value",
		Scope:        "openid",
		ExpiresIn:    3600,
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.Equal(http.MethodPost, r.Method)
		s.Equal("/oidc/test/token", r.URL.Path)
		s.Equal("application/x-www-form-urlencoded", r.Header.Get("Content-Type"))

		s.NoError(r.ParseForm())
		s.Equal("urn:ietf:params:oauth:grant-type:device_code", r.Form.Get("grant_type"))
		s.Equal("my-app", r.Form.Get("client_id"))
		s.Equal("device-code-abc", r.Form.Get("device_code"))

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(expected)
	}))
	defer server.Close()

	testee := NewClient(server.URL, "test").Oidc()

	result, err := testee.PollDeviceToken(s.T().Context(), "my-app", "device-code-abc")

	s.Require().NoError(err)
	s.Equal(expected, result)
}

func (s *OidcClientSuite) TestPollDeviceToken_AuthorizationPending() {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "authorization_pending"})
	}))
	defer server.Close()

	testee := NewClient(server.URL, "test").Oidc()

	_, err := testee.PollDeviceToken(s.T().Context(), "my-app", "device-code-abc")

	s.ErrorIs(err, ErrAuthorizationPending)
}

func (s *OidcClientSuite) TestPollDeviceToken_AccessDenied() {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "access_denied"})
	}))
	defer server.Close()

	testee := NewClient(server.URL, "test").Oidc()

	_, err := testee.PollDeviceToken(s.T().Context(), "my-app", "device-code-abc")

	s.ErrorIs(err, ErrAccessDenied)
}

func (s *OidcClientSuite) TestPollDeviceToken_ExpiredToken() {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "expired_token"})
	}))
	defer server.Close()

	testee := NewClient(server.URL, "test").Oidc()

	_, err := testee.PollDeviceToken(s.T().Context(), "my-app", "device-code-abc")

	s.ErrorIs(err, ErrExpiredToken)
}
