package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/The127/Keyline/api"
	"net/http"
	"net/url"
)

type IdentityProviderClient interface {
	Create(ctx context.Context, dto api.CreateIdentityProviderRequestDto) (api.CreateIdentityProviderResponseDto, error)
	Get(ctx context.Context, name string) (api.GetIdentityProviderResponseDto, error)
}

func NewIdentityProviderClient(transport *Transport) IdentityProviderClient {
	return &identityProviderClient{
		transport: transport,
	}
}

type identityProviderClient struct {
	transport *Transport
}

func (c *identityProviderClient) Create(ctx context.Context, dto api.CreateIdentityProviderRequestDto) (api.CreateIdentityProviderResponseDto, error) {
	jsonBytes, err := json.Marshal(dto)
	if err != nil {
		return api.CreateIdentityProviderResponseDto{}, fmt.Errorf("marshaling dto: %w", err)
	}

	request, err := c.transport.NewTenantRequest(ctx, http.MethodPost, "/identity-providers", bytes.NewBuffer(jsonBytes))
	if err != nil {
		return api.CreateIdentityProviderResponseDto{}, fmt.Errorf("creating request: %w", err)
	}

	response, err := c.transport.Do(request)
	if err != nil {
		return api.CreateIdentityProviderResponseDto{}, fmt.Errorf("doing request: %w", err)
	}

	defer response.Body.Close() //nolint:errcheck

	var responseDto api.CreateIdentityProviderResponseDto
	err = json.NewDecoder(response.Body).Decode(&responseDto)
	if err != nil {
		return api.CreateIdentityProviderResponseDto{}, fmt.Errorf("decoding response: %w", err)
	}

	return responseDto, nil
}

func (c *identityProviderClient) Get(ctx context.Context, name string) (api.GetIdentityProviderResponseDto, error) {
	request, err := c.transport.NewTenantRequest(ctx, http.MethodGet, fmt.Sprintf("/identity-providers/%s", url.PathEscape(name)), nil)
	if err != nil {
		return api.GetIdentityProviderResponseDto{}, fmt.Errorf("creating request: %w", err)
	}

	response, err := c.transport.Do(request)
	if err != nil {
		return api.GetIdentityProviderResponseDto{}, fmt.Errorf("doing request: %w", err)
	}

	defer response.Body.Close() //nolint:errcheck

	var responseDto api.GetIdentityProviderResponseDto
	err = json.NewDecoder(response.Body).Decode(&responseDto)
	if err != nil {
		return api.GetIdentityProviderResponseDto{}, fmt.Errorf("decoding response: %w", err)
	}

	return responseDto, nil
}
