package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/The127/Keyline/api"
	"github.com/google/uuid"
)

type ListRoleParams struct {
	Page int
	Size int
}

type ListUsersInRoleParams struct {
	Page int
	Size int
}

type RoleClient interface {
	Create(ctx context.Context, dto api.CreateRoleRequestDto) (api.CreateRoleResponseDto, error)
	List(ctx context.Context, params ListRoleParams) (api.PagedRolesResponseDto, error)
	Get(ctx context.Context, id uuid.UUID) (api.GetRoleByIdResponseDto, error)
	Patch(ctx context.Context, id uuid.UUID, dto api.PatchRoleRequestDto) error
	Delete(ctx context.Context, id uuid.UUID) error
	Assign(ctx context.Context, roleId uuid.UUID, userId uuid.UUID) error
	ListUsers(ctx context.Context, roleId uuid.UUID, params ListUsersInRoleParams) (api.PagedUsersInRoleResponseDto, error)
}

func NewRoleClient(transport *Transport, projectSlug string) RoleClient {
	return &roleClient{
		transport:   transport,
		projectSlug: projectSlug,
	}
}

type roleClient struct {
	transport   *Transport
	projectSlug string
}

func (c *roleClient) Create(ctx context.Context, dto api.CreateRoleRequestDto) (api.CreateRoleResponseDto, error) {
	endpoint := fmt.Sprintf("/projects/%s/roles", c.projectSlug)

	jsonBytes, err := json.Marshal(dto)
	if err != nil {
		return api.CreateRoleResponseDto{}, fmt.Errorf("marshaling dto: %w", err)
	}

	request, err := c.transport.NewTenantRequest(ctx, http.MethodPost, endpoint, bytes.NewBuffer(jsonBytes))
	if err != nil {
		return api.CreateRoleResponseDto{}, fmt.Errorf("creating request: %w", err)
	}

	response, err := c.transport.Do(request)
	if err != nil {
		return api.CreateRoleResponseDto{}, fmt.Errorf("doing request: %w", err)
	}
	defer response.Body.Close() //nolint:errcheck

	var responseDto api.CreateRoleResponseDto
	if err := json.NewDecoder(response.Body).Decode(&responseDto); err != nil {
		return api.CreateRoleResponseDto{}, fmt.Errorf("decoding response: %w", err)
	}

	return responseDto, nil
}

func (c *roleClient) List(ctx context.Context, params ListRoleParams) (api.PagedRolesResponseDto, error) {
	values := url.Values{}
	values.Add("page", fmt.Sprintf("%d", params.Page))
	values.Add("size", fmt.Sprintf("%d", params.Size))

	endpoint := fmt.Sprintf("/projects/%s/roles?%s", c.projectSlug, values.Encode())

	request, err := c.transport.NewTenantRequest(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return api.PagedRolesResponseDto{}, fmt.Errorf("creating request: %w", err)
	}

	response, err := c.transport.Do(request)
	if err != nil {
		return api.PagedRolesResponseDto{}, fmt.Errorf("doing request: %w", err)
	}
	defer response.Body.Close() //nolint:errcheck

	var responseDto api.PagedRolesResponseDto
	if err := json.NewDecoder(response.Body).Decode(&responseDto); err != nil {
		return api.PagedRolesResponseDto{}, fmt.Errorf("decoding response: %w", err)
	}

	return responseDto, nil
}

func (c *roleClient) Get(ctx context.Context, id uuid.UUID) (api.GetRoleByIdResponseDto, error) {
	endpoint := fmt.Sprintf("/projects/%s/roles/%s", c.projectSlug, id.String())

	request, err := c.transport.NewTenantRequest(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return api.GetRoleByIdResponseDto{}, fmt.Errorf("creating request: %w", err)
	}

	response, err := c.transport.Do(request)
	if err != nil {
		return api.GetRoleByIdResponseDto{}, fmt.Errorf("doing request: %w", err)
	}
	defer response.Body.Close() //nolint:errcheck

	var responseDto api.GetRoleByIdResponseDto
	if err := json.NewDecoder(response.Body).Decode(&responseDto); err != nil {
		return api.GetRoleByIdResponseDto{}, fmt.Errorf("decoding response: %w", err)
	}

	return responseDto, nil
}

func (c *roleClient) Patch(ctx context.Context, id uuid.UUID, dto api.PatchRoleRequestDto) error {
	endpoint := fmt.Sprintf("/projects/%s/roles/%s", c.projectSlug, id.String())

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

func (c *roleClient) Delete(ctx context.Context, id uuid.UUID) error {
	endpoint := fmt.Sprintf("/projects/%s/roles/%s", c.projectSlug, id.String())

	request, err := c.transport.NewTenantRequest(ctx, http.MethodDelete, endpoint, nil)
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

func (c *roleClient) Assign(ctx context.Context, roleId uuid.UUID, userId uuid.UUID) error {
	endpoint := fmt.Sprintf("/projects/%s/roles/%s/assign", c.projectSlug, roleId.String())

	dto := api.AssignRoleRequestDto{UserId: userId}
	jsonBytes, err := json.Marshal(dto)
	if err != nil {
		return fmt.Errorf("marshaling dto: %w", err)
	}

	request, err := c.transport.NewTenantRequest(ctx, http.MethodPost, endpoint, bytes.NewBuffer(jsonBytes))
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

func (c *roleClient) ListUsers(ctx context.Context, roleId uuid.UUID, params ListUsersInRoleParams) (api.PagedUsersInRoleResponseDto, error) {
	values := url.Values{}
	values.Add("page", fmt.Sprintf("%d", params.Page))
	values.Add("size", fmt.Sprintf("%d", params.Size))

	endpoint := fmt.Sprintf("/projects/%s/roles/%s/users?%s", c.projectSlug, roleId.String(), values.Encode())

	request, err := c.transport.NewTenantRequest(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return api.PagedUsersInRoleResponseDto{}, fmt.Errorf("creating request: %w", err)
	}

	response, err := c.transport.Do(request)
	if err != nil {
		return api.PagedUsersInRoleResponseDto{}, fmt.Errorf("doing request: %w", err)
	}
	defer response.Body.Close() //nolint:errcheck

	var responseDto api.PagedUsersInRoleResponseDto
	if err := json.NewDecoder(response.Body).Decode(&responseDto); err != nil {
		return api.PagedUsersInRoleResponseDto{}, fmt.Errorf("decoding response: %w", err)
	}

	return responseDto, nil
}
