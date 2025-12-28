package postgres

import (
	"Keyline/internal/change"
	db "Keyline/internal/database"
	"Keyline/internal/repositories"
	"Keyline/internal/repositories/postgres"
	"context"
	"database/sql"
	"fmt"

	"github.com/google/uuid"
)

type Context struct {
	db            *sql.DB
	changeTracker *change.Tracker

	applications            *postgres.ApplicationRepository
	applicationUserMetadata *postgres.ApplicationUserMetadataRepository
	auditLogs               *postgres.AuditLogRepository
	credentials             *postgres.CredentialRepository
	files                   *postgres.FileRepository
	groupRoles              *postgres.GroupRoleRepository
	groups                  *postgres.GroupRepository
	outboxMessages          *postgres.OutboxMessageRepository
	passwordRules           *postgres.PasswordRuleRepository
	projects                *postgres.ProjectRepository
	resourceServers         *postgres.ResourceServerRepository
	resourceServerScopes    *postgres.ResourceServerScopeRepository
	roles                   *postgres.RoleRepository
	sessions                *postgres.SessionRepository
	templates               *postgres.TemplateRepository
	userRoleAssignments     *postgres.UserRoleAssignmentRepository
	users                   *postgres.UserRepository
	virtualServers          *postgres.VirtualServerRepository
}

func (c *Context) Applications() repositories.ApplicationRepository {
	if c.applications == nil {
		c.applications = postgres.NewApplicationRepository(c.db, c.changeTracker, db.ApplicationEntityType)
	}

	return c.applications
}

func (c *Context) ApplicationUserMetadata() repositories.ApplicationUserMetadataRepository {
	if c.applicationUserMetadata == nil {
		c.applicationUserMetadata = postgres.NewApplicationUserMetadataRepository(c.db, c.changeTracker, db.ApplicationUserMetadataEntityType)
	}

	return c.applicationUserMetadata
}

func (c *Context) AuditLogs() repositories.AuditLogRepository {
	if c.auditLogs == nil {
		c.auditLogs = postgres.NewAuditLogRepository(c.db, c.changeTracker, db.AuditLogEntityType)
	}

	return c.auditLogs
}

func (c *Context) Credentials() repositories.CredentialRepository {
	if c.credentials == nil {
		c.credentials = postgres.NewCredentialRepository(c.db, c.changeTracker, db.CredentialEntityType)
	}

	return c.credentials
}

func (c *Context) Files() repositories.FileRepository {
	if c.files == nil {
		c.files = postgres.NewFileRepository(c.db, c.changeTracker, db.FileEntityType)
	}

	return c.files
}

func (c *Context) GroupRoles() repositories.GroupRoleRepository {
	if c.groupRoles == nil {
		c.groupRoles = postgres.NewGroupRoleRepository(c.db, c.changeTracker, db.GroupRoleEntityType)
	}

	return c.groupRoles
}

func (c *Context) Groups() repositories.GroupRepository {
	if c.groups == nil {
		c.groups = postgres.NewGroupRepository(c.db, c.changeTracker, db.GroupEntityType)
	}

	return c.groups
}

func (c *Context) OutboxMessages() repositories.OutboxMessageRepository {
	if c.outboxMessages == nil {
		c.outboxMessages = postgres.NewOutboxMessageRepository(c.db, c.changeTracker, db.OutboxMessageEntityType)
	}

	return c.outboxMessages
}

func (c *Context) PasswordRules() repositories.PasswordRuleRepository {
	if c.passwordRules == nil {
		c.passwordRules = postgres.NewPasswordRuleRepository(c.db, c.changeTracker, db.PasswordRuleEntityType)
	}

	return c.passwordRules
}

func (c *Context) Projects() repositories.ProjectRepository {
	if c.projects == nil {
		c.projects = postgres.NewProjectRepository(c.db, c.changeTracker, db.ProjectEntityType)
	}

	return c.projects
}

func (c *Context) ResourceServers() repositories.ResourceServerRepository {
	if c.resourceServers == nil {
		c.resourceServers = postgres.NewResourceServerRepository(c.db, c.changeTracker, db.ResourceServerEntityType)
	}

	return c.resourceServers
}

func (c *Context) ResourceServerScopes() repositories.ResourceServerScopeRepository {
	if c.resourceServerScopes == nil {
		c.resourceServerScopes = postgres.NewResourceServerScopeRepository(c.db, c.changeTracker, db.ResourceServerScopeEntityType)
	}

	return c.resourceServerScopes
}

func (c *Context) Roles() repositories.RoleRepository {
	if c.roles == nil {
		c.roles = postgres.NewRoleRepository(c.db, c.changeTracker, db.RoleEntityType)
	}

	return c.roles
}

