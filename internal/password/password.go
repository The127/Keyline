package password

import "context"

//go:generate mockgen -destination=./mock/mock_validator.go -package=mock . Validator
type Validator interface {
	Validate(password string) error
}

type validator struct {
}

func NewValidator(ctx context.Context) Validator {
	// get shit from the db
	return &validator{}
}

func (v *validator) Validate(password string) error {
	panic("implement me")
}

//go:generate mockgen -destination=./mock/mock_policy.go -package=mock . Policy
type Policy interface {
	Validate(password string) error
}
