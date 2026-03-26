package commands

import (
	"Keyline/internal/authentication/permissions"
	"Keyline/internal/behaviours"
	"Keyline/internal/database"
	"Keyline/internal/middlewares"
	"Keyline/internal/repositories"
	"context"
	"fmt"

	"github.com/The127/ioc"
)

type DeletePasswordRule struct {
	VirtualServerName string
	Type              repositories.PasswordRuleType
}

func (a DeletePasswordRule) LogRequest() bool {
	return true
}

func (a DeletePasswordRule) LogResponse() bool {
	return true
}

func (a DeletePasswordRule) IsAllowed(ctx context.Context) (behaviours.PolicyResult, error) {
	return behaviours.PermissionBasedPolicy(ctx, permissions.VirtualServerUpdate)
}

func (a DeletePasswordRule) GetRequestName() string {
	return "DeletePasswordRule"
}

type DeletePasswordRuleResponse struct{}

func HandleDeletePasswordRule(ctx context.Context, command DeletePasswordRule) (*DeletePasswordRuleResponse, error) {
	scope := middlewares.GetScope(ctx)
	dbContext := ioc.GetDependency[database.Context](scope)

	virtualServerFilter := repositories.NewVirtualServerFilter().Name(command.VirtualServerName)
	virtualServer, err := dbContext.VirtualServers().FirstOrErr(ctx, virtualServerFilter)
	if err != nil {
		return nil, fmt.Errorf("getting virtual server: %w", err)
	}

	passwordRuleFilter := repositories.NewPasswordRuleFilter().
		VirtualServerId(virtualServer.Id()).
		Type(command.Type)
	passwordRule, err := dbContext.PasswordRules().FirstOrNil(ctx, passwordRuleFilter)
	if err != nil {
		return nil, fmt.Errorf("getting password rule: %w", err)
	}

	if passwordRule == nil {
		return &DeletePasswordRuleResponse{}, nil
	}

	dbContext.PasswordRules().Delete(passwordRule.Id())

	return &DeletePasswordRuleResponse{}, nil
}
