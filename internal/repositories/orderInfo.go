package repositories

import "github.com/huandu/go-sqlbuilder"

type orderInfo struct {
	orderBy  string
	orderDir string
}

func (i *orderInfo) apply(s *sqlbuilder.SelectBuilder) {
	if i.orderBy == "" {
		return
	}

	s.OrderBy(i.orderBy)
	if i.orderDir != "" {
		if i.orderDir == "asc" {
			s.Asc()
		} else {
			s.Desc()
		}
	}
}
