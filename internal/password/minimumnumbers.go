package password

import "fmt"

type minimumNumbersPolicy struct {
	minAmount int
}

func (p *minimumNumbersPolicy) Validate(password string) error {
	amount := 0

	for _, c := range password {
		if c >= '0' && c <= '9' {
			amount++
		}
	}

	if amount < p.minAmount {
		return fmt.Errorf("password must contain at least %d special characters", p.minAmount)
	}

	return nil
}