func (c *Context) Sessions() repositories.SessionRepository {
	if c.sessions == nil {
		c.sessions = postgres.NewSessionRepository(c.db, c.changeTracker, db.SessionEntityType)
	}

	return c.sessions
}

func (c *Context) Templates() repositories.TemplateRepository {
	if c.templates == nil {
		c.templates = postgres.NewTemplateRepository(c.db, c.changeTracker, db.TemplateEntityType)
	}

	return c.templates
}

func (c *Context) UserRoleAssignments() repositories.UserRoleAssignmentRepository {
	if c.userRoleAssignments == nil {
		c.userRoleAssignments = postgres.NewUserRoleAssignmentRepository(c.db, c.changeTracker, db.UserRoleAssignmentEntityType)
	}

	return c.userRoleAssignments
}

func (c *Context) Users() repositories.UserRepository {
	if c.users == nil {
		c.users = postgres.NewUserRepository(c.db, c.changeTracker, db.UserEntityType)
	}

	return c.users
}

func (c *Context) VirtualServers() repositories.VirtualServerRepository {
	if c.virtualServers == nil {
		c.virtualServers = postgres.NewVirtualServerRepository(c.db, c.changeTracker, db.VirtualServerEntityType)
	}

	return c.virtualServers
}

func newContext(db *sql.DB) *Context {
	return &Context{
		db:            db,
		changeTracker: change.NewTracker(),
	}
}

func (c *Context) SaveChanges(ctx context.Context) error {
	tx, err := c.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("starting transaction: %w", err)
	}

	for _, ch := range c.changeTracker.GetChanges() {
		err := c.applyChange(ctx, tx, ch)
		if err != nil {
			return fmt.Errorf("applying change: %w", err)
		}
	}

	err = tx.Commit()
	if err != nil {
		return fmt.Errorf("committing transaction: %w", err)
	}

	c.changeTracker.Clear()
	return nil
}

func (c *Context) applyChange(ctx context.Context, tx *sql.Tx, ch *change.Entry) error {
	switch ch.GetItemType() {
	case db.ApplicationEntityType:
		return c.applyApplicationChange(ctx, tx, ch)

	case db.ApplicationUserMetadataEntityType:
		return c.applyApplicationUserMetadataChange(ctx, tx, ch)

	case db.AuditLogEntityType:
		return c.applyAuditLogChange(ctx, tx, ch)

	case db.CredentialEntityType:
		return c.applyCredentialChange(ctx, tx, ch)

	case db.FileEntityType:
		return c.applyFileChange(ctx, tx, ch)

	case db.GroupRoleEntityType:
		return c.applyGroupRoleChange(ctx, tx, ch)

	case db.GroupEntityType:
		return c.applyGroupChange(ctx, tx, ch)

	case db.OutboxMessageEntityType:
		return c.applyOutboxMessageChange(ctx, tx, ch)

	case db.PasswordRuleEntityType:
		return c.applyPasswordRuleChange(ctx, tx, ch)

	case db.ProjectEntityType:
		return c.applyProjectChange(ctx, tx, ch)

	case db.ResourceServerEntityType:
		return c.applyResourceServerChange(ctx, tx, ch)

	case db.ResourceServerScopeEntityType:
		return c.applyResourceServerScopeChange(ctx, tx, ch)

	case db.RoleEntityType:
		return c.applyRoleChange(ctx, tx, ch)

	case db.SessionEntityType:
		return c.applySessionChange(ctx, tx, ch)

	case db.TemplateEntityType:
		return c.applyTemplateChange(ctx, tx, ch)

	case db.UserRoleAssignmentEntityType:
		return c.applyUserRoleAssignmentChange(ctx, tx, ch)

	case db.UserEntityType:
		return c.applyUserChange(ctx, tx, ch)

	case db.VirtualServerEntityType:
		return c.applyVirtualServerChange(ctx, tx, ch)

	default:
		return fmt.Errorf("unsupported item type: %v", ch.GetItemType())
	}
}

func (c *Context) applyApplicationChange(ctx context.Context, tx *sql.Tx, ch *change.Entry) error {
	switch ch.GetChangeType() {
	case change.Added:
		return c.applications.ExecuteInsert(ctx, tx, ch.GetItem().(*repositories.Application))

	case change.Updated:
		return c.applications.ExecuteUpdate(ctx, tx, ch.GetItem().(*repositories.Application))

	case change.Deleted:
		return c.applications.ExecuteDelete(ctx, tx, ch.GetItem().(uuid.UUID))

	default:
		return fmt.Errorf("unsupported change type: %v", ch.GetChangeType())
	}
}

