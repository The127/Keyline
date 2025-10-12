package repositories

type SearchType string

const (
	SearchTypeContains SearchType = "contains"
	SearchTypeExact    SearchType = "exact"
	SearchTypeStart    SearchType = "start"
	SearchTypeEnd      SearchType = "end"
)

type SearchFilter struct {
	q          string
	searchType SearchType
}

func NewSearchFilter(q string, searchType SearchType) SearchFilter {
	return SearchFilter{
		q:          q,
		searchType: searchType,
	}
}

func NewExactSearchFilter(q string) SearchFilter {
	return NewSearchFilter(q, SearchTypeExact)
}

func NewContainsSearchFilter(q string) SearchFilter {
	return NewSearchFilter(q, SearchTypeContains)
}

func NewStartSearchFilter(q string) SearchFilter {
	return NewSearchFilter(q, SearchTypeStart)
}

func NewEndSearchFilter(q string) SearchFilter {
	return NewSearchFilter(q, SearchTypeEnd)
}

func (f SearchFilter) Term() string {
	switch f.searchType {
	case SearchTypeExact:
		return f.q

	case SearchTypeContains:
		return "%" + f.q + "%"

	case SearchTypeStart:
		return f.q + "%"

	case SearchTypeEnd:

		return "%" + f.q
	default:
		panic("unknown search type: " + string(f.searchType) + "")
	}
}
