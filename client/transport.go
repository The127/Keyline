package client

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

type ApiError struct {
	Message string `json:"message"`
	Code    int    `json:"code"`
}

func (e ApiError) Error() string {
	return fmt.Sprintf("API error: %s (%d)", e.Message, e.Code)
}

type TransportOptions func(*Transport)

func WithClient(client *http.Client) TransportOptions {
	return func(t *Transport) {
		t.client = client
	}
}

func WithBaseURL(baseURL string) TransportOptions {
	return func(t *Transport) {
		t.baseURL = baseURL
	}
}

func WithRoundTripper(roundTripperFactory func(next http.RoundTripper) http.RoundTripper) TransportOptions {
	return func(t *Transport) {
		t.client.Transport = roundTripperFactory(t.client.Transport)
	}
}

type Transport struct {
	baseURL       string
	virtualServer string
	client        *http.Client
}

func NewTransport(baseUrl string, virtualServer string, options ...TransportOptions) *Transport {
	transport := &Transport{
		baseURL:       baseUrl,
		virtualServer: virtualServer,
		client:        http.DefaultClient,
	}

	for _, option := range options {
		option(transport)
	}

	return transport
}

func (t *Transport) NewRequest(ctx context.Context, method string, endpoint string, body io.Reader) (*http.Request, error) {
	base, err := url.Parse(t.baseURL)
	if err != nil {
		return nil, fmt.Errorf("parsing base URL: %w", err)
	}

	ref, err := url.Parse("/api/virtual-servers/" + t.virtualServer + endpoint)
	if err != nil {
		return nil, fmt.Errorf("parsing endpoint: %w", err)
	}

	fullURL := base.ResolveReference(ref)

	request, err := http.NewRequestWithContext(ctx, method, fullURL.String(), body)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	if method == http.MethodPost || method == http.MethodPut || method == http.MethodPatch {
		request.Header.Set("Content-Type", "application/json")
	}

	return request, nil
}

func (t *Transport) Do(req *http.Request) (*http.Response, error) {
	response, err := t.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("doing request: %w", err)
	}

	if response.StatusCode >= 400 {
		return nil, ApiError{
			Message: response.Status,
			Code:    response.StatusCode,
		}
	}

	return response, nil
}
