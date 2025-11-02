package utils

import (
	"strings"
)

func InvariantBase64Equals(a, b string) bool {
	a = invariantBase64String(a)
	b = invariantBase64String(b)
	return a == b
}

func invariantBase64String(s string) string {
	stringLength := len(s)

	var b strings.Builder
	b.Grow(stringLength)

	for i := 0; i < stringLength; i++ {
		c := s[i]
		switch c {
		case '+':
			b.WriteByte('-')

		case '/':
			b.WriteByte('_')

		case '=':
			continue

		default:
			b.WriteByte(c)
		}
	}

	return b.String()
}
