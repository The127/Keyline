package behaviours

import (
	"Keyline/internal/authentication"
	"Keyline/internal/authentication/permissions"
	"Keyline/internal/authentication/roles"
	"Keyline/internal/middlewares"
	"Keyline/ioc"
	"Keyline/mediator"
	"Keyline/utils"
	"context"
	"fmt"

	"github.com/google/uuid"
)

type AuditLogger interface {
	Log(ctx context.Context, policy Policy, result PolicyResult) error
}

type PolicyResult struct {
	allowed bool
	userId  uuid.UUID
	reason  AllowReason
}

func (p PolicyResult) IsAllowed() bool {
	return p.allowed
}

func (p PolicyResult) UserId() uuid.UUID {
	return p.userId
}

func (p PolicyResult) Reason() AllowReason {
	return p.reason
}

type AllowReason interface {
	ImplementsAllowReason()
}

type AllowedByAnyone struct{}

func NewAllowedByAnyone() AllowedByAnyone {
	return AllowedByAnyone{}
}

func (a AllowedByAnyone) String() string {
	return "Anyone"
}

func (a AllowedByAnyone) ImplementsAllowReason() {}

type AllowedByOwnership struct{}

func NewAllowedByOwnership() AllowedByOwnership {
	return AllowedByOwnership{}
}

func (a AllowedByOwnership) String() string {
	return "Ownership"
}

func (a AllowedByOwnership) ImplementsAllowReason() {}

type AllowedByPermission struct {
	Permission  permissions.Permission
	SourceRoles []roles.Role
}

func NewAllowedByPermission(permission permissions.Permission, sourceRoles []roles.Role) AllowedByPermission {
	return AllowedByPermission{
		Permission:  permission,
		SourceRoles: sourceRoles,
	}
}

func (a AllowedByPermission) String() string {
	return fmt.Sprintf("Permission: %s, SourceRoles: %v", a.Permission, a.SourceRoles)
}

func (a AllowedByPermission) ImplementsAllowReason() {}

func Allowed(userId uuid.UUID, reason AllowReason) PolicyResult {
	return PolicyResult{
		allowed: true,
		userId:  userId,
		reason:  reason,
	}
}

func Denied(userId uuid.UUID) PolicyResult {
	return PolicyResult{
		allowed: false,
		userId:  userId,
	}
}

type Policy interface {
	IsAllowed(ctx context.Context) (PolicyResult, error)
	GetRequestName() string
}

func PolicyBehaviour(ctx context.Context, request Policy, next mediator.Next) error {
	policyResult, err := evaluatePolicy(ctx, request)
	if err != nil {
		return fmt.Errorf("failed to check if request is allowed: %w", err)
	}

	scope := middlewares.GetScope(ctx)
	auditLogger := ioc.GetDependency[AuditLogger](scope)
	err = auditLogger.Log(ctx, request, policyResult)
	if err != nil {
		return fmt.Errorf("failed to log request: %w", err)
	}

	if !policyResult.allowed {
		return fmt.Errorf("request not allowed: %w", utils.ErrHttpUnauthorized)
	}

	return next()
}

func evaluatePolicy(ctx context.Context, request Policy) (PolicyResult, error) {
	currentUser := authentication.GetCurrentUser(ctx)
	isSystemUser := currentUser.HasPermission(permissions.SystemUser)
	if isSystemUser.IsSuccess() {
		return Allowed(
			currentUser.UserId,
			NewAllowedByPermission(permissions.SystemUser, isSystemUser.SourceRoles),
		), nil
	}

	return request.IsAllowed(ctx)
}
