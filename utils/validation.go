package utils

import (
	"fmt"

	"github.com/go-playground/validator/v10"
)

var validate = validator.New()

func ValidateDto(s any) error {
	err := validate.Struct(s)
	if err != nil {
		return fmt.Errorf("invalid request: %s: %w", err.Error(), ErrHttpBadRequest)
	}

	// TODO: make an api friendly error type

	return nil
}
