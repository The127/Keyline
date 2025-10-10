package repositories

import "github.com/huandu/go-sqlbuilder"

type pagingInfo struct {
	page int
	size int
}

func (i *pagingInfo) apply(s *sqlbuilder.SelectBuilder) {
	if i.page == 0 {
		return
	}

	s.Limit(i.size).Offset(i.offset())
}

func (i *pagingInfo) offset() int {
	return (i.page - 1) * i.size
}
