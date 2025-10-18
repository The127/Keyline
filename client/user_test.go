package client

import (
	"Keyline/internal/handlers"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"
)

type UserClientSuite struct {
	suite.Suite
}

func TestUserClientSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(UserClientSuite))
}

func (s *UserClientSuite) TestListUsers_HappyPath() {
	// arrange
	requestParams := ListUserParams{
		Page: 1,
		Size: 11,
	}

	response := handlers.PagedUsersResponseDto{
		Items: []handlers.ListUsersResponseDto{
			{
				Id:            uuid.New(),
				Username:      "username",
				DisplayName:   "displayName",
				PrimaryEmail:  "primary@Email",
				IsServiceUser: false,
			},
		},
		Pagination: handlers.Pagination{
			Page:       1,
			Size:       11,
			TotalPages: 2,
			TotalItems: 22,
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.Equal(http.MethodGet, r.Method)
		s.Equal("/api/virtual-servers/test/users", r.URL.Path)

		err := json.NewEncoder(w).Encode(response)
		s.NoError(err)
	}))
	defer server.Close()

	testee := NewClient(server.URL, "test").User()

	// act
	responseDto, err := testee.List(s.T().Context(), requestParams)

	// assert
	s.Require().NoError(err)
	s.Equal(response, responseDto)
}

func (s *UserClientSuite) TestGetUser_HappyPath() {
	// arrange
	requestId := uuid.New()

	response := handlers.GetUserByIdResponseDto{
		Id:            requestId,
		Username:      "username",
		DisplayName:   "displayName",
		PrimaryEmail:  "primary@Email",
		IsServiceUser: false,
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.Equal(http.MethodGet, r.Method)
		s.Equal(fmt.Sprintf("/api/virtual-servers/test/users/%s", requestId), r.URL.Path)

		err := json.NewEncoder(w).Encode(response)
		s.NoError(err)
	}))
	defer server.Close()

	testee := NewClient(server.URL, "test").User()

	// act
	responseDto, err := testee.Get(s.T().Context(), requestId)

	// assert
	s.Require().NoError(err)
	s.Equal(response, responseDto)
}
