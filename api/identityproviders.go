package api

import "github.com/google/uuid"

type CreateIdentityProviderRequestDto struct {
	Name        string `json:"name" validate:"required,min=1,max=255"`
	DisplayName string `json:"displayName" validate:"required,min=1,max=255"`
}

type CreateIdentityProviderResponseDto struct {
	Id uuid.UUID `json:"id"`
}
