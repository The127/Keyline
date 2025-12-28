package database

import (
	"Keyline/internal/repositories"
	"context"
)

const (
	ApplicationEntityType = iota
	ApplicationUserMetadataEntityType
	AuditLogEntityType
	CredentialEntityType
	FileEntityType
	GroupRoleEntityType
	GroupEntityType
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
	ApplicationUserMetadata() repositories.ApplicationUserMetadataRepository
	AuditLogs() repositories.AuditLogRepository
	Credentials() repositories.CredentialRepository
	Files() repositories.FileRepository
	GroupRoles() repositories.GroupRoleRepository
	Groups() repositories.GroupRepository
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
