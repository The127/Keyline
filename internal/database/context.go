package database

import (
	"context"
	"github.com/The127/Keyline/internal/repositories"
)

const (
	ApplicationEntityType = iota
	ApplicationKeyEntityType
	ApplicationUserMetadataEntityType
	AuditLogEntityType
	CredentialEntityType
	FileEntityType
	GroupRoleEntityType
	GroupEntityType
	IdentityProviderEntityType
	OutboxMessageEntityType
	PasswordRuleEntityType
	ProjectEntityType
	ResourceServerEntityType
	ResourceServerScopeEntityType
	RoleEntityType
	SessionEntityType
	TemplateEntityType
	UserRoleAssignmentEntityType
	UserEntityType
	VirtualServerEntityType
)

//go:generate mockgen -destination=../mocks/mock_context.go -package=mocks Keyline/internal/database Context
type Context interface {
	Applications() repositories.ApplicationRepository
	ApplicationKeys() repositories.ApplicationKeyRepository
	ApplicationUserMetadata() repositories.ApplicationUserMetadataRepository
	AuditLogs() repositories.AuditLogRepository
	Credentials() repositories.CredentialRepository
	Files() repositories.FileRepository
	GroupRoles() repositories.GroupRoleRepository
	Groups() repositories.GroupRepository
	IdentityProviders() repositories.IdentityProviderRepository
	OutboxMessages() repositories.OutboxMessageRepository
	PasswordRules() repositories.PasswordRuleRepository
	Projects() repositories.ProjectRepository
	ResourceServers() repositories.ResourceServerRepository
	ResourceServerScopes() repositories.ResourceServerScopeRepository
	Roles() repositories.RoleRepository
	Sessions() repositories.SessionRepository
	Templates() repositories.TemplateRepository
	UserRoleAssignments() repositories.UserRoleAssignmentRepository
	Users() repositories.UserRepository
	VirtualServers() repositories.VirtualServerRepository

	SaveChanges(ctx context.Context) error
}
