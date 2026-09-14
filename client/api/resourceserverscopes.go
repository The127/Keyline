package api

import (
	"time"

	"github.com/google/uuid"
)

type CreateResourceServerScopeRequestDto struct {
	Scope       string `json:"scope" validate:"required,min=1,max=255"`
	Name        string `json:"name" validate:"required,min=1,max=255"`
	Description string `json:"description"`
}

type CreateResourceServerScopeResponseDto struct {
	Id uuid.UUID `json:"id"`
}

type PagedResourceServerScopeResponseDto = PagedResponseDto[ListResourceServerScopesResponseDto]

type ListResourceServerScopesResponseDto struct {
	Id    uuid.UUID `json:"id"`
	Scope string    `json:"scope"`
	Name  string    `json:"name"`
}

type GetResourceServerScopeResponseDto struct {
	Id          uuid.UUID `json:"id"`
	Scope       string    `json:"scope"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}
