package commands

import (
	"Keyline/internal/authentication/permissions"
	"Keyline/internal/behaviours"
	"Keyline/internal/middlewares"
	"Keyline/internal/password"
	"Keyline/internal/repositories"
	"Keyline/ioc"
	"Keyline/utils"
	"context"
	"encoding/json"
	"fmt"
)

type CreatePasswordRule struct {
	VirtualServerName string
	Type              repositories.PasswordRuleType
	Details           map[string]interface{}
}

func (a CreatePasswordRule) LogRequest() bool {
	return true
}

func (a CreatePasswordRule) LogResponse() bool {
	return true
}

func (a CreatePasswordRule) IsAllowed(ctx context.Context) (behaviours.PolicyResult, error) {
	return behaviours.PermissionBasedPolicy(ctx, permissions.VirtualServerUpdate)
}

func (a CreatePasswordRule) GetRequestName() string {
	return "CreatePasswordRule"
}

type CreatePasswordRuleResponse struct{}

func HandleCreatePasswordRule(ctx context.Context, command CreatePasswordRule) (*CreatePasswordRuleResponse, error) {
	scope := middlewares.GetScope(ctx)

	// TODO: validation

	virtualServerRepository := ioc.GetDependency[repositories.VirtualServerRepository](scope)
	virtualServerFilter := repositories.NewVirtualServerFilter().Name(command.VirtualServerName)
	virtualServer, err := virtualServerRepository.Single(ctx, virtualServerFilter)
	if err != nil {
		return nil, fmt.Errorf("getting virtual server: %w", err)
	}

	passwordRuleRepository := ioc.GetDependency[repositories.PasswordRuleRepository](scope)
	passwordRuleFilter := repositories.NewPasswordRuleFilter().
		VirtualServerId(virtualServer.Id()).
		Type(command.Type)
	passwordRule, err := passwordRuleRepository.First(ctx, passwordRuleFilter)
	if err != nil {
		return nil, fmt.Errorf("getting password rule: %w", err)
	}

	if passwordRule != nil {
		return nil, fmt.Errorf("password rule already exists: %w", utils.ErrHttpConflict)
	}

	jsonBytes, err := json.Marshal(command.Details)
	if err != nil {
		return nil, fmt.Errorf("marshaling details: %w", err)
	}

	details, err := password.DeserializePolicy(command.Type, jsonBytes)
	if err != nil {
		return nil, fmt.Errorf("deserializing details: %w", err)
	}

	passwordRule, err = repositories.NewPasswordRule(virtualServer.Id(), details)
	if err != nil {
		return nil, fmt.Errorf("creating password rule: %w", err)
	}

	err = passwordRuleRepository.Insert(ctx, passwordRule)
	if err != nil {
		return nil, fmt.Errorf("inserting password rule: %w", err)
	}

	return &CreatePasswordRuleResponse{}, nil
}
