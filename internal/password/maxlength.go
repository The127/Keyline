package password

import "fmt"

type maxLengthPolicy struct {
	maxLength int
}

func (p *maxLengthPolicy) Validate(password string) error {
	if len(password) > p.maxLength {
		return fmt.Errorf("password must be at most %d characters long", p.maxLength)
	}
	return nil
}
