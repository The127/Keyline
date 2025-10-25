package password

import "fmt"

type minimumUpperCasePolicy struct {
	MinAmount int `json:"minAmount"`
}

func (p *minimumUpperCasePolicy) Validate(password string) error {
	amount := 0

	for _, c := range password {
		if c >= 'A' && c <= 'Z' {
			amount++
		}
	}

	if amount < p.MinAmount {
		return fmt.Errorf("password must contain at least %d uppercase characters", p.MinAmount)
	}

	return nil
}
