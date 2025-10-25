package password

import (
	"Keyline/internal/repositories"
	"encoding/json"
	"fmt"
)

type minimumLowerCasePolicy struct {
	MinAmount int `json:"minAmount"`
}

func (p *minimumLowerCasePolicy) GetPasswordRuleType() repositories.PasswordRuleType {
	return repositories.PasswordRuleTypeLowerCase
}

func (p *minimumLowerCasePolicy) Serialize() ([]byte, error) {
	jsonBytes, err := json.Marshal(p)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize max length rule: %w", err)
	}
	return jsonBytes, nil
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
