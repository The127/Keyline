package api

import (
	"time"

	"github.com/google/uuid"
)

type CreateResourceServerRequestDto struct {
	Slug        string `json:"slug" validate:"required,min=1,max=255"`
	Name        string `json:"name" validate:"required"`
	Description string `json:"description"`
}

type PagedResourceServersResponseDto = PagedResponseDto[ListResourceServersResponseDto]

type ListResourceServersResponseDto struct {
	Id   uuid.UUID `json:"id"`
	Slug string    `json:"slug"`
	Name string    `json:"name"`
}

type GetResourceServerResponseDto struct {
	Id          uuid.UUID `json:"id"`
	Slug        string    `json:"slug"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}
