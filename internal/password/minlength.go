package password

import "fmt"

type minLengthPolicy struct {
	minLength int
}

func (p *minLengthPolicy) Validate(password string) error {
	if len(password) < p.minLength {
		return fmt.Errorf("password must be at least %d characters long", p.minLength)
	}
	return nil
}

type characterClassPolicy struct {
	characterClasses []string
}
