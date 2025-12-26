package queries

import (
	"Keyline/internal/authentication/permissions"
	"Keyline/internal/behaviours"
	"Keyline/internal/database"
	"Keyline/internal/middlewares"
	"Keyline/internal/repositories"
	"Keyline/utils"
	"context"
	"fmt"

	"github.com/The127/ioc"

	"github.com/google/uuid"
)

type ListPasswordRules struct {
	VirtualServerName string
}

func (a ListPasswordRules) LogRequest() bool {
	return false
}

func (a ListPasswordRules) LogResponse() bool {
	return false
}

func (a ListPasswordRules) IsAllowed(ctx context.Context) (behaviours.PolicyResult, error) {
	return behaviours.PermissionBasedPolicy(ctx, permissions.VirtualServerView)
}

func (a ListPasswordRules) GetRequestName() string {
	return "ListPasswordRules"
}

type ListPasswordRulesResponse struct {
	Items []ListPasswordRulesResponseItem
}

type ListPasswordRulesResponseItem struct {
	Id      uuid.UUID
	Type    repositories.PasswordRuleType
	Details []byte
}

func HandleListPasswordRules(ctx context.Context, query ListPasswordRules) (*ListPasswordRulesResponse, error) {
	scope := middlewares.GetScope(ctx)
	dbContext := ioc.GetDependency[database.Context](scope)

	virtualServerFilter := repositories.NewVirtualServerFilter().Name(query.VirtualServerName)
	virtualServer, err := dbContext.VirtualServers().Single(ctx, virtualServerFilter)
	if err != nil {
		return nil, fmt.Errorf("getting virtual server: %w", err)
	}

	passwordRuleFilter := repositories.NewPasswordRuleFilter().
		VirtualServerId(virtualServer.Id())
	passwordRules, err := dbContext.PasswordRules().List(ctx, passwordRuleFilter)
	if err != nil {
		return nil, fmt.Errorf("getting password rules: %w", err)
	}

	items := utils.MapSlice(passwordRules, func(x *repositories.PasswordRule) ListPasswordRulesResponseItem {
		return ListPasswordRulesResponseItem{
			Id:      x.Id(),
			Type:    x.Type(),
			Details: x.Details(),
		}
	})

	return &ListPasswordRulesResponse{
		Items: items,
	}, nil
}
