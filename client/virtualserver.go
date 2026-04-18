package client

import (
	"github.com/The127/Keyline/api"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

// PatchVirtualServerInput holds the fields that can be patched on a virtual server.
type PatchVirtualServerInput struct {
	DisplayName              *string `json:"displayName"`
	EnableRegistration       *bool   `json:"enableRegistration"`
	Require2fa               *bool   `json:"require2fa"`
	RequireEmailVerification *bool   `json:"requireEmailVerification"`
}

type VirtualServerClient interface {
	Create(ctx context.Context, dto api.CreateVirtualServerRequestDto) error
	Get(ctx context.Context) (api.GetVirtualServerResponseDto, error)
	GetPublicInfo(ctx context.Context) (api.GetVirtualServerListResponseDto, error)
	Patch(ctx context.Context, input PatchVirtualServerInput) error
}

func NewVirtualServerClient(transport *Transport) VirtualServerClient {
	return &virtualServerClient{
		transport: transport,
	}
}

type virtualServerClient struct {
	transport *Transport
}

func (c *virtualServerClient) Create(ctx context.Context, dto api.CreateVirtualServerRequestDto) error {
	jsonBytes, err := json.Marshal(dto)
	if err != nil {
		return fmt.Errorf("marshaling dto: %w", err)
	}

	request, err := c.transport.NewRootRequest(ctx, http.MethodPost, "/api/virtual-servers", bytes.NewBuffer(jsonBytes))
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

func (c *virtualServerClient) Get(ctx context.Context) (api.GetVirtualServerResponseDto, error) {
	endpoint := ""

	request, err := c.transport.NewTenantRequest(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return api.GetVirtualServerResponseDto{}, fmt.Errorf("creating request: %w", err)
	}

	response, err := c.transport.Do(request)
	if err != nil {
		return api.GetVirtualServerResponseDto{}, fmt.Errorf("doing request: %w", err)
	}
	defer response.Body.Close() //nolint:errcheck

	var state api.GetVirtualServerResponseDto
	if err := json.NewDecoder(response.Body).Decode(&state); err != nil {
		return api.GetVirtualServerResponseDto{}, fmt.Errorf("decoding response: %w", err)
	}

	return state, nil
}

func (c *virtualServerClient) GetPublicInfo(ctx context.Context) (api.GetVirtualServerListResponseDto, error) {
	endpoint := "/public-info"

	request, err := c.transport.NewTenantRequest(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return api.GetVirtualServerListResponseDto{}, fmt.Errorf("creating request: %w", err)
	}

	response, err := c.transport.Do(request)
	if err != nil {
		return api.GetVirtualServerListResponseDto{}, fmt.Errorf("doing request: %w", err)
	}
	defer response.Body.Close() //nolint:errcheck

	var responseDto api.GetVirtualServerListResponseDto
	err = json.NewDecoder(response.Body).Decode(&responseDto)
	if err != nil {
		return api.GetVirtualServerListResponseDto{}, fmt.Errorf("decoding response: %w", err)
	}

	return responseDto, nil
}

func (c *virtualServerClient) Patch(ctx context.Context, dto PatchVirtualServerInput) error {
	endpoint := ""

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
