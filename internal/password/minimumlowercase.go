package password

import "fmt"

type minimumLowerCasePolicy struct {
	minAmount int
}

func (p *minimumLowerCasePolicy) Validate(password string) error {
	amount := 0
	for _, c := range password {
		if c >= 'a' && c <= 'z' {
			amount++
		}
	}

	if amount < p.minAmount {
		return fmt.Errorf("password must contain at least %d lowercase characters", p.minAmount)
	}

	return nil
}
