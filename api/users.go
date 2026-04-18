package api

import (
	"time"

	"github.com/google/uuid"
)

type RegisterUserRequestDto struct {
	Username    string `json:"username" validate:"required,min=1,max=255"`
	DisplayName string `json:"displayName" validate:"required,min=1,max=255"`
	Password    string `json:"password" validate:"required"`
	Email       string `json:"email" validate:"required"`
}

type CreateUserRequestDto struct {
	Username      string                       `json:"username" validate:"required"`
	DisplayName   string                       `json:"displayName" validate:"required"`
	Email         string                       `json:"email" validate:"required"`
	EmailVerified bool                         `json:"emailVerified" validate:"required"`
	Password      *CreateUserRequestDtoPasword `json:"password"`
}

type CreateUserRequestDtoPasword struct {
	Plain     string `json:"plain" validate:"required"`
	Temporary bool   `json:"temporary"`
}

type CreateUserResponseDto struct {
	Id uuid.UUID `json:"id"`
}

type ListUsersResponseDto struct {
	Id            uuid.UUID `json:"id"`
	Username      string    `json:"username"`
	DisplayName   string    `json:"displayName"`
	PrimaryEmail  string    `json:"primaryEmail"`
	IsServiceUser bool      `json:"isServiceUser"`
}

type PagedUsersResponseDto struct {
	Items      []ListUsersResponseDto `json:"items"`
	Pagination Pagination             `json:"pagination"`
}

type GetUserByIdResponseDto struct {
	Id            uuid.UUID `json:"id"`
	Username      string    `json:"username"`
	DisplayName   string    `json:"displayName"`
	PrimaryEmail  string    `json:"primaryEmail"`
	EmailVerified bool      `json:"emailVerified"`
	IsServiceUser bool      `json:"isServiceUser"`
	CreatedAt     time.Time `json:"createdAt"`
	UpdatedAt     time.Time `json:"updatedAt"`
}

type GetUserApplicationMetadataResponseDto map[string]any

type GetUserGlobalMetadataResponseDto map[string]any

type GetUserMetadataResponseDto struct {
	Metadata            map[string]any `json:"metadata,omitempty"`
	ApplicationMetadata map[string]any `json:"applicationMetadata,omitempty"`
}

type UpdateUserGlobalMetadataRequestDto map[string]any

type PatchUserGlobalMetadataRequestDto map[string]any

type UpdateUserApplicationMetadataRequestDto map[string]any

type PatchUserApplicationMetadataRequestDto map[string]any

type PatchUserRequestDto struct {
	DisplayName   *string `json:"displayName"`
	EmailVerified *bool   `json:"emailVerified"`
}

type CreateServiceUserRequestDto struct {
	Username string `json:"username" validate:"required,min=1,max=255"`
}

type CreateServiceUserResponseDto struct {
	Id uuid.UUID `json:"id"`
}

type AssociateServiceUserPublicKeyRequestDto struct {
	PublicKey string `json:"publicKey" validate:"required"`
}

type AssociateServiceUserPublicKeyResponseDto struct {
	Kid string `json:"kid"`
}

type PasskeyCreateChallengeResponseDto struct {
	Id          uuid.UUID `json:"id"`
	Challenge   string    `json:"challenge" validate:"required"`
	UserId      uuid.UUID `json:"userId"`
	Username    string    `json:"username"`
	DisplayName string    `json:"displayName"`
}

type PasskeyValidateChallengeRequestDto struct {
	Id               uuid.UUID `json:"id" validate:"required"`
	WebauthnResponse struct {
		Id       string `json:"id"`
		RawId    string `json:"rawId"`
		Response struct {
			ClientDataJSON     string   `json:"clientDataJSON"`
			AuthenticatorData  string   `json:"authenticatorData"`
			Transports         []string `json:"transports"`
			PublicKey          string   `json:"publicKey"`
			PublicKeyAlgorithm int      `json:"publicKeyAlgorithm"`
			AttestationObject  string   `json:"attestationObject"`
		} `json:"response"`
		AuthenticatorAttachment string `json:"authenticatorAttachment"`
		Type                    string `json:"type"`
	} `json:"webauthnResponse" validate:"required"`
}

type ListPasskeyResponseDto struct {
	Id uuid.UUID `json:"id"`
}

type PagedListPasskeyResponseDto struct {
	Items []ListPasskeyResponseDto `json:"items"`
}
