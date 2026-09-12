package api

import "github.com/google/uuid"

type CreateIdentityProviderRequestDto struct {
	Name                  string   `json:"name" validate:"max=255"`
	DisplayName           string   `json:"displayName" validate:"max=255"`
	Preset                string   `json:"preset,omitempty"`
	Issuer                string   `json:"issuer,omitempty"`
	AuthorizationEndpoint string   `json:"authorizationEndpoint,omitempty"`
	TokenEndpoint         string   `json:"tokenEndpoint,omitempty"`
	UserinfoEndpoint      string   `json:"userinfoEndpoint,omitempty"`
	Scopes                []string `json:"scopes"`
	ClientId              string   `json:"clientId" validate:"required"`
	ClientSecret          string   `json:"clientSecret" validate:"required"`
}

type CreateIdentityProviderResponseDto struct {
	Id uuid.UUID `json:"id"`
}

type GetIdentityProviderResponseDto struct {
	Name                  string   `json:"name"`
	DisplayName           string   `json:"displayName"`
	Preset                string   `json:"preset,omitempty"`
	Issuer                string   `json:"issuer,omitempty"`
	AuthorizationEndpoint string   `json:"authorizationEndpoint"`
	TokenEndpoint         string   `json:"tokenEndpoint"`
	UserinfoEndpoint      string   `json:"userinfoEndpoint"`
	Scopes                []string `json:"scopes"`
	ClientId              string   `json:"clientId"`
}