func (c *Context) applyApplicationUserMetadataChange(ctx context.Context, tx *sql.Tx, ch *change.Entry) error {
	switch ch.GetChangeType() {
	case change.Added:
		return c.applicationUserMetadata.ExecuteInsert(ctx, tx, ch.GetItem().(*repositories.ApplicationUserMetadata))

	case change.Updated:
		return c.applicationUserMetadata.ExecuteUpdate(ctx, tx, ch.GetItem().(*repositories.ApplicationUserMetadata))

	default:
		return fmt.Errorf("unsupported change type: %v", ch.GetChangeType())
	}
}

func (c *Context) applyAuditLogChange(ctx context.Context, tx *sql.Tx, ch *change.Entry) error {
	switch ch.GetChangeType() {
	case change.Added:
		return c.auditLogs.ExecuteInsert(ctx, tx, ch.GetItem().(*repositories.AuditLog))

	default:
		return fmt.Errorf("unsupported change type: %v", ch.GetChangeType())
	}
}

func (c *Context) applyCredentialChange(ctx context.Context, tx *sql.Tx, ch *change.Entry) error {
	switch ch.GetChangeType() {
	case change.Added:
		return c.credentials.ExecuteInsert(ctx, tx, ch.GetItem().(*repositories.Credential))

	case change.Updated:
		return c.credentials.ExecuteUpdate(ctx, tx, ch.GetItem().(*repositories.Credential))

	case change.Deleted:
		return c.credentials.ExecuteDelete(ctx, tx, ch.GetItem().(uuid.UUID))

	default:
		return fmt.Errorf("unsupported change type: %v", ch.GetChangeType())
	}
}

func (c *Context) applyFileChange(ctx context.Context, tx *sql.Tx, ch *change.Entry) error {
	switch ch.GetChangeType() {
	case change.Added:
		return c.files.ExecuteInsert(ctx, tx, ch.GetItem().(*repositories.File))

	default:
		return fmt.Errorf("unsupported change type: %v", ch.GetChangeType())
	}
}

func (c *Context) applyGroupRoleChange(_ context.Context, _ *sql.Tx, ch *change.Entry) error {
	return fmt.Errorf("unsupported change type: %v", ch.GetChangeType())
}

func (c *Context) applyGroupChange(ctx context.Context, tx *sql.Tx, ch *change.Entry) error {
	switch ch.GetChangeType() {
	case change.Added:
		return c.groups.ExecuteInsert(ctx, tx, ch.GetItem().(*repositories.Group))

	case change.Updated:
		return c.groups.ExecuteUpdate(ctx, tx, ch.GetItem().(*repositories.Group))

	case change.Deleted:
		return c.groups.ExecuteDelete(ctx, tx, ch.GetItem().(uuid.UUID))

	default:
		return fmt.Errorf("unsupported change type: %v", ch.GetChangeType())
	}
}

func (c *Context) applyOutboxMessageChange(ctx context.Context, tx *sql.Tx, ch *change.Entry) error {
	switch ch.GetChangeType() {
	case change.Added:
		return c.outboxMessages.ExecuteInsert(ctx, tx, ch.GetItem().(*repositories.OutboxMessage))

	case change.Deleted:
		return c.outboxMessages.ExecuteDelete(ctx, tx, ch.GetItem().(uuid.UUID))

	default:
		return fmt.Errorf("unsupported change type: %v", ch.GetChangeType())
	}
}

func (c *Context) applyPasswordRuleChange(ctx context.Context, tx *sql.Tx, ch *change.Entry) error {
	switch ch.GetChangeType() {
	case change.Added:
		return c.passwordRules.ExecuteInsert(ctx, tx, ch.GetItem().(*repositories.PasswordRule))

	case change.Updated:
		return c.passwordRules.ExecuteUpdate(ctx, tx, ch.GetItem().(*repositories.PasswordRule))

	case change.Deleted:
		return c.passwordRules.ExecuteDelete(ctx, tx, ch.GetItem().(uuid.UUID))

	default:
		return fmt.Errorf("unsupported change type: %v", ch.GetChangeType())
	}
}

func (c *Context) applyProjectChange(ctx context.Context, tx *sql.Tx, ch *change.Entry) error {
	switch ch.GetChangeType() {
	case change.Added:
		return c.projects.ExecuteInsert(ctx, tx, ch.GetItem().(*repositories.Project))

	case change.Updated:
		return c.projects.ExecuteUpdate(ctx, tx, ch.GetItem().(*repositories.Project))

	case change.Deleted:
		return c.projects.ExecuteDelete(ctx, tx, ch.GetItem().(uuid.UUID))

	default:
		return fmt.Errorf("unsupported change type: %v", ch.GetChangeType())
	}
}

