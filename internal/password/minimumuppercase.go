package password

import (
	"Keyline/internal/repositories"
	"encoding/json"
	"fmt"
)

type minimumUpperCasePolicy struct {
	MinAmount int `json:"minAmount"`
}

func (p *minimumUpperCasePolicy) GetPasswordRuleType() repositories.PasswordRuleType {
	return repositories.PasswordRuleTypeUpperCase
}

func (p *minimumUpperCasePolicy) Serialize() ([]byte, error) {
	jsonBytes, err := json.Marshal(p)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize upper case rule: %w", err)
	}
	return jsonBytes, nil
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
