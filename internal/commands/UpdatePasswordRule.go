package commands

import (
	"Keyline/internal/database"
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
	dbContext := ioc.GetDependency[database.Context](scope)

	virtualServerFilter := repositories.NewVirtualServerFilter().Name(command.VirtualServerName)
	virtualServer, err := dbContext.VirtualServers().Single(ctx, virtualServerFilter)
	if err != nil {
		return nil, fmt.Errorf("getting virtual server: %w", err)
	}

	passwordRuleFilter := repositories.NewPasswordRuleFilter().
		Type(command.Type).
		VirtualServerId(virtualServer.Id())
	passwordRule, err := dbContext.PasswordRules().Single(ctx, passwordRuleFilter)
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

	dbContext.PasswordRules().Update(passwordRule)
	return &UpdatePasswordRuleResponse{}, nil
}
