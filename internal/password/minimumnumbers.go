package password

import (
	"Keyline/internal/repositories"
	"encoding/json"
	"fmt"
)

type minimumNumbersPolicy struct {
	MinAmount int `json:"minAmount"`
}

func (p *minimumNumbersPolicy) GetPasswordRuleType() repositories.PasswordRuleType {
	return repositories.PasswordRuleTypeDigits
}

func (p *minimumNumbersPolicy) Serialize() ([]byte, error) {
	jsonBytes, err := json.Marshal(p)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize max length rule: %w", err)
	}
	return jsonBytes, nil
}

func (p *minimumNumbersPolicy) Validate(password string) error {
	amount := 0

	for _, c := range password {
		if c >= '0' && c <= '9' {
			amount++
		}
	}

	if amount < p.MinAmount {
		return fmt.Errorf("password must contain at least %d numeric characters", p.MinAmount)
	}

	return nil
}
