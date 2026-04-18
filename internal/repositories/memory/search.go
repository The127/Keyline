package memory

import (
	"github.com/The127/Keyline/internal/repositories"
	"strings"
)

func matchesSearch(value string, sf repositories.SearchFilter) bool {
	term := sf.Term()
	switch {
	case strings.HasPrefix(term, "%") && strings.HasSuffix(term, "%"):
		// contains
		inner := term[1 : len(term)-1]
		return strings.Contains(strings.ToLower(value), strings.ToLower(inner))
	case strings.HasSuffix(term, "%"):
		// starts with
		prefix := term[:len(term)-1]
		return strings.HasPrefix(strings.ToLower(value), strings.ToLower(prefix))
	case strings.HasPrefix(term, "%"):
		// ends with
		suffix := term[1:]
		return strings.HasSuffix(strings.ToLower(value), strings.ToLower(suffix))
	default:
		// exact
		return strings.EqualFold(value, term)
	}
}

func paginateSlice[T any](items []T, pi repositories.PagingInfo) []T {
	if pi.IsZero() {
		return items
	}
	page := pi.Page()
	size := pi.Size()
	if page < 1 {
		page = 1
	}
	offset := (page - 1) * size
	if offset >= len(items) {
		return []T{}
	}
	end := offset + size
	if end > len(items) {
		end = len(items)
	}
	return items[offset:end]
}
