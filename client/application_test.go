package client

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/The127/Keyline/api"
	"github.com/The127/Keyline/utils"
	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"
)

type ApplicationClientSuite struct {
	suite.Suite
}

func TestApplicationClientSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(ApplicationClientSuite))
}

func (s *ApplicationClientSuite) TestCreateApplication_HappyPath() {
	// arrange
	request := api.CreateApplicationRequestDto{
		Name:           "applicationName",
		DisplayName:    "displayName",
		RedirectUris:   []string{"http://localhost:8080/callback"},
		PostLogoutUris: []string{"http://localhost:8080/logout"},
		Type:           "confidential",
	}

	response := api.CreateApplicationResponseDto{
		Id:     uuid.New(),
		Secret: utils.Ptr("secret"),
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.Equal(http.MethodPost, r.Method)
		s.Equal("/api/virtual-servers/test/projects/my-project/applications", r.URL.Path)

		var requestDto api.CreateApplicationRequestDto
		err := json.NewDecoder(r.Body).Decode(&requestDto)
		s.NoError(err)
		s.Equal(request, requestDto)

		err = json.NewEncoder(w).Encode(response)
		s.NoError(err)
	}))
	defer server.Close()

	testee := NewClient(server.URL, "test").Project().Application("my-project")

	// act
	responseDto, err := testee.Create(s.T().Context(), request)

	// assert
	s.Require().NoError(err)
	s.Equal(response, responseDto)
}

func (s *ApplicationClientSuite) TestListApplications_HappyPath() {
	// arrange
	requestParams := ListApplicationParams{
		Page: 1,
		Size: 11,
	}

	response := api.PagedApplicationsResponseDto{
		Items: []api.ListApplicationsResponseDto{
			{
				Id:                uuid.New(),
				Name:              "name",
				DisplayName:       "displayName",
				Type:              "public",
				SystemApplication: false,
			},
		},
		Pagination: &api.Pagination{
			Page:       1,
			Size:       11,
			TotalPages: 2,
			TotalItems: 22,
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.Equal(http.MethodGet, r.Method)
		s.Equal("/api/virtual-servers/test/projects/my-project/applications", r.URL.Path)

		err := json.NewEncoder(w).Encode(response)
		s.NoError(err)
	}))
	defer server.Close()

	testee := NewClient(server.URL, "test").Project().Application("my-project")

	// act
	responseDto, err := testee.List(s.T().Context(), requestParams)

	// assert
	s.Require().NoError(err)
	s.Equal(response, responseDto)
}

func (s *ApplicationClientSuite) TestGetApplication_HappyPath() {
	// arrange
	requestId := uuid.New()

	response := api.GetApplicationResponseDto{
		Id:                uuid.UUID{},
		Name:              "name",
		DisplayName:       "displayName",
		Type:              "public",
		SystemApplication: false,
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.Equal(http.MethodGet, r.Method)
		s.Equal(fmt.Sprintf("/api/virtual-servers/test/projects/my-project/applications/%s", requestId), r.URL.Path)

		err := json.NewEncoder(w).Encode(response)
		s.NoError(err)
	}))
	defer server.Close()

	testee := NewClient(server.URL, "test").Project().Application("my-project")

	// act
	responseDto, err := testee.Get(s.T().Context(), requestId)

	// assert
	s.Require().NoError(err)
	s.Equal(response, responseDto)
}

func (s *ApplicationClientSuite) TestDeleteApplication_HappyPath() {
	// arrange
	requestId := uuid.New()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.Equal(http.MethodDelete, r.Method)
		s.Equal(fmt.Sprintf("/api/virtual-servers/test/projects/my-project/applications/%s", requestId), r.URL.Path)
	}))
	defer server.Close()

	testee := NewClient(server.URL, "test").Project().Application("my-project")

	// act
	err := testee.Delete(s.T().Context(), requestId)

	// assert
	s.Require().NoError(err)
}

func (s *ApplicationClientSuite) TestPatchApplication_HappyPath() {
	// arrange
	requestId := uuid.New()
	request := api.PatchApplicationRequestDto{
		DisplayName:         utils.Ptr("New display name"),
		ClaimsMappingScript: nil,
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.Equal(http.MethodPatch, r.Method)
		s.Equal(fmt.Sprintf("/api/virtual-servers/test/projects/my-project/applications/%s", requestId), r.URL.Path)

		var requestDto api.PatchApplicationRequestDto
		err := json.NewDecoder(r.Body).Decode(&requestDto)
		s.NoError(err)
		s.Equal(request, requestDto)

		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	testee := NewClient(server.URL, "test").Project().Application("my-project")

	// act
	err := testee.Patch(s.T().Context(), requestId, request)

	// assert
	s.Require().NoError(err)
}

func (s *ApplicationClientSuite) TestCreateApplication_WithSigningAlgorithm() {
	// arrange
	request := api.CreateApplicationRequestDto{
		Name:             "rs256-app",
		DisplayName:      "RS256 App",
		RedirectUris:     []string{"http://localhost/callback"},
		Type:             "public",
		SigningAlgorithm: utils.Ptr("RS256"),
	}

	response := api.CreateApplicationResponseDto{
		Id: uuid.New(),
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var requestDto api.CreateApplicationRequestDto
		err := json.NewDecoder(r.Body).Decode(&requestDto)
		s.NoError(err)
		s.Equal(request, requestDto)
		s.NotNil(requestDto.SigningAlgorithm)
		s.Equal("RS256", *requestDto.SigningAlgorithm)

		err = json.NewEncoder(w).Encode(response)
		s.NoError(err)
	}))
	defer server.Close()

	testee := NewClient(server.URL, "test").Project().Application("my-project")

	// act
	responseDto, err := testee.Create(s.T().Context(), request)

	// assert
	s.Require().NoError(err)
	s.Equal(response, responseDto)
}

func (s *ApplicationClientSuite) TestPatchApplication_WithSigningAlgorithm() {
	// arrange
	requestId := uuid.New()
	request := api.PatchApplicationRequestDto{
		SigningAlgorithm: utils.Ptr("RS256"),
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var requestDto api.PatchApplicationRequestDto
		err := json.NewDecoder(r.Body).Decode(&requestDto)
		s.NoError(err)
		s.NotNil(requestDto.SigningAlgorithm)
		s.Equal("RS256", *requestDto.SigningAlgorithm)

		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	testee := NewClient(server.URL, "test").Project().Application("my-project")

	// act
	err := testee.Patch(s.T().Context(), requestId, request)

	// assert
	s.Require().NoError(err)
}

func (s *ApplicationClientSuite) TestGetApplication_ReturnsSigningAlgorithm() {
	// arrange
	requestId := uuid.New()
	response := api.GetApplicationResponseDto{
		Id:               requestId,
		Name:             "rs256-app",
		DisplayName:      "RS256 App",
		Type:             "public",
		SigningAlgorithm: utils.Ptr("RS256"),
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		err := json.NewEncoder(w).Encode(response)
		s.NoError(err)
	}))
	defer server.Close()

	testee := NewClient(server.URL, "test").Project().Application("my-project")

	// act
	responseDto, err := testee.Get(s.T().Context(), requestId)

	// assert
	s.Require().NoError(err)
	s.Require().NotNil(responseDto.SigningAlgorithm)
	s.Equal("RS256", *responseDto.SigningAlgorithm)
}
