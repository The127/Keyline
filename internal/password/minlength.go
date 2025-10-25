package password

import "fmt"

type minLengthPolicy struct {
	MinLength int `json:"minLength"`
}

func (p *minLengthPolicy) Validate(password string) error {
	if len(password) < p.MinLength {
		return fmt.Errorf("password must be at least %d characters long", p.MinLength)
	}
	return nil
}

type characterClassPolicy struct {
	characterClasses []string
}
