package api

import (
	"time"

	"github.com/google/uuid"
)

type CreateVirtualServerRequestDtoAdminDto struct {
	Username     string   `json:"username" validate:"required,min=1,max=255"`
	DisplayName  string   `json:"displayName" validate:"required,min=1,max=255"`
	PrimaryEmail string   `json:"primaryEmail" validate:"required,email"`
	PasswordHash string   `json:"passwordHash" validate:"required"`
	Roles        []string `json:"roles"`
}

type CreateVirtualServerRequestDtoServiceUserDto struct {
	Username  string   `json:"username" validate:"required,min=1,max=255"`
	Roles     []string `json:"roles"`
	PublicKey struct {
		Pem string `json:"pem" validate:"required"`
		Kid string `json:"kid" validate:"required"`
	} `json:"publicKey" validate:"required"`
}

type CreateVirtualServerRequestDtoProjectDtoRoleDto struct {
	Name        string `json:"name" validate:"required,min=1,max=255"`
	Description string `json:"description"`
}

type CreateVirtualServerRequestDtoProjectDtoApplicationDto struct {
	Name           string   `json:"name" validate:"required,min=1,max=255"`
	DisplayName    string   `json:"displayName" validate:"required,min=1,max=255"`
	Type           string   `json:"type" validate:"required,oneof=public confidential"`
	HashedSecret   *string  `json:"hashedSecret"`
	RedirectUris   []string `json:"redirectUris" validate:"required,dive,url,min=1"`
	PostLogoutUris []string `json:"postLogoutUris" validate:"dive,url"`
}

type CreateVirtualServerRequestDtoProjectDtoResourceServerDto struct {
	Slug        string `json:"slug" validate:"required,min=1,max=255"`
	Name        string `json:"name" validate:"required,min=1,max=255"`
	Description string `json:"description"`
}

type CreateVirtualServerRequestDtoProjectDto struct {
	Slug        string `json:"slug" validate:"required,min=1,max=255"`
	Name        string `json:"name" validate:"required,min=1,max=255"`
	Description string `json:"description"`

	Roles           []CreateVirtualServerRequestDtoProjectDtoRoleDto           `json:"roles"`
	Applications    []CreateVirtualServerRequestDtoProjectDtoApplicationDto    `json:"applications"`
	ResourceServers []CreateVirtualServerRequestDtoProjectDtoResourceServerDto `json:"resourceServers"`
}

type CreateVirtualServerRequestDto struct {
	Name                        string   `json:"name" validate:"required,min=1,max=255,alphanum"`
	DisplayName                 string   `json:"displayName" validate:"required,min=1,max=255"`
	EnableRegistration          bool     `json:"enableRegistration"`
	PrimarySigningAlgorithm     *string  `json:"primarySigningAlgorithm" validate:"omitempty,oneof=RS256 EdDSA"`
	AdditionalSigningAlgorithms []string `json:"additionalSigningAlgorithms" validate:"omitempty,dive,oneof=RS256 EdDSA"`
	Require2fa                  bool     `json:"require2fa"`

	Admin        *CreateVirtualServerRequestDtoAdminDto        `json:"admin"`
	ServiceUsers []CreateVirtualServerRequestDtoServiceUserDto `json:"serviceUsers"`
	Projects     []CreateVirtualServerRequestDtoProjectDto     `json:"projects"`
}

type GetVirtualServerResponseDto struct {
	Id                          uuid.UUID `json:"id"`
	Name                        string    `json:"name"`
	DisplayName                 string    `json:"displayName"`
	RegistrationEnabled         bool      `json:"registrationEnabled"`
	Require2fa                  bool      `json:"require2fa"`
	RequireEmailVerification    bool      `json:"requireEmailVerification"`
	PrimarySigningAlgorithm     string    `json:"primarySigningAlgorithm"`
	AdditionalSigningAlgorithms []string  `json:"additionalSigningAlgorithms"`
	CreatedAt                   time.Time `json:"createdAt"`
	UpdatedAt                   time.Time `json:"updatedAt"`
}

type GetVirtualServerListResponseDto struct {
	Name                string `json:"name"`
	DisplayName         string `json:"displayName"`
	RegistrationEnabled bool   `json:"registrationEnabled"`
}

type PatchVirtualServerRequestDto struct {
	DisplayName *string `json:"displayName"`

	EnableRegistration       *bool `json:"enableRegistration"`
	Require2fa               *bool `json:"require2fa"`
	RequireEmailVerification *bool `json:"requireEmailVerification"`

	PrimarySigningAlgorithm     *string   `json:"primarySigningAlgorithm" validate:"omitempty,oneof=RS256 EdDSA"`
	AdditionalSigningAlgorithms *[]string `json:"additionalSigningAlgorithms" validate:"omitempty,dive,oneof=RS256 EdDSA"`
}
