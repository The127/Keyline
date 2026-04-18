package api

import "github.com/google/uuid"

type PagedGroupsResponseDto = PagedResponseDto[ListGroupsResponseDto]

type ListGroupsResponseDto struct {
	Id   uuid.UUID `json:"id"`
	Name string    `json:"name"`
}
