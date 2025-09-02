package utils

import "strings"

func TrimSpace(s *string) *string {
	if s == nil {
		return nil
	}

	return Ptr(strings.TrimSpace(*s))
}
