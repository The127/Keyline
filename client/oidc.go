package client

import (
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
	defer resp.Body.Close()

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
	defer resp.Body.Close()

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
