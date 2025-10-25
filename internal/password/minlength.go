package password

import (
	"Keyline/internal/repositories"
	"encoding/json"
	"fmt"
)

type minLengthPolicy struct {
	MinLength int `json:"minLength"`
}

func (p *minLengthPolicy) GetPasswordRuleType() repositories.PasswordRuleType {
	return repositories.PasswordRuleTypeMinLength
}

func (p *minLengthPolicy) Serialize() ([]byte, error) {
	jsonBytes, err := json.Marshal(p)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize min length rule: %w", err)
	}
	return jsonBytes, nil
}

func (p *minLengthPolicy) Validate(password string) error {
	if len(password) < p.MinLength {
		return fmt.Errorf("password must be at least %d characters long", p.MinLength)
	}
	return nil
}
