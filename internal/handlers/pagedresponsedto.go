package handlers

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

func NewPagedResponseDto[T any](items []T, queryOps *QueryOps, totalItems int) PagedResponseDto[T] {
	var pagination *Pagination
	if queryOps.PageSize > 0 {
		pagination = &Pagination{
			Size:       queryOps.PageSize,
			Page:       queryOps.Page,
			TotalPages: totalItems/queryOps.PageSize + 1,
			TotalItems: totalItems,
		}
	}

	return PagedResponseDto[T]{
		Items:      items,
		Pagination: pagination,
	}
}
