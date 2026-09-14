package api

import (
	"time"

	"github.com/google/uuid"
)

// PagedTemplatesResponseDto is the paged envelope for ListTemplates.
type PagedTemplatesResponseDto struct {
	Items      []ListTemplatesResponseDto `json:"items"`
	Pagination Pagination                 `json:"pagination"`
}

type GetTemplateResponseDto struct {
	Id        uuid.UUID `json:"id"`
	Type      string    `json:"type"`
	Text      string    `json:"text"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type ListTemplatesResponseDto struct {
	Id   uuid.UUID `json:"id"`
	Type string    `json:"type"`
}
