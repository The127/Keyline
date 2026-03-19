package client

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

var (
	ErrAuthorizationPending = errors.New("authorization_pending")
	ErrAccessDenied         = errors.New("access_denied")
	ErrExpiredToken         = errors.New("expired_token")
	ErrSlowDown             = errors.New("slow_down")
	ErrInvalidUserCode      = errors.New("invalid_user_code")
)

type DeviceAuthorizationResponse struct {
	DeviceCode              string `json:"device_code"`
	UserCode                string `json:"user_code"`
	VerificationUri         string `json:"verification_uri"`
	VerificationUriComplete string `json:"verification_uri_complete"`
	ExpiresIn               int    `json:"expires_in"`
	Interval                int    `json:"interval"`
}

type DeviceTokenResponse struct {
	TokenType    string `json:"token_type"`
	IdToken      string `json:"id_token"`
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	Scope        string `json:"scope"`
	ExpiresIn    int    `json:"expires_in"`
}

type OidcClient interface {
	BeginDeviceFlow(ctx context.Context, clientId string, scope string) (DeviceAuthorizationResponse, error)
	PollDeviceToken(ctx context.Context, clientId string, deviceCode string) (DeviceTokenResponse, error)
	PostActivate(ctx context.Context, userCode string) (loginToken string, err error)
	VerifyPassword(ctx context.Context, loginToken string, username string, password string) error
	FinishLogin(ctx context.Context, loginToken string) error
}

type oidcClient struct {
	transport *Transport
}

func NewOidcClient(transport *Transport) OidcClient {
	return &oidcClient{transport: transport}
}

func (o *oidcClient) BeginDeviceFlow(ctx context.Context, clientId string, scope string) (DeviceAuthorizationResponse, error) {
	formValues := url.Values{
		"client_id": {clientId},
		"scope":     {scope},
	}

	req, err := o.transport.NewOidcRequest(ctx, http.MethodPost, "/device", strings.NewReader(formValues.Encode()))
	if err != nil {
		return DeviceAuthorizationResponse{}, fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := o.transport.Do(req)
	if err != nil {
		return DeviceAuthorizationResponse{}, fmt.Errorf("doing request: %w", err)
	}
	defer resp.Body.Close() //nolint:errcheck

	var result DeviceAuthorizationResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return DeviceAuthorizationResponse{}, fmt.Errorf("decoding response: %w", err)
	}

	return result, nil
}

func (o *oidcClient) PollDeviceToken(ctx context.Context, clientId string, deviceCode string) (DeviceTokenResponse, error) {
	formValues := url.Values{
		"grant_type":  {"urn:ietf:params:oauth:grant-type:device_code"},
		"client_id":   {clientId},
		"device_code": {deviceCode},
	}

	req, err := o.transport.NewOidcRequest(ctx, http.MethodPost, "/token", strings.NewReader(formValues.Encode()))
	if err != nil {
		return DeviceTokenResponse{}, fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := o.transport.DoRaw(req)
	if err != nil {
		return DeviceTokenResponse{}, fmt.Errorf("doing request: %w", err)
	}
	defer resp.Body.Close() //nolint:errcheck

	if resp.StatusCode == http.StatusBadRequest {
		var oauthErr struct {
			Error            string `json:"error"`
			ErrorDescription string `json:"error_description"`
		}
		_ = json.NewDecoder(resp.Body).Decode(&oauthErr)
		switch oauthErr.Error {
		case "authorization_pending":
			return DeviceTokenResponse{}, ErrAuthorizationPending
		case "access_denied":
			return DeviceTokenResponse{}, ErrAccessDenied
		case "expired_token":
			return DeviceTokenResponse{}, ErrExpiredToken
		case "slow_down":
			return DeviceTokenResponse{}, ErrSlowDown
		default:
			return DeviceTokenResponse{}, fmt.Errorf("oauth error %s: %s", oauthErr.Error, oauthErr.ErrorDescription)
		}
	}

	if resp.StatusCode != http.StatusOK {
		return DeviceTokenResponse{}, ApiError{Message: resp.Status, Code: resp.StatusCode}
	}

	var result DeviceTokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return DeviceTokenResponse{}, fmt.Errorf("decoding response: %w", err)
	}

	return result, nil
}

func (o *oidcClient) PostActivate(ctx context.Context, userCode string) (string, error) {
	formValues := url.Values{"user_code": {userCode}}

	req, err := o.transport.NewOidcRequest(ctx, http.MethodPost, "/activate", strings.NewReader(formValues.Encode()))
	if err != nil {
		return "", fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := o.transport.DoNoRedirect(req)
	if err != nil {
		return "", fmt.Errorf("doing request: %w", err)
	}
	defer resp.Body.Close() //nolint:errcheck

	if resp.StatusCode == http.StatusNotFound {
		return "", ErrInvalidUserCode
	}
	if resp.StatusCode != http.StatusFound {
		return "", ApiError{Message: resp.Status, Code: resp.StatusCode}
	}

	location := resp.Header.Get("Location")
	parsed, err := url.Parse(location)
	if err != nil {
		return "", fmt.Errorf("parsing redirect location: %w", err)
	}

	token := parsed.Query().Get("token")
	if token == "" {
		return "", fmt.Errorf("no login token in redirect location")
	}

	return token, nil
}

func (o *oidcClient) VerifyPassword(ctx context.Context, loginToken string, username string, password string) error {
	body, err := json.Marshal(map[string]string{"username": username, "password": password})
	if err != nil {
		return fmt.Errorf("marshaling request: %w", err)
	}

	req, err := o.transport.NewRootRequest(ctx, http.MethodPost, fmt.Sprintf("/logins/%s/verify-password", loginToken), bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}

	resp, err := o.transport.DoRaw(req)
	if err != nil {
		return fmt.Errorf("doing request: %w", err)
	}
	defer resp.Body.Close() //nolint:errcheck

	if resp.StatusCode == http.StatusUnauthorized {
		return fmt.Errorf("invalid credentials")
	}
	if resp.StatusCode >= 400 {
		return ApiError{Message: resp.Status, Code: resp.StatusCode}
	}

	return nil
}

func (o *oidcClient) FinishLogin(ctx context.Context, loginToken string) error {
	req, err := o.transport.NewRootRequest(ctx, http.MethodPost, fmt.Sprintf("/logins/%s/finish-login", loginToken), nil)
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}

	resp, err := o.transport.DoNoRedirect(req)
	if err != nil {
		return fmt.Errorf("doing request: %w", err)
	}
	defer resp.Body.Close() //nolint:errcheck

	if resp.StatusCode >= 400 {
		return ApiError{Message: resp.Status, Code: resp.StatusCode}
	}

	return nil
}
