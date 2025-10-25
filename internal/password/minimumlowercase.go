package password

import "fmt"

type minimumLowerCasePolicy struct {
	MinAmount int `json:"minAmount"`
}

func (p *minimumLowerCasePolicy) Validate(password string) error {
	amount := 0
	for _, c := range password {
		if c >= 'a' && c <= 'z' {
			amount++
		}
	}

	if amount < p.MinAmount {
		return fmt.Errorf("password must contain at least %d lowercase characters", p.MinAmount)
	}

	return nil
}
