package password

import "fmt"

type minimumSpecialPolicy struct {
	minAmount int
}

func (p *minimumSpecialPolicy) Validate(password string) error {
	amount := 0

	for _, c := range password {
		if c >= '!' && c <= '/' || c >= ':' && c <= '@' || c >= '[' && c <= '`' {
			amount++
		}
	}

	if amount < p.minAmount {
		return fmt.Errorf("password must contain at least %d special characters", p.minAmount)
	}

	return nil
}
