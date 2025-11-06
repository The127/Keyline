package client

import (
	"Keyline/internal/handlers"
	"Keyline/utils"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"
)

type VirtualServerClientSuite struct {
	suite.Suite
}

func TestVirtualServerClientSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(VirtualServerClientSuite))
}

func (s *VirtualServerClientSuite) TestCreate_HappyPath() {
	// arrange
	request := handlers.CreateVirtualServerRequestDto{
		Name:               "name",
		DisplayName:        "Display Name",
		EnableRegistration: false,
		Require2fa:         false,
		SigningAlgorithm:   utils.Ptr("EdDSA"),
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.Equal(http.MethodPost, r.Method)
		s.Equal("/api/virtual-servers", r.URL.Path)

		var requestDto handlers.CreateVirtualServerRequestDto
		err := json.NewDecoder(r.Body).Decode(&requestDto)
		s.NoError(err)
		s.Equal(request, requestDto)

		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	testee := NewClient(server.URL, "test").VirtualServer()

	// act
	err := testee.Create(s.T().Context(), request)

	// assert
	s.Require().NoError(err)
}

func (s *VirtualServerClientSuite) TestGet_HappyPath() {
	// arrange
	response := handlers.GetVirtualServerResponseDto{
		Id:                       uuid.New(),
		Name:                     "name",
		DisplayName:              "Display Name",
		SigningAlgorithm:         "EdDSA",
		Require2fa:               false,
		RequireEmailVerification: false,
		RegistrationEnabled:      false,
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.Equal(http.MethodGet, r.Method)
		s.Equal("/api/virtual-servers/test", r.URL.Path)

		err := json.NewEncoder(w).Encode(response)
		s.NoError(err)
	}))
	defer server.Close()

	testee := NewClient(server.URL, "test").VirtualServer()

	// act
	responseDto, err := testee.Get(s.T().Context())

	// assert
	s.Require().NoError(err)
	s.Equal(response, responseDto)
}

func (s *VirtualServerClientSuite) TestGetPublic_InfoHappyPath() {
	// arrange
	response := handlers.GetVirtualServerListResponseDto{
		Name:                "name",
		DisplayName:         "Display Name",
		RegistrationEnabled: false,
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.Equal(http.MethodGet, r.Method)
		s.Equal("/api/virtual-servers/test/public-info", r.URL.Path)

		err := json.NewEncoder(w).Encode(response)
		s.NoError(err)
	}))
	defer server.Close()

	testee := NewClient(server.URL, "test").VirtualServer()

	// act
	responseDto, err := testee.GetPublicInfo(s.T().Context())

	// assert
	s.Require().NoError(err)
	s.Equal(response, responseDto)
}

func (s *VirtualServerClientSuite) TestPatch_HappyPath() {
	// arrange
	request := handlers.PatchVirtualServerRequestDto{
		DisplayName:              utils.Ptr("New display name"),
		EnableRegistration:       utils.Ptr(true),
		Require2fa:               utils.Ptr(false),
		RequireEmailVerification: nil,
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.Equal(http.MethodPatch, r.Method)
		s.Equal("/api/virtual-servers/test", r.URL.Path)

		var requestDto handlers.PatchVirtualServerRequestDto
		err := json.NewDecoder(r.Body).Decode(&requestDto)
		s.NoError(err)
		s.Equal(request, requestDto)

		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	testee := NewClient(server.URL, "test").VirtualServer()

	// act
	err := testee.Patch(s.T().Context(), request)

	// assert
	s.Require().NoError(err)
}
