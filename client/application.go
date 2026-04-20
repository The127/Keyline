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

type ListApplicationParams struct {
	Page int
	Size int
}

type ApplicationClient interface {
	Create(ctx context.Context, dto api.CreateApplicationRequestDto) (api.CreateApplicationResponseDto, error)
	List(ctx context.Context, params ListApplicationParams) (api.PagedApplicationsResponseDto, error)
	Get(ctx context.Context, id uuid.UUID) (api.GetApplicationResponseDto, error)
	Patch(ctx context.Context, id uuid.UUID, dto api.PatchApplicationRequestDto) error
	Delete(ctx context.Context, id uuid.UUID) error
}

func NewApplicationClient(transport *Transport, projectSlug string) ApplicationClient {
	return &application{
		transport:   transport,
		projectSlug: projectSlug,
	}
}

type application struct {
	transport   *Transport
	projectSlug string
}

func (a *application) Create(ctx context.Context, dto api.CreateApplicationRequestDto) (api.CreateApplicationResponseDto, error) {
	endpoint := fmt.Sprintf("/projects/%s/applications", a.projectSlug)

	jsonBytes, err := json.Marshal(dto)
	if err != nil {
		return api.CreateApplicationResponseDto{}, fmt.Errorf("marshaling dto: %w", err)
	}

	request, err := a.transport.NewTenantRequest(ctx, http.MethodPost, endpoint, bytes.NewBuffer(jsonBytes))
	if err != nil {
		return api.CreateApplicationResponseDto{}, fmt.Errorf("creating request: %w", err)
	}

	response, err := a.transport.Do(request)
	if err != nil {
		return api.CreateApplicationResponseDto{}, fmt.Errorf("doing request: %w", err)
	}
	defer response.Body.Close() //nolint:errcheck

	var responseDto api.CreateApplicationResponseDto
	err = json.NewDecoder(response.Body).Decode(&responseDto)
	if err != nil {
		return api.CreateApplicationResponseDto{}, fmt.Errorf("decoding response: %w", err)
	}

	return responseDto, nil
}

func (a *application) List(ctx context.Context, params ListApplicationParams) (api.PagedApplicationsResponseDto, error) {
	values := url.Values{}
	values.Add("page", fmt.Sprintf("%d", params.Page))
	values.Add("size", fmt.Sprintf("%d", params.Size))

	endpoint := fmt.Sprintf("/projects/%s/applications?%s", a.projectSlug, values.Encode())

	request, err := a.transport.NewTenantRequest(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return api.PagedApplicationsResponseDto{}, fmt.Errorf("creating request: %w", err)
	}

	response, err := a.transport.Do(request)
	if err != nil {
		return api.PagedApplicationsResponseDto{}, fmt.Errorf("doing request: %w", err)
	}
	defer response.Body.Close() //nolint:errcheck

	var responseDto api.PagedApplicationsResponseDto
	err = json.NewDecoder(response.Body).Decode(&responseDto)
	if err != nil {
		return api.PagedApplicationsResponseDto{}, fmt.Errorf("decoding response: %w", err)
	}

	return responseDto, nil
}

func (a *application) Get(ctx context.Context, id uuid.UUID) (api.GetApplicationResponseDto, error) {
	endpoint := fmt.Sprintf("/projects/%s/applications/%s", a.projectSlug, id.String())

	request, err := a.transport.NewTenantRequest(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return api.GetApplicationResponseDto{}, fmt.Errorf("creating request: %w", err)
	}

	response, err := a.transport.Do(request)
	if err != nil {
		return api.GetApplicationResponseDto{}, fmt.Errorf("doing request: %w", err)
	}
	defer response.Body.Close() //nolint:errcheck

	var responseDto api.GetApplicationResponseDto
	err = json.NewDecoder(response.Body).Decode(&responseDto)
	if err != nil {
		return api.GetApplicationResponseDto{}, fmt.Errorf("decoding response: %w", err)
	}

	return responseDto, nil
}

func (a *application) Delete(ctx context.Context, id uuid.UUID) error {
	endpoint := fmt.Sprintf("/projects/%s/applications/%s", a.projectSlug, id.String())

	request, err := a.transport.NewTenantRequest(ctx, http.MethodDelete, endpoint, nil)
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}

	response, err := a.transport.Do(request)
	if err != nil {
		return fmt.Errorf("doing request: %w", err)
	}
	defer response.Body.Close() //nolint:errcheck

	return nil
}

func (a *application) Patch(ctx context.Context, id uuid.UUID, dto api.PatchApplicationRequestDto) error {
	endpoint := fmt.Sprintf("/projects/%s/applications/%s", a.projectSlug, id.String())

	jsonBytes, err := json.Marshal(dto)
	if err != nil {
		return fmt.Errorf("marshaling dto: %w", err)
	}

	request, err := a.transport.NewTenantRequest(ctx, http.MethodPatch, endpoint, bytes.NewBuffer(jsonBytes))
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}

	response, err := a.transport.Do(request)
	if err != nil {
		return fmt.Errorf("doing request: %w", err)
	}
	defer response.Body.Close() //nolint:errcheck

	return nil
}
