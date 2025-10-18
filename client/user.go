package client

import (
	"Keyline/internal/handlers"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/google/uuid"
)

type ListUserParams struct {
	Page int
	Size int
}

type UserClient interface {
	List(ctx context.Context, params ListUserParams) (handlers.PagedUsersResponseDto, error)
	Get(ctx context.Context, id uuid.UUID) (handlers.GetUserByIdResponseDto, error)
	Patch(ctx context.Context, id uuid.UUID, dto handlers.PatchUserRequestDto) error
}

func NewUserClient(transport *Transport) UserClient {
	return &userClient{
		transport: transport,
	}
}

type userClient struct {
	transport *Transport
}

func (c *userClient) List(ctx context.Context, params ListUserParams) (handlers.PagedUsersResponseDto, error) {
	values := url.Values{}
	values.Add("page", fmt.Sprintf("%d", params.Page))
	values.Add("size", fmt.Sprintf("%d", params.Size))

	endpoint := fmt.Sprintf("/users?%s", values.Encode())

	request, err := c.transport.NewTenantRequest(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return handlers.PagedUsersResponseDto{}, fmt.Errorf("creating request: %w", err)
	}

	response, err := c.transport.Do(request)
	if err != nil {
		return handlers.PagedUsersResponseDto{}, fmt.Errorf("doing request: %w", err)
	}

	var responseDto handlers.PagedUsersResponseDto
	err = json.NewDecoder(response.Body).Decode(&responseDto)
	if err != nil {
		return handlers.PagedUsersResponseDto{}, fmt.Errorf("decoding response: %w", err)
	}

	return responseDto, nil
}

func (c *userClient) Get(ctx context.Context, id uuid.UUID) (handlers.GetUserByIdResponseDto, error) {
	endpoint := fmt.Sprintf("/users/%s", id.String())

	request, err := c.transport.NewTenantRequest(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return handlers.GetUserByIdResponseDto{}, fmt.Errorf("creating request: %w", err)
	}

	response, err := c.transport.Do(request)
	if err != nil {
		return handlers.GetUserByIdResponseDto{}, fmt.Errorf("doing request: %w", err)
	}

	var responseDto handlers.GetUserByIdResponseDto
	err = json.NewDecoder(response.Body).Decode(&responseDto)
	if err != nil {
		return handlers.GetUserByIdResponseDto{}, fmt.Errorf("decoding response: %w", err)
	}

	return responseDto, nil
}

func (c *userClient) Patch(ctx context.Context, id uuid.UUID, dto handlers.PatchUserRequestDto) error {
	endpoint := fmt.Sprintf("/users/%s", id.String())

	jsonBytes, err := json.Marshal(dto)
	if err != nil {
		return fmt.Errorf("marshaling dto: %w", err)
	}

	request, err := c.transport.NewTenantRequest(ctx, http.MethodPatch, endpoint, bytes.NewBuffer(jsonBytes))
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}

	_, err = c.transport.Do(request)
	if err != nil {
		return fmt.Errorf("doing request: %w", err)
	}

	return nil
}
