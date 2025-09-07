package queries

type PagedResponse[T any] struct {
	Items      []T
	TotalCount int
}

func NewPagedResponse[T any](items []T, totalCount int) PagedResponse[T] {
	return PagedResponse[T]{
		Items:      items,
		TotalCount: totalCount,
	}
}
