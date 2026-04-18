package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/The127/Keyline/api"
)

type ProjectClient interface {
	Create(ctx context.Context, input api.CreateProjectRequestDto) (api.CreateProjectResponseDto, error)
	Get(ctx context.Context, slug string) (api.GetProjectResponseDto, error)
}

func NewProjectClient(transport *Transport) ProjectClient {
	return &projectClient{transport: transport}
}

type projectClient struct {
	transport *Transport
}

func (c *projectClient) Create(ctx context.Context, input api.CreateProjectRequestDto) (api.CreateProjectResponseDto, error) {
	jsonBytes, err := json.Marshal(input)
	if err != nil {
		return api.CreateProjectResponseDto{}, fmt.Errorf("marshaling input: %w", err)
	}

	request, err := c.transport.NewTenantRequest(ctx, http.MethodPost, "/projects", bytes.NewBuffer(jsonBytes))
	if err != nil {
		return api.CreateProjectResponseDto{}, fmt.Errorf("creating request: %w", err)
	}

	response, err := c.transport.Do(request)
	if err != nil {
		return api.CreateProjectResponseDto{}, fmt.Errorf("doing request: %w", err)
	}
	defer response.Body.Close() //nolint:errcheck

	var responseDto api.CreateProjectResponseDto
	if err := json.NewDecoder(response.Body).Decode(&responseDto); err != nil {
		return api.CreateProjectResponseDto{}, fmt.Errorf("decoding response: %w", err)
	}

	return responseDto, nil
}

func (c *projectClient) Get(ctx context.Context, slug string) (api.GetProjectResponseDto, error) {
	endpoint := fmt.Sprintf("/projects/%s", slug)

	request, err := c.transport.NewTenantRequest(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return api.GetProjectResponseDto{}, fmt.Errorf("creating request: %w", err)
	}

	response, err := c.transport.Do(request)
	if err != nil {
		return api.GetProjectResponseDto{}, fmt.Errorf("doing request: %w", err)
	}
	defer response.Body.Close() //nolint:errcheck

	var responseDto api.GetProjectResponseDto
	if err := json.NewDecoder(response.Body).Decode(&responseDto); err != nil {
		return api.GetProjectResponseDto{}, fmt.Errorf("decoding response: %w", err)
	}

	return responseDto, nil
}
