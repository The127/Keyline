package client

import (
	"Keyline/internal/handlers"
	"Keyline/utils"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

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
	request := handlers.CreateApplicationRequestDto{
		Name:           "applicationName",
		DisplayName:    "displayName",
		RedirectUris:   []string{"http://localhost:8080/callback"},
		PostLogoutUris: []string{"http://localhost:8080/logout"},
		Type:           "confidential",
	}

	response := handlers.CreateApplicationResponseDto{
		Id:     uuid.New(),
		Secret: utils.Ptr("secret"),
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.Equal(http.MethodPost, r.Method)
		s.Equal("/api/virtual-servers/test/applications", r.URL.Path)

		var requestDto handlers.CreateApplicationRequestDto
		err := json.NewDecoder(r.Body).Decode(&requestDto)
		s.Require().NoError(err)
		s.EqualValues(request, requestDto)

		err = json.NewEncoder(w).Encode(response)
		s.Require().NoError(err)
	}))
	defer server.Close()

	testee := NewClient(server.URL, "test").Application()

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

	response := handlers.PagedApplicationsResponseDto{
		Items: []handlers.ListApplicationsResponseDto{
			{
				Id:                uuid.UUID{},
				Name:              "name",
				DisplayName:       "displayName",
				Type:              "public",
				SystemApplication: false,
			},
		},
		Pagination: &handlers.Pagination{
			Page:       1,
			Size:       11,
			TotalPages: 2,
			TotalItems: 22,
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.Equal(http.MethodGet, r.Method)
		s.Equal("/api/virtual-servers/test/applications", r.URL.Path)

		err := json.NewEncoder(w).Encode(response)
		s.Require().NoError(err)
	}))
	defer server.Close()

	testee := NewClient(server.URL, "test").Application()

	// act
	responseDto, err := testee.List(s.T().Context(), requestParams)

	// assert
	s.Require().NoError(err)
	s.Equal(response, responseDto)
}

func (s *ApplicationClientSuite) TestGetApplication_HappyPath() {
	// arrange
	requestId := uuid.New()

	response := handlers.GetApplicationResponseDto{
		Id:                uuid.UUID{},
		Name:              "name",
		DisplayName:       "displayName",
		Type:              "public",
		SystemApplication: false,
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.Equal(http.MethodGet, r.Method)
		s.Equal(fmt.Sprintf("/api/virtual-servers/test/applications/%s", requestId), r.URL.Path)

		err := json.NewEncoder(w).Encode(response)
		s.Require().NoError(err)
	}))
	defer server.Close()

	testee := NewClient(server.URL, "test").Application()

	// act
	responseDto, err := testee.Get(s.T().Context(), requestId)

	// assert
	s.Require().NoError(err)
	s.Equal(response, responseDto)
}
