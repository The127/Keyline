package password

import (
	"Keyline/internal/middlewares"
	"Keyline/internal/repositories"
	"Keyline/ioc"
	"context"
	"encoding/json"
	"errors"
	"fmt"
)

//go:generate mockgen -destination=./mock/mock_validator.go -package=mock . Validator
type Validator interface {
	Validate(password string) error
}

type validator struct {
	rules []Policy
}

func NewValidator(ctx context.Context) (Validator, error) {
	scope := middlewares.GetScope(ctx)

	vsName, err := middlewares.GetVirtualServerName(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get virtual server name: %w", err)
	}

	virtualServerRepository := ioc.GetDependency[repositories.VirtualServerRepository](scope)
	virtualServerFilter := repositories.NewVirtualServerFilter().Name(vsName)
	virtualServer, err := virtualServerRepository.Single(ctx, virtualServerFilter)
	if err != nil {
		return nil, fmt.Errorf("failed to get virtual server: %w", err)
	}

	passwordRuleRepository := ioc.GetDependency[repositories.PasswordRuleRepository](scope)
	passwordRuleFilter := repositories.NewPasswordRuleFilter().VirtualServerId(virtualServer.Id())
	passwordRules, err := passwordRuleRepository.List(ctx, passwordRuleFilter)
	if err != nil {
		return nil, fmt.Errorf("failed to get password rules: %w", err)
	}

	rules := make([]Policy, len(passwordRules))
	for _, passwordRule := range passwordRules {
		switch passwordRule.Type() {
		case repositories.PasswordRuleTypeMinLength:
			var minLengthRule minLengthPolicy
			err := json.Unmarshal(passwordRule.Details(), &minLengthRule)
			if err != nil {
				return nil, fmt.Errorf("failed to unmarshal min length rule: %w", err)
			}
			rules = append(rules, &minLengthRule)

		case repositories.PasswordRuleTypeMaxLength:
			var maxLengthRule maxLengthPolicy
			err := json.Unmarshal(passwordRule.Details(), &maxLengthRule)
			if err != nil {
				return nil, fmt.Errorf("failed to unmarshal max length rule: %w", err)
			}
			rules = append(rules, &maxLengthRule)

		case repositories.PasswordRuleTypeDigits:
			var numberRule minimumNumbersPolicy
			err := json.Unmarshal(passwordRule.Details(), &numberRule)
			if err != nil {
				return nil, fmt.Errorf("failed to unmarshal number rule: %w", err)
			}
			rules = append(rules, &numberRule)

		case repositories.PasswordRuleTypeLowerCase:
			var lowerCaseRule minimumLowerCasePolicy
			err := json.Unmarshal(passwordRule.Details(), &lowerCaseRule)
			if err != nil {
				return nil, fmt.Errorf("failed to unmarshal lower case rule: %w", err)
			}
			rules = append(rules, &lowerCaseRule)

		case repositories.PasswordRuleTypeUpperCase:
			var upperCaseRule minimumUpperCasePolicy
			err := json.Unmarshal(passwordRule.Details(), &upperCaseRule)
			if err != nil {
				return nil, fmt.Errorf("failed to unmarshal upper case rule: %w", err)
			}
			rules = append(rules, &upperCaseRule)

		case repositories.PasswordRuleTypeSpecial:
			var specialRule minimumSpecialPolicy
			err := json.Unmarshal(passwordRule.Details(), &specialRule)
			if err != nil {
				return nil, fmt.Errorf("failed to unmarshal special rule: %w", err)
			}
			rules = append(rules, &specialRule)

		default:
			return nil, fmt.Errorf("unknown password rule type: %s", passwordRule.Type())
		}

		// always add common policy
		rules = append(rules, &commonPolicy{})
	}

	return &validator{
		rules: rules,
	}, nil
}

func (v *validator) Validate(password string) error {
	var aggregateErr []error

	for _, rule := range v.rules {
		err := rule.Validate(password)
		if err != nil {
			aggregateErr = append(aggregateErr, err)
		}
	}

	return errors.Join(aggregateErr...)
}

//go:generate mockgen -destination=./mock/mock_policy.go -package=mock . Policy
type Policy interface {
	Validate(password string) error
}
