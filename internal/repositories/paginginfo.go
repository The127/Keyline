package repositories

import "github.com/huandu/go-sqlbuilder"

type PagingInfo struct {
	page int
	size int
}

func (i PagingInfo) Apply(s *sqlbuilder.SelectBuilder) {
	if i.page == 0 {
		return
	}

	s.Limit(i.size).Offset(i.offset())
}

func (i PagingInfo) offset() int {
	return (i.page - 1) * i.size
}
