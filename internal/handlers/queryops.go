package handlers

import (
	queries2 "Keyline/internal/queries"
	"fmt"
	"net/http"
	"strconv"
)

type QueryOps struct {
	PageSize int
	Page     int
	OrderBy  string
	OrderDir string
	Search   string
}

func (q *QueryOps) ToPagedQuery() queries2.PagedQuery {
	return queries2.PagedQuery{
		PageSize: q.PageSize,
		Page:     q.Page,
	}
}

func (q *QueryOps) ToOrderedQuery() queries2.OrderedQuery {
	return queries2.OrderedQuery{
		OrderBy:  q.OrderBy,
		OrderDir: q.OrderDir,
	}
}

func ParseQueryOps(r *http.Request) (*QueryOps, error) {
	err := r.ParseForm()
	if err != nil {
		return nil, fmt.Errorf("parsing form: %w", err)
	}

	pageSize := 0
	page := 0

	pageSizeString := r.Form.Get("pageSize")
	if pageSizeString != "" {
		pageSize, err = strconv.Atoi(pageSizeString)
		if err != nil {
			return nil, fmt.Errorf("parsing page size: %w", err)
		}

		if pageSize < 0 {
			pageSize = 0
		}
	}

	pageString := r.Form.Get("page")
	if pageString != "" {
		page, err = strconv.Atoi(pageString)
		if err != nil {
			return nil, fmt.Errorf("parsing page: %w", err)
		}

		if page < 0 {
			page = 0
		}
	}

	if pageSize > 0 && page == 0 {
		page = 1
	}

	if page > 0 && pageSize == 0 {
		pageSize = 10
	}

	orderBy := r.Form.Get("orderBy")
	orderDir := r.Form.Get("orderDir")
	if orderDir != "asc" && orderDir != "desc" {
		orderDir = "asc"
	}

	search := r.Form.Get("search")

	return &QueryOps{
		PageSize: pageSize,
		Page:     page,
		OrderBy:  orderBy,
		OrderDir: orderDir,
		Search:   search,
	}, nil
}
