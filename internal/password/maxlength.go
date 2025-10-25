package password

import (
	"Keyline/internal/repositories"
	"encoding/json"
	"fmt"
)

type maxLengthPolicy struct {
	MaxLength int `json:"maxLength"`
}

func (p *maxLengthPolicy) GetPasswordRuleType() repositories.PasswordRuleType {
	return repositories.PasswordRuleTypeMaxLength
}

func (p *maxLengthPolicy) Serialize() ([]byte, error) {
	jsonBytes, err := json.Marshal(p)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize max length rule: %w", err)
	}
	return jsonBytes, nil
}

func (p *maxLengthPolicy) Validate(password string) error {
	if len(password) > p.MaxLength {
		return fmt.Errorf("password must be at most %d characters long", p.MaxLength)
	}
	return nil
}
