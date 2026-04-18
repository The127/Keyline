package api

import (
	"time"

	"github.com/google/uuid"
)

type CreateApplicationRequestDto struct {
	Name                  string   `json:"name" validate:"required,min=1,max=255"`
	DisplayName           string   `json:"displayName" validate:"required,min=1,max=255"`
	RedirectUris          []string `json:"redirectUris" validate:"required,dive,url,min=1"`
	PostLogoutUris        []string `json:"postLogoutUris" validate:"dive,url"`
	Type                  string   `json:"type" validate:"required,oneof=public confidential"`
	AccessTokenHeaderType *string  `json:"accessTokenHeaderType" validate:"omitempty,oneof=at+jwt JWT"`
	DeviceFlowEnabled     bool     `json:"deviceFlowEnabled"`
}

type CreateApplicationResponseDto struct {
	Id     uuid.UUID `json:"id"`
	Secret *string   `json:"secret,omitempty"`
}

type GetApplicationResponseDto struct {
	Id          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	DisplayName string    `json:"displayName"`
	Type        string    `json:"type"`

	RedirectUris           []string `json:"redirectUris"`
	PostLogoutRedirectUris []string `json:"postLogoutRedirectUris"`

	SystemApplication bool `json:"systemApplication"`

	ClaimsMappingScript *string `json:"customClaimsMappingScript"`

	DeviceFlowEnabled bool `json:"deviceFlowEnabled"`

	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type PatchApplicationRequestDto struct {
	DisplayName         *string `json:"displayName"`
	ClaimsMappingScript *string `json:"customClaimsMappingScript"`
	DeviceFlowEnabled   *bool   `json:"deviceFlowEnabled"`
}

type PagedApplicationsResponseDto = PagedResponseDto[ListApplicationsResponseDto]

type ListApplicationsResponseDto struct {
	Id                uuid.UUID `json:"id"`
	Name              string    `json:"name"`
	DisplayName       string    `json:"displayName"`
	Type              string    `json:"type"`
	SystemApplication bool      `json:"systemApplication"`
}
