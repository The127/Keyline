package password

import (
	"Keyline/internal/repositories"
	"encoding/json"
	"fmt"
)

type minimumSpecialPolicy struct {
	MinAmount int `json:"minAmount"`
}

func (p *minimumSpecialPolicy) GetPasswordRuleType() repositories.PasswordRuleType {
	return repositories.PasswordRuleTypeSpecial
}

func (p *minimumSpecialPolicy) Serialize() ([]byte, error) {
	jsonBytes, err := json.Marshal(p)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize max length rule: %w", err)
	}
	return jsonBytes, nil
}

func (p *minimumSpecialPolicy) Validate(password string) error {
	amount := 0

	for _, c := range password {
		if c >= '!' && c <= '/' || c >= ':' && c <= '@' || c >= '[' && c <= '`' {
			amount++
		}
	}

	if amount < p.MinAmount {
		return fmt.Errorf("password must contain at least %d special characters", p.MinAmount)
	}

	return nil
}
