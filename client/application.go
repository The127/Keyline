package client

import (
	"Keyline/internal/handlers"
	"Keyline/utils"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/google/uuid"
)

type ListApplicationParams struct {
	Page int
	Size int
}

type ApplicationClient interface {
	Create(ctx context.Context, dto handlers.CreateApplicationRequestDto) (handlers.CreateApplicationResponseDto, error)
	List(ctx context.Context, params ListApplicationParams) (handlers.PagedApplicationsResponseDto, error)
	Get(ctx context.Context, id uuid.UUID) (handlers.GetApplicationResponseDto, error)
	Delete(ctx context.Context, id uuid.UUID) error
}

func NewApplicationClient(transport *Transport) ApplicationClient {
	return &application{
		transport: transport,
	}
}

type application struct {
	transport *Transport
}

func (a *application) Create(ctx context.Context, dto handlers.CreateApplicationRequestDto) (handlers.CreateApplicationResponseDto, error) {
	endpoint := "/applications"

	jsonBytes, err := json.Marshal(dto)
	if err != nil {
		return handlers.CreateApplicationResponseDto{}, fmt.Errorf("marshaling dto: %w", err)
	}

	request, err := a.transport.NewRequest(ctx, http.MethodPost, endpoint, bytes.NewBuffer(jsonBytes))
	if err != nil {
		return handlers.CreateApplicationResponseDto{}, fmt.Errorf("creating request: %w", err)
	}
	defer utils.PanicOnError(request.Body.Close, "closing request body")

	response, err := a.transport.Do(request)
	if err != nil {
		return handlers.CreateApplicationResponseDto{}, fmt.Errorf("doing request: %w", err)
	}

	var responseDto handlers.CreateApplicationResponseDto
	err = json.NewDecoder(response.Body).Decode(&responseDto)
	if err != nil {
		return handlers.CreateApplicationResponseDto{}, fmt.Errorf("decoding response: %w", err)
	}

	return responseDto, nil
}

func (a *application) List(ctx context.Context, params ListApplicationParams) (handlers.PagedApplicationsResponseDto, error) {
	endpoint := fmt.Sprintf("/applications?page=%d&size=%d", params.Page, params.Size)

	request, err := a.transport.NewRequest(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return handlers.PagedApplicationsResponseDto{}, fmt.Errorf("creating request: %w", err)
	}

	response, err := a.transport.Do(request)
	if err != nil {
		return handlers.PagedApplicationsResponseDto{}, fmt.Errorf("doing request: %w", err)
	}

	var responseDto handlers.PagedApplicationsResponseDto
	err = json.NewDecoder(response.Body).Decode(&responseDto)

	return responseDto, nil
}

func (a *application) Get(ctx context.Context, id uuid.UUID) (handlers.GetApplicationResponseDto, error) {
	endpoint := fmt.Sprintf("/applications/%s", id.String())

	request, err := a.transport.NewRequest(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return handlers.GetApplicationResponseDto{}, fmt.Errorf("creating request: %w", err)
	}

	response, err := a.transport.Do(request)
	if err != nil {
		return handlers.GetApplicationResponseDto{}, fmt.Errorf("doing request: %w", err)
	}

	var responseDto handlers.GetApplicationResponseDto
	err = json.NewDecoder(response.Body).Decode(&responseDto)
	if err != nil {
		return handlers.GetApplicationResponseDto{}, fmt.Errorf("decoding response: %w", err)
	}

	return responseDto, nil
}

func (a *application) Delete(ctx context.Context, id uuid.UUID) error {
	endpoint := fmt.Sprintf("/applications/%s", id.String())

	request, err := a.transport.NewRequest(ctx, http.MethodDelete, endpoint, nil)
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}

	_, err = a.transport.Do(request)
	if err != nil {
		return fmt.Errorf("doing request: %w", err)
	}

	return nil
}