func (c *Context) applyResourceServerChange(ctx context.Context, tx *sql.Tx, ch *change.Entry) error {
	switch ch.GetChangeType() {
	case change.Added:
		return c.resourceServers.ExecuteInsert(ctx, tx, ch.GetItem().(*repositories.ResourceServer))

	case change.Updated:
		return c.resourceServers.ExecuteUpdate(ctx, tx, ch.GetItem().(*repositories.ResourceServer))

	case change.Deleted:
		return c.resourceServers.ExecuteDelete(ctx, tx, ch.GetItem().(uuid.UUID))

	default:
		return fmt.Errorf("unsupported change type: %v", ch.GetChangeType())
	}
}

func (c *Context) applyResourceServerScopeChange(ctx context.Context, tx *sql.Tx, ch *change.Entry) error {
	switch ch.GetChangeType() {
	case change.Added:
		return c.resourceServerScopes.ExecuteInsert(ctx, tx, ch.GetItem().(*repositories.ResourceServerScope))

	case change.Updated:
		return c.resourceServerScopes.ExecuteUpdate(ctx, tx, ch.GetItem().(*repositories.ResourceServerScope))

	case change.Deleted:
		return c.resourceServerScopes.ExecuteDelete(ctx, tx, ch.GetItem().(uuid.UUID))

	default:
		return fmt.Errorf("unsupported change type: %v", ch.GetChangeType())
	}
}

func (c *Context) applyRoleChange(ctx context.Context, tx *sql.Tx, ch *change.Entry) error {
	switch ch.GetChangeType() {
	case change.Added:
		return c.roles.ExecuteInsert(ctx, tx, ch.GetItem().(*repositories.Role))

	case change.Updated:
		return c.roles.ExecuteUpdate(ctx, tx, ch.GetItem().(*repositories.Role))

	case change.Deleted:
		return c.roles.ExecuteDelete(ctx, tx, ch.GetItem().(uuid.UUID))

	default:
		return fmt.Errorf("unsupported change type: %v", ch.GetChangeType())
	}
}

func (c *Context) applySessionChange(ctx context.Context, tx *sql.Tx, ch *change.Entry) error {
	switch ch.GetChangeType() {
	case change.Added:
		return c.sessions.ExecuteInsert(ctx, tx, ch.GetItem().(*repositories.Session))

	case change.Deleted:
		return c.sessions.ExecuteDelete(ctx, tx, ch.GetItem().(uuid.UUID))

	default:
		return fmt.Errorf("unsupported change type: %v", ch.GetChangeType())
	}
}

func (c *Context) applyTemplateChange(ctx context.Context, tx *sql.Tx, ch *change.Entry) error {
	switch ch.GetChangeType() {
	case change.Added:
		return c.templates.ExecuteInsert(ctx, tx, ch.GetItem().(*repositories.Template))

	default:
		return fmt.Errorf("unsupported change type: %v", ch.GetChangeType())
	}
}

func (c *Context) applyUserRoleAssignmentChange(ctx context.Context, tx *sql.Tx, ch *change.Entry) error {
	switch ch.GetChangeType() {
	case change.Added:
		return c.userRoleAssignments.ExecuteInsert(ctx, tx, ch.GetItem().(*repositories.UserRoleAssignment))

	default:
		return fmt.Errorf("unsupported change type: %v", ch.GetChangeType())
	}
}

func (c *Context) applyUserChange(ctx context.Context, tx *sql.Tx, ch *change.Entry) error {
	switch ch.GetChangeType() {
	case change.Added:
		return c.users.ExecuteInsert(ctx, tx, ch.GetItem().(*repositories.User))

	case change.Updated:
		return c.users.ExecuteUpdate(ctx, tx, ch.GetItem().(*repositories.User))

	default:
		return fmt.Errorf("unsupported change type: %v", ch.GetChangeType())
	}
}

func (c *Context) applyVirtualServerChange(ctx context.Context, tx *sql.Tx, ch *change.Entry) error {
	switch ch.GetChangeType() {
	case change.Added:
		return c.virtualServers.ExecuteInsert(ctx, tx, ch.GetItem().(*repositories.VirtualServer))

	case change.Updated:
		return c.virtualServers.ExecuteUpdate(ctx, tx, ch.GetItem().(*repositories.VirtualServer))

	default:
		return fmt.Errorf("unsupported change type: %v", ch.GetChangeType())
	}
}
