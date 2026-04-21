package client

import (
	"encoding/json"
	"fmt"
	"github.com/The127/Keyline/api"
	"github.com/The127/Keyline/utils"
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

func (s *UserClientSuite) TestCreateUser_HappyPath() {
	// arrange
	request := api.CreateUserRequestDto{
		Username:      "newuser",
		DisplayName:   "New User",
		Email:         "newuser@example.com",
		EmailVerified: utils.Ptr(true),
		Password: &api.CreateUserRequestDtoPasword{
			Plain:     "hunter2",
			Temporary: true,
		},
	}
	response := api.CreateUserResponseDto{
		Id: uuid.New(),
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.Equal(http.MethodPost, r.Method)
		s.Equal("/api/virtual-servers/test/users", r.URL.Path)

		var requestDto api.CreateUserRequestDto
		err := json.NewDecoder(r.Body).Decode(&requestDto)
		s.NoError(err)
		s.Equal(request, requestDto)

		w.WriteHeader(http.StatusCreated)
		err = json.NewEncoder(w).Encode(response)
		s.NoError(err)
	}))
	defer server.Close()

	testee := NewClient(server.URL, "test").User()

	// act
	responseDto, err := testee.Create(s.T().Context(), request)

	// assert
	s.Require().NoError(err)
	s.Equal(response, responseDto)
}

func (s *UserClientSuite) TestListUsers_HappyPath() {
	// arrange
	requestParams := ListUserParams{
		Page: 1,
		Size: 11,
	}

	response := api.PagedUsersResponseDto{
		Items: []api.ListUsersResponseDto{
			{
				Id:            uuid.New(),
				Username:      "username",
				DisplayName:   "displayName",
				PrimaryEmail:  "primary@Email",
				IsServiceUser: false,
			},
		},
		Pagination: api.Pagination{
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

	response := api.GetUserByIdResponseDto{
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

func (s *UserClientSuite) TestAssociateServiceUserPublicKey_HappyPath() {
	// arrange
	serviceUserId := uuid.New()
	request := api.AssociateServiceUserPublicKeyRequestDto{
		PublicKey: "-----BEGIN PUBLIC KEY-----\nabc\n-----END PUBLIC KEY-----",
		Kid:       utils.Ptr("my-kid"),
	}
	response := api.AssociateServiceUserPublicKeyResponseDto{Kid: "my-kid"}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.Equal(http.MethodPost, r.Method)
		s.Equal(fmt.Sprintf("/api/virtual-servers/test/users/service-users/%s/keys", serviceUserId), r.URL.Path)

		var requestDto api.AssociateServiceUserPublicKeyRequestDto
		err := json.NewDecoder(r.Body).Decode(&requestDto)
		s.NoError(err)
		s.Equal(request, requestDto)

		err = json.NewEncoder(w).Encode(response)
		s.NoError(err)
	}))
	defer server.Close()

	testee := NewClient(server.URL, "test").User()

	// act
	responseDto, err := testee.AssociateServiceUserPublicKey(s.T().Context(), serviceUserId, request)

	// assert
	s.Require().NoError(err)
	s.Equal(response, responseDto)
}

func (s *UserClientSuite) TestAssociateServiceUserPublicKey_NoKid() {
	// arrange
	serviceUserId := uuid.New()
	request := api.AssociateServiceUserPublicKeyRequestDto{
		PublicKey: "-----BEGIN PUBLIC KEY-----\nabc\n-----END PUBLIC KEY-----",
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var requestDto api.AssociateServiceUserPublicKeyRequestDto
		err := json.NewDecoder(r.Body).Decode(&requestDto)
		s.NoError(err)
		s.Nil(requestDto.Kid)

		err = json.NewEncoder(w).Encode(api.AssociateServiceUserPublicKeyResponseDto{Kid: "server-generated-kid"})
		s.NoError(err)
	}))
	defer server.Close()

	testee := NewClient(server.URL, "test").User()

	// act
	responseDto, err := testee.AssociateServiceUserPublicKey(s.T().Context(), serviceUserId, request)

	// assert
	s.Require().NoError(err)
	s.Equal("server-generated-kid", responseDto.Kid)
}

func (s *UserClientSuite) TestRemoveServiceUserPublicKey_HappyPath() {
	// arrange
	serviceUserId := uuid.New()
	kid := "my-kid"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.Equal(http.MethodDelete, r.Method)
		s.Equal(fmt.Sprintf("/api/virtual-servers/test/users/service-users/%s/keys/%s", serviceUserId, kid), r.URL.Path)
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	testee := NewClient(server.URL, "test").User()

	// act
	err := testee.RemoveServiceUserPublicKey(s.T().Context(), serviceUserId, kid)

	// assert
	s.Require().NoError(err)
}

func (s *UserClientSuite) TestPatchUser_HappyPath() {
	// arrange
	requestId := uuid.New()
	request := api.PatchUserRequestDto{
		DisplayName: utils.Ptr("New display name"),
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.Equal(http.MethodPatch, r.Method)
		s.Equal(fmt.Sprintf("/api/virtual-servers/test/users/%s", requestId), r.URL.Path)

		var requestDto api.PatchUserRequestDto
		err := json.NewDecoder(r.Body).Decode(&requestDto)
		s.NoError(err)
		s.Equal(request, requestDto)

		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	testee := NewClient(server.URL, "test").User()

	// act
	err := testee.Patch(s.T().Context(), requestId, request)

	// assert
	s.Require().NoError(err)
}
