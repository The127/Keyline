package api

import "github.com/google/uuid"

type CreateIdentityProviderRequestDto struct {
	Name                  string   `json:"name" validate:"required,min=1,max=255,excludesall=/?#%"`
	DisplayName           string   `json:"displayName" validate:"required,min=1,max=255"`
	Preset                string   `json:"preset,omitempty"`
	AuthorizationEndpoint string   `json:"authorizationEndpoint,omitempty" validate:"omitempty,http_url"`
	TokenEndpoint         string   `json:"tokenEndpoint,omitempty" validate:"omitempty,http_url"`
	UserinfoEndpoint      string   `json:"userinfoEndpoint,omitempty" validate:"omitempty,http_url"`
	Scopes                []string `json:"scopes" validate:"dive,required"`
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
	AuthorizationEndpoint string   `json:"authorizationEndpoint"`
	TokenEndpoint         string   `json:"tokenEndpoint"`
	UserinfoEndpoint      string   `json:"userinfoEndpoint"`
	Scopes                []string `json:"scopes"`
	ClientId              string   `json:"clientId"`
}
