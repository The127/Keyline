package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/The127/Keyline/api"
	"net/http"
	"net/url"

	"github.com/google/uuid"
)

type ListUserParams struct {
	Page int
	Size int
}

type UserClient interface {
	List(ctx context.Context, params ListUserParams) (api.PagedUsersResponseDto, error)
	Get(ctx context.Context, id uuid.UUID) (api.GetUserByIdResponseDto, error)
	Patch(ctx context.Context, id uuid.UUID, dto api.PatchUserRequestDto) error
	CreateServiceUser(ctx context.Context, username string) (uuid.UUID, error)
	AssociateServiceUserPublicKey(ctx context.Context, serviceUserID uuid.UUID, publicKeyPEM string) (string, error)
}

func NewUserClient(transport *Transport) UserClient {
	return &userClient{
		transport: transport,
	}
}

type userClient struct {
	transport *Transport
}

func (c *userClient) List(ctx context.Context, params ListUserParams) (api.PagedUsersResponseDto, error) {
	values := url.Values{}
	values.Add("page", fmt.Sprintf("%d", params.Page))
	values.Add("size", fmt.Sprintf("%d", params.Size))

	endpoint := fmt.Sprintf("/users?%s", values.Encode())

	request, err := c.transport.NewTenantRequest(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return api.PagedUsersResponseDto{}, fmt.Errorf("creating request: %w", err)
	}

	response, err := c.transport.Do(request)
	if err != nil {
		return api.PagedUsersResponseDto{}, fmt.Errorf("doing request: %w", err)
	}
	defer response.Body.Close() //nolint:errcheck

	var responseDto api.PagedUsersResponseDto
	err = json.NewDecoder(response.Body).Decode(&responseDto)
	if err != nil {
		return api.PagedUsersResponseDto{}, fmt.Errorf("decoding response: %w", err)
	}

	return responseDto, nil
}

func (c *userClient) Get(ctx context.Context, id uuid.UUID) (api.GetUserByIdResponseDto, error) {
	endpoint := fmt.Sprintf("/users/%s", id.String())

	request, err := c.transport.NewTenantRequest(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return api.GetUserByIdResponseDto{}, fmt.Errorf("creating request: %w", err)
	}

	response, err := c.transport.Do(request)
	if err != nil {
		return api.GetUserByIdResponseDto{}, fmt.Errorf("doing request: %w", err)
	}
	defer response.Body.Close() //nolint:errcheck

	var responseDto api.GetUserByIdResponseDto
	err = json.NewDecoder(response.Body).Decode(&responseDto)
	if err != nil {
		return api.GetUserByIdResponseDto{}, fmt.Errorf("decoding response: %w", err)
	}

	return responseDto, nil
}

func (c *userClient) Patch(ctx context.Context, id uuid.UUID, dto api.PatchUserRequestDto) error {
	endpoint := fmt.Sprintf("/users/%s", id.String())

	jsonBytes, err := json.Marshal(dto)
	if err != nil {
		return fmt.Errorf("marshaling dto: %w", err)
	}

	request, err := c.transport.NewTenantRequest(ctx, http.MethodPatch, endpoint, bytes.NewBuffer(jsonBytes))
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}

	response, err := c.transport.Do(request)
	if err != nil {
		return fmt.Errorf("doing request: %w", err)
	}
	defer response.Body.Close() //nolint:errcheck

	return nil
}

func (c *userClient) CreateServiceUser(ctx context.Context, username string) (uuid.UUID, error) {
	jsonBytes, err := json.Marshal(api.CreateServiceUserRequestDto{Username: username})
	if err != nil {
		return uuid.UUID{}, fmt.Errorf("marshaling dto: %w", err)
	}

	request, err := c.transport.NewTenantRequest(ctx, http.MethodPost, "/users/service-users", bytes.NewBuffer(jsonBytes))
	if err != nil {
		return uuid.UUID{}, fmt.Errorf("creating request: %w", err)
	}

	response, err := c.transport.Do(request)
	if err != nil {
		return uuid.UUID{}, fmt.Errorf("doing request: %w", err)
	}
	defer response.Body.Close() //nolint:errcheck

	var responseDto api.CreateServiceUserResponseDto
	if err := json.NewDecoder(response.Body).Decode(&responseDto); err != nil {
		return uuid.UUID{}, fmt.Errorf("decoding response: %w", err)
	}

	return responseDto.Id, nil
}

func (c *userClient) AssociateServiceUserPublicKey(ctx context.Context, serviceUserID uuid.UUID, publicKeyPEM string) (string, error) {
	jsonBytes, err := json.Marshal(api.AssociateServiceUserPublicKeyRequestDto{PublicKey: publicKeyPEM})
	if err != nil {
		return "", fmt.Errorf("marshaling dto: %w", err)
	}

	endpoint := fmt.Sprintf("/users/service-users/%s/keys", serviceUserID)
	request, err := c.transport.NewTenantRequest(ctx, http.MethodPost, endpoint, bytes.NewBuffer(jsonBytes))
	if err != nil {
		return "", fmt.Errorf("creating request: %w", err)
	}

	response, err := c.transport.Do(request)
	if err != nil {
		return "", fmt.Errorf("doing request: %w", err)
	}
	defer response.Body.Close() //nolint:errcheck

	var responseDto api.AssociateServiceUserPublicKeyResponseDto
	if err := json.NewDecoder(response.Body).Decode(&responseDto); err != nil {
		return "", fmt.Errorf("decoding response: %w", err)
	}

	return responseDto.Kid, nil
}
