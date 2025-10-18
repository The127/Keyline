package client

import (
	"Keyline/internal/handlers"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
)

type ListUserParams struct {
	Page int
	Size int
}

type UserClient interface {
	List(ctx context.Context, params ListUserParams) (handlers.PagedUsersResponseDto, error)
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

	request, err := c.transport.NewRequest(ctx, http.MethodGet, endpoint, nil)
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
