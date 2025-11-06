package behaviours

import (
	"Keyline/internal/authentication"
	"Keyline/internal/authentication/permissions"
	"Keyline/internal/authentication/roles"
	"Keyline/internal/middlewares"
	"Keyline/internal/repositories"
	"Keyline/ioc"
	"Keyline/utils"
	"context"
	"fmt"

	"github.com/The127/mediatr"

	"github.com/google/uuid"
)

type AuditLogger interface {
	Log(ctx context.Context, policy Policy, policyResult PolicyResult, response any) error
}

type PolicyResult struct {
	allowed         bool
	userId          uuid.UUID
	virtualServerId uuid.UUID
	reason          AllowReason
}

func (p PolicyResult) IsAllowed() bool {
	return p.allowed
}

func (p PolicyResult) UserId() uuid.UUID {
	return p.userId
}

func (p PolicyResult) VirtualServerId() uuid.UUID {
	return p.virtualServerId
}

func (p PolicyResult) Reason() AllowReason {
	return p.reason
}

type AllowReason interface {
	ImplementsAllowReason()
	GetReasonType() string
	IsServiceUserReason() bool
}

type AllowedByAnyone struct{}

func NewAllowedByAnyone() AllowedByAnyone {
	return AllowedByAnyone{}
}

func (a AllowedByAnyone) IsServiceUserReason() bool {
	return false
}

func (a AllowedByAnyone) GetReasonType() string {
	return "anyone"
}

func (a AllowedByAnyone) String() string {
	return "Anyone"
}

func (a AllowedByAnyone) ImplementsAllowReason() {}

type AllowedByOwnership struct{}

func NewAllowedByOwnership() AllowedByOwnership {
	return AllowedByOwnership{}
}

func (a AllowedByOwnership) IsServiceUserReason() bool {
	return false
}

func (a AllowedByOwnership) GetReasonType() string {
	return "ownership"
}

func (a AllowedByOwnership) String() string {
	return "Ownership"
}

func (a AllowedByOwnership) ImplementsAllowReason() {}

type AllowedByPermission struct {
	Permission  permissions.Permission
	SourceRoles []roles.Role
}

func (a AllowedByPermission) IsServiceUserReason() bool {
	return a.Permission == permissions.SystemUser
}

func NewAllowedByPermission(permission permissions.Permission, sourceRoles []roles.Role) AllowedByPermission {
	return AllowedByPermission{
		Permission:  permission,
		SourceRoles: sourceRoles,
	}
}

func (a AllowedByPermission) GetReasonType() string {
	return "permission"
}

func (a AllowedByPermission) String() string {
	return fmt.Sprintf("Permission: %s, SourceRoles: %v", a.Permission, a.SourceRoles)
}

func (a AllowedByPermission) ImplementsAllowReason() {}

func Allowed(userId uuid.UUID, virtualServerId uuid.UUID, reason AllowReason) PolicyResult {
	return PolicyResult{
		allowed:         true,
		userId:          userId,
		virtualServerId: virtualServerId,
		reason:          reason,
	}
}

func Denied(userId uuid.UUID, virtualServerId uuid.UUID) PolicyResult {
	return PolicyResult{
		allowed:         false,
		userId:          userId,
		virtualServerId: virtualServerId,
	}
}

type Policy interface {
	IsAllowed(ctx context.Context) (PolicyResult, error)
	GetRequestName() string
	LogResponse() bool
	LogRequest() bool
}

func PolicyBehaviour(ctx context.Context, request Policy, next mediatr.Next) (any, error) {
	policyResult, err := evaluatePolicy(ctx, request)
	if err != nil {
		return nil, fmt.Errorf("failed to check if request is allowed: %w", err)
	}

	if !policyResult.allowed {
		return nil, fmt.Errorf("request not allowed: %w", utils.ErrHttpUnauthorized)
	}

	response, err := next()

	// don't log if there was an error and only log if the request says so
	if err == nil && request.LogRequest() {
		scope := middlewares.GetScope(ctx)
		auditLogger := ioc.GetDependency[AuditLogger](scope)

		var logResponse any = nil
		if request.LogResponse() {
			logResponse = response
		}

		err = auditLogger.Log(ctx, request, policyResult, logResponse)
		if err != nil {
			return nil, fmt.Errorf("failed to log request: %w", err)
		}
	}

	return response, err
}

func evaluatePolicy(ctx context.Context, request Policy) (PolicyResult, error) {
	currentUser := authentication.GetCurrentUser(ctx)
	isSystemUser := currentUser.HasPermission(permissions.SystemUser)

	virtualServerName, err := middlewares.GetVirtualServerName(ctx)
	if err != nil {
		virtualServerName = ""
	}
	virtualServerId := uuid.Nil

	// if virtual server name is not set, we don't need to check for virtual server
	// this should only happen for internal bootstrap requests where there is no virtual server yet
	// and we don't want to fail on that
	// this can only be the case for the system user
	// cron jobs and other internal requests should ensure that the virtual server is set in the context
	if virtualServerName != "" {
		scope := middlewares.GetScope(ctx)

		virtualServerRepository := ioc.GetDependency[repositories.VirtualServerRepository](scope)
		virtualServerFilter := repositories.NewVirtualServerFilter().Name(virtualServerName)
		virtualServer, err := virtualServerRepository.Single(ctx, virtualServerFilter)
		if err != nil {
			return PolicyResult{}, fmt.Errorf("getting virtual server: %w", err)
		}

		virtualServerId = virtualServer.Id()
	}

	if isSystemUser.IsSuccess() {
		return Allowed(
			currentUser.UserId,
			virtualServerId,
			NewAllowedByPermission(
				permissions.SystemUser,
				isSystemUser.SourceRoles,
			),
		), nil
	}

	return request.IsAllowed(ctx)
}

func PermissionBasedPolicy(ctx context.Context, permission permissions.Permission) (PolicyResult, error) {
	virtualServerName, err := middlewares.GetVirtualServerName(ctx)
	if err != nil {
		virtualServerName = ""
	}

	virtualServerId := uuid.Nil

	if virtualServerName != "" {
		scope := middlewares.GetScope(ctx)

		virtualServerRepository := ioc.GetDependency[repositories.VirtualServerRepository](scope)
		virtualServerFilter := repositories.NewVirtualServerFilter().Name(virtualServerName)
		virtualServer, err := virtualServerRepository.First(ctx, virtualServerFilter)
		if err != nil {
			return PolicyResult{}, fmt.Errorf("getting virtual server: %w", err)
		}

		virtualServerId = virtualServer.Id()
	}

	currentUser := authentication.GetCurrentUser(ctx)
	if !currentUser.IsAuthenticated() {
		return Denied(currentUser.UserId, virtualServerId), nil
	}

	hasPermission := currentUser.HasPermission(permission)
	if !hasPermission.IsSuccess() {
		return Denied(currentUser.UserId, virtualServerId), nil
	}

	return Allowed(
		currentUser.UserId,
		virtualServerId,
		NewAllowedByPermission(
			permission,
			hasPermission.SourceRoles,
		),
	), nil
}
