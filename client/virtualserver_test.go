package client

import (
	"encoding/json"
	"github.com/The127/Keyline/api"
	"github.com/The127/Keyline/utils"
	"net/http"
	"net/http/httptest"
	"testing"

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
	request := api.CreateVirtualServerRequestDto{
		Name:                    "name",
		DisplayName:             "Display Name",
		EnableRegistration:      false,
		Require2fa:              false,
		PrimarySigningAlgorithm: utils.Ptr("EdDSA"),
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.Equal(http.MethodPost, r.Method)
		s.Equal("/api/virtual-servers", r.URL.Path)

		var requestDto api.CreateVirtualServerRequestDto
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
	response := api.GetVirtualServerResponseDto{
		DisplayName:              "Display Name",
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
	response := api.GetVirtualServerListResponseDto{
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
	request := PatchVirtualServerInput{
		DisplayName:              utils.Ptr("New display name"),
		EnableRegistration:       utils.Ptr(true),
		Require2fa:               utils.Ptr(false),
		RequireEmailVerification: nil,
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.Equal(http.MethodPatch, r.Method)
		s.Equal("/api/virtual-servers/test", r.URL.Path)

		var requestDto PatchVirtualServerInput
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

func (s *VirtualServerClientSuite) TestPatch_WithSigningAlgorithms() {
	// arrange
	additional := []string{"RS256"}
	request := PatchVirtualServerInput{
		PrimarySigningAlgorithm:     utils.Ptr("EdDSA"),
		AdditionalSigningAlgorithms: &additional,
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var requestDto PatchVirtualServerInput
		err := json.NewDecoder(r.Body).Decode(&requestDto)
		s.NoError(err)
		s.NotNil(requestDto.PrimarySigningAlgorithm)
		s.Equal("EdDSA", *requestDto.PrimarySigningAlgorithm)
		s.NotNil(requestDto.AdditionalSigningAlgorithms)
		s.Equal([]string{"RS256"}, *requestDto.AdditionalSigningAlgorithms)

		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	testee := NewClient(server.URL, "test").VirtualServer()

	// act
	err := testee.Patch(s.T().Context(), request)

	// assert
	s.Require().NoError(err)
}

func (s *VirtualServerClientSuite) TestGet_ReturnsSigningAlgorithms() {
	// arrange
	response := api.GetVirtualServerResponseDto{
		DisplayName:                 "Display Name",
		PrimarySigningAlgorithm:     "EdDSA",
		AdditionalSigningAlgorithms: []string{"RS256"},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		err := json.NewEncoder(w).Encode(response)
		s.NoError(err)
	}))
	defer server.Close()

	testee := NewClient(server.URL, "test").VirtualServer()

	// act
	responseDto, err := testee.Get(s.T().Context())

	// assert
	s.Require().NoError(err)
	s.Equal("EdDSA", responseDto.PrimarySigningAlgorithm)
	s.Equal([]string{"RS256"}, responseDto.AdditionalSigningAlgorithms)
}
