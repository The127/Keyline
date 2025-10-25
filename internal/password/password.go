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
	Validate(ctx context.Context, password string) error
}

type validator struct{}

func NewValidator() Validator {
	return &validator{}
}

func (v *validator) Validate(ctx context.Context, password string) error {
	scope := middlewares.GetScope(ctx)

	vsName, err := middlewares.GetVirtualServerName(ctx)
	if err != nil {
		return fmt.Errorf("failed to get virtual server name: %w", err)
	}

	virtualServerRepository := ioc.GetDependency[repositories.VirtualServerRepository](scope)
	virtualServerFilter := repositories.NewVirtualServerFilter().Name(vsName)
	virtualServer, err := virtualServerRepository.Single(ctx, virtualServerFilter)
	if err != nil {
		return fmt.Errorf("failed to get virtual server: %w", err)
	}

	passwordRuleRepository := ioc.GetDependency[repositories.PasswordRuleRepository](scope)
	passwordRuleFilter := repositories.NewPasswordRuleFilter().VirtualServerId(virtualServer.Id())
	passwordRules, err := passwordRuleRepository.List(ctx, passwordRuleFilter)
	if err != nil {
		return fmt.Errorf("failed to get password rules: %w", err)
	}

	var rules []Policy
	for _, passwordRule := range passwordRules {
		rule, err := DeserializePolicy(passwordRule.Type(), []byte(passwordRule.Details()))
		if err != nil {
			return fmt.Errorf("failed to deserialize password rule: %w", err)
		}
		rules = append(rules, rule)
	}

	// always add common policy
	rules = append(rules, &commonPolicy{})

	var aggregateErr []error

	for _, rule := range rules {
		err := rule.Validate(password)
		if err != nil {
			aggregateErr = append(aggregateErr, err)
		}
	}

	return errors.Join(aggregateErr...)
}

//go:generate mockgen -destination=./mock/mock_policy.go -package=mock . Policy
type Policy interface {
	repositories.PasswordRuleDetails
	Validate(password string) error
}

func DeserializePolicy(ruleType repositories.PasswordRuleType, jsonBytes []byte) (Policy, error) {
	switch ruleType {
	case repositories.PasswordRuleTypeMinLength:
		var minLengthRule minLengthPolicy
		err := json.Unmarshal(jsonBytes, &minLengthRule)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal min length rule: %w", err)
		}

	case repositories.PasswordRuleTypeMaxLength:
		var maxLengthRule maxLengthPolicy
		err := json.Unmarshal(jsonBytes, &maxLengthRule)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal max length rule: %w", err)
		}

	case repositories.PasswordRuleTypeDigits:
		var numberRule minimumNumbersPolicy
		err := json.Unmarshal(jsonBytes, &numberRule)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal number rule: %w", err)
		}
		return &numberRule, nil

	case repositories.PasswordRuleTypeLowerCase:
		var lowerCaseRule minimumLowerCasePolicy
		err := json.Unmarshal(jsonBytes, &lowerCaseRule)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal lower case rule: %w", err)
		}
		return &lowerCaseRule, nil

	case repositories.PasswordRuleTypeUpperCase:
		var upperCaseRule minimumUpperCasePolicy
		err := json.Unmarshal(jsonBytes, &upperCaseRule)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal upper case rule: %w", err)
		}
		return &upperCaseRule, nil

	case repositories.PasswordRuleTypeSpecial:
		var specialRule minimumSpecialPolicy
		err := json.Unmarshal(jsonBytes, &specialRule)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal special rule: %w", err)
		}
		return &specialRule, nil

	default:
		return nil, fmt.Errorf("unknown password rule type: %s", ruleType)
	}

	panic("unreachable")
}
