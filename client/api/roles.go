package api

import (
	"time"

	"github.com/google/uuid"
)

type GetRoleByIdResponseDto struct {
	Id          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

type PagedRolesResponseDto struct {
	Items      []ListRolesResponseDto `json:"items"`
	Pagination Pagination             `json:"pagination"`
}

type ListRolesResponseDto struct {
	Id   uuid.UUID `json:"id"`
	Name string    `json:"name"`
}

type CreateRoleRequestDto struct {
	Name        string `json:"name" validate:"required,min=1,max=255"`
	Description string `json:"description" validate:"max=1024"`
}

type CreateRoleResponseDto struct {
	Id uuid.UUID `json:"id"`
}

type PatchRoleRequestDto struct {
	Name        *string `json:"name,omitempty"`
	Description *string `json:"description,omitempty"`
}

type AssignRoleRequestDto struct {
	UserId uuid.UUID `json:"userId" validate:"required,uuid=4"`
}

type PagedUsersInRoleResponseDto = PagedResponseDto[ListUsersInRoleResponseDto]

type ListUsersInRoleResponseDto struct {
	Id          uuid.UUID `json:"id"`
	Username    string    `json:"username"`
	DisplayName string    `json:"displayName"`
}
