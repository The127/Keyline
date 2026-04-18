package api

import (
	"time"

	"github.com/google/uuid"
)

type CreateProjectRequestDto struct {
	Slug        string `json:"slug" validate:"required,min=1,max=255"`
	Name        string `json:"name" validate:"required,min=1,max=255"`
	Description string `json:"description"`
}

type CreateProjectResponseDto struct {
	Id uuid.UUID `json:"id"`
}

type PagedProjectsResponseDto = PagedResponseDto[ListProjectsResponseDto]

type ListProjectsResponseDto struct {
	Id            uuid.UUID `json:"id"`
	Slug          string    `json:"slug"`
	Name          string    `json:"name"`
	SystemProject bool      `json:"systemProject"`
}

type GetProjectResponseDto struct {
	Id            uuid.UUID `json:"id"`
	Slug          string    `json:"slug"`
	Name          string    `json:"name"`
	Description   string    `json:"description"`
	SystemProject bool      `json:"systemProject"`

	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}
