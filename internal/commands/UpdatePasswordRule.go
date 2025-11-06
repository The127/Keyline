package commands

import (
	"Keyline/internal/middlewares"
	"Keyline/internal/password"
	"Keyline/internal/repositories"
	"context"
	"encoding/json"
	"fmt"
	"github.com/The127/ioc"
)

type UpdatePasswordRule struct {
	VirtualServerName string
	Type              repositories.PasswordRuleType
	Details           map[string]interface{}
}

type UpdatePasswordRuleResponse struct{}

func HandleUpdatePasswordRule(ctx context.Context, command UpdatePasswordRule) (*UpdatePasswordRuleResponse, error) {
	scope := middlewares.GetScope(ctx)

	virtualServerRepository := ioc.GetDependency[repositories.VirtualServerRepository](scope)
	virtualServerFilter := repositories.NewVirtualServerFilter().Name(command.VirtualServerName)
	virtualServer, err := virtualServerRepository.Single(ctx, virtualServerFilter)
	if err != nil {
		return nil, fmt.Errorf("getting virtual server: %w", err)
	}

	passwordRuleRepository := ioc.GetDependency[repositories.PasswordRuleRepository](scope)
	passwordRuleFilter := repositories.NewPasswordRuleFilter().
		Type(command.Type).
		VirtualServerId(virtualServer.Id())
	passwordRule, err := passwordRuleRepository.Single(ctx, passwordRuleFilter)
	if err != nil {
		return nil, fmt.Errorf("getting password rule: %w", err)
	}

	jsonBytes, err := json.Marshal(command.Details)
	if err != nil {
		return nil, fmt.Errorf("marshaling details: %w", err)
	}

	details, err := password.DeserializePolicy(command.Type, jsonBytes)
	if err != nil {
		return nil, fmt.Errorf("deserializing details: %w", err)
	}

	err = passwordRule.SetDetails(details)
	if err != nil {
		return nil, fmt.Errorf("setting details: %w", err)
	}

	err = passwordRuleRepository.Update(ctx, passwordRule)
	if err != nil {
		return nil, fmt.Errorf("updating password rule: %w", err)
	}

	return &UpdatePasswordRuleResponse{}, nil
}
