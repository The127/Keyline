package handlers

import "github.com/The127/Keyline/api"

func NewPagedResponseDto[T any](items []T, queryOps *QueryOps, totalItems int) api.PagedResponseDto[T] {
	var pagination *api.Pagination
	if queryOps.PageSize > 0 {
		pagination = &api.Pagination{
			Size:       queryOps.PageSize,
			Page:       queryOps.Page,
			TotalPages: totalItems/queryOps.PageSize + 1,
			TotalItems: totalItems,
		}
	}

	return api.PagedResponseDto[T]{
		Items:      items,
		Pagination: pagination,
	}
}
