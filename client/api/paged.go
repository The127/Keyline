package api

type PagedResponseDto[T any] struct {
	Items      []T         `json:"items"`
	Pagination *Pagination `json:"pagination"`
}

type Pagination struct {
	Size       int `json:"size"`
	Page       int `json:"page"`
	TotalPages int `json:"totalPages"`
	TotalItems int `json:"totalItems"`
}
