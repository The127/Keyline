package pghelpers

import (
	"errors"

	"github.com/lib/pq"
)

const uniqueViolationCode = "23505"

func IsUniqueViolation(err error) bool {
	var pqErr *pq.Error
	if !errors.As(err, &pqErr) {
		return false
	}
	return string(pqErr.Code) == uniqueViolationCode
}
