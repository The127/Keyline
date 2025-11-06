package client

import (
	"Keyline/internal/handlers"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

type VirtualServerClient interface {
	Create(ctx context.Context, dto handlers.CreateVirtualServerRequestDto) error
	Get(ctx context.Context) (handlers.GetVirtualServerResponseDto, error)
	GetPublicInfo(ctx context.Context) (handlers.GetVirtualServerListResponseDto, error)
	Patch(ctx context.Context, dto handlers.PatchVirtualServerRequestDto) error
}

func NewVirtualServerClient(transport *Transport) VirtualServerClient {
	return &virtualServerClient{
		transport: transport,
	}
}

type virtualServerClient struct {
	transport *Transport
}

func (c *virtualServerClient) Create(ctx context.Context, dto handlers.CreateVirtualServerRequestDto) error {
	jsonBytes, err := json.Marshal(dto)
	if err != nil {
		return fmt.Errorf("marshaling dto: %w", err)
	}

	request, err := c.transport.NewRootRequest(ctx, http.MethodPost, "/api/virtual-servers", bytes.NewBuffer(jsonBytes))
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}

	_, err = c.transport.Do(request)
	if err != nil {
		return fmt.Errorf("doing request: %w", err)
	}

	return nil
}

func (c *virtualServerClient) Get(ctx context.Context) (handlers.GetVirtualServerResponseDto, error) {
	endpoint := ""

	request, err := c.transport.NewTenantRequest(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return handlers.GetVirtualServerResponseDto{}, fmt.Errorf("creating request: %w", err)
	}

	response, err := c.transport.Do(request)
	if err != nil {
		return handlers.GetVirtualServerResponseDto{}, fmt.Errorf("doing request: %w", err)
	}

	var responseDto handlers.GetVirtualServerResponseDto
	err = json.NewDecoder(response.Body).Decode(&responseDto)
	if err != nil {
		return handlers.GetVirtualServerResponseDto{}, fmt.Errorf("decoding response: %w", err)
	}

	return responseDto, nil
}

func (c *virtualServerClient) GetPublicInfo(ctx context.Context) (handlers.GetVirtualServerListResponseDto, error) {
	endpoint := "/public-info"

	request, err := c.transport.NewTenantRequest(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return handlers.GetVirtualServerListResponseDto{}, fmt.Errorf("creating request: %w", err)
	}

	response, err := c.transport.Do(request)
	if err != nil {
		return handlers.GetVirtualServerListResponseDto{}, fmt.Errorf("doing request: %w", err)
	}

	var responseDto handlers.GetVirtualServerListResponseDto
	err = json.NewDecoder(response.Body).Decode(&responseDto)
	if err != nil {
		return handlers.GetVirtualServerListResponseDto{}, fmt.Errorf("decoding response: %w", err)
	}

	return responseDto, nil
}

func (c *virtualServerClient) Patch(ctx context.Context, dto handlers.PatchVirtualServerRequestDto) error {
	endpoint := ""

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
