package password

import "fmt"

type maxLengthPolicy struct {
	MaxLength int `json:"maxLength"`
}

func (p *maxLengthPolicy) Validate(password string) error {
	if len(password) > p.MaxLength {
		return fmt.Errorf("password must be at most %d characters long", p.MaxLength)
	}
	return nil
}
