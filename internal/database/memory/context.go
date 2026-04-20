package memory

import (
	"context"
	"fmt"
	"github.com/The127/Keyline/internal/change"
	db "github.com/The127/Keyline/internal/database"
	"github.com/The127/Keyline/internal/repositories"
	memrepos "github.com/The127/Keyline/internal/repositories/memory"

	"github.com/google/uuid"
)

type Context struct {
	stores        *Stores
	changeTracker *change.Tracker

	applications            *memrepos.ApplicationRepository
	applicationUserMetadata *memrepos.ApplicationUserMetadataRepository
	auditLogs               *memrepos.AuditLogRepository
	credentials             *memrepos.CredentialRepository
	files                   *memrepos.FileRepository
	groupRoles              *memrepos.GroupRoleRepository
	groups                  *memrepos.GroupRepository
	outboxMessages          *memrepos.OutboxMessageRepository
	passwordRules           *memrepos.PasswordRuleRepository
	projects                *memrepos.ProjectRepository
	resourceServers         *memrepos.ResourceServerRepository
	resourceServerScopes    *memrepos.ResourceServerScopeRepository
	roles                   *memrepos.RoleRepository
	sessions                *memrepos.SessionRepository
	templates               *memrepos.TemplateRepository
	userRoleAssignments     *memrepos.UserRoleAssignmentRepository
	users                   *memrepos.UserRepository
	virtualServers          *memrepos.VirtualServerRepository
}

func newContext(stores *Stores) *Context {
	return &Context{
		stores:        stores,
		changeTracker: change.NewTracker(),
	}
}

func (c *Context) Applications() repositories.ApplicationRepository {
	if c.applications == nil {
		c.applications = memrepos.NewApplicationRepository(c.stores.Applications, &c.stores.mu, c.changeTracker, db.ApplicationEntityType)
	}
	return c.applications
}

func (c *Context) ApplicationUserMetadata() repositories.ApplicationUserMetadataRepository {
	if c.applicationUserMetadata == nil {
		c.applicationUserMetadata = memrepos.NewApplicationUserMetadataRepository(c.stores.ApplicationUserMetadata, &c.stores.mu, c.changeTracker, db.ApplicationUserMetadataEntityType)
	}
	return c.applicationUserMetadata
}

func (c *Context) AuditLogs() repositories.AuditLogRepository {
	if c.auditLogs == nil {
		c.auditLogs = memrepos.NewAuditLogRepository(c.stores.AuditLogs, &c.stores.mu, c.changeTracker, db.AuditLogEntityType)
	}
	return c.auditLogs
}

func (c *Context) Credentials() repositories.CredentialRepository {
	if c.credentials == nil {
		c.credentials = memrepos.NewCredentialRepository(c.stores.Credentials, &c.stores.mu, c.changeTracker, db.CredentialEntityType)
	}
	return c.credentials
}

func (c *Context) Files() repositories.FileRepository {
	if c.files == nil {
		c.files = memrepos.NewFileRepository(c.stores.Files, &c.stores.mu, c.changeTracker, db.FileEntityType)
	}
	return c.files
}

func (c *Context) GroupRoles() repositories.GroupRoleRepository {
	if c.groupRoles == nil {
		c.groupRoles = memrepos.NewGroupRoleRepository(c.stores.GroupRoles, &c.stores.mu, c.changeTracker, db.GroupRoleEntityType)
	}
	return c.groupRoles
}

func (c *Context) Groups() repositories.GroupRepository {
	if c.groups == nil {
		c.groups = memrepos.NewGroupRepository(c.stores.Groups, &c.stores.mu, c.changeTracker, db.GroupEntityType)
	}
	return c.groups
}

func (c *Context) OutboxMessages() repositories.OutboxMessageRepository {
	if c.outboxMessages == nil {
		c.outboxMessages = memrepos.NewOutboxMessageRepository(c.stores.OutboxMessages, &c.stores.mu, c.changeTracker, db.OutboxMessageEntityType)
	}
	return c.outboxMessages
}

func (c *Context) PasswordRules() repositories.PasswordRuleRepository {
	if c.passwordRules == nil {
		c.passwordRules = memrepos.NewPasswordRuleRepository(c.stores.PasswordRules, &c.stores.mu, c.changeTracker, db.PasswordRuleEntityType)
	}
	return c.passwordRules
}

func (c *Context) Projects() repositories.ProjectRepository {
	if c.projects == nil {
		c.projects = memrepos.NewProjectRepository(c.stores.Projects, &c.stores.mu, c.changeTracker, db.ProjectEntityType)
	}
	return c.projects
}

func (c *Context) ResourceServers() repositories.ResourceServerRepository {
	if c.resourceServers == nil {
		c.resourceServers = memrepos.NewResourceServerRepository(c.stores.ResourceServers, &c.stores.mu, c.changeTracker, db.ResourceServerEntityType)
	}
	return c.resourceServers
}

func (c *Context) ResourceServerScopes() repositories.ResourceServerScopeRepository {
	if c.resourceServerScopes == nil {
		c.resourceServerScopes = memrepos.NewResourceServerScopeRepository(c.stores.ResourceServerScopes, &c.stores.mu, c.changeTracker, db.ResourceServerScopeEntityType)
	}
	return c.resourceServerScopes
}

func (c *Context) Roles() repositories.RoleRepository {
	if c.roles == nil {
		c.roles = memrepos.NewRoleRepository(c.stores.Roles, &c.stores.mu, c.changeTracker, db.RoleEntityType)
	}
	return c.roles
}

func (c *Context) Sessions() repositories.SessionRepository {
	if c.sessions == nil {
		c.sessions = memrepos.NewSessionRepository(c.stores.Sessions, &c.stores.mu, c.changeTracker, db.SessionEntityType)
	}
	return c.sessions
}

func (c *Context) Templates() repositories.TemplateRepository {
	if c.templates == nil {
		c.templates = memrepos.NewTemplateRepository(c.stores.Templates, &c.stores.mu, c.changeTracker, db.TemplateEntityType)
	}
	return c.templates
}

func (c *Context) UserRoleAssignments() repositories.UserRoleAssignmentRepository {
	if c.userRoleAssignments == nil {
		c.userRoleAssignments = memrepos.NewUserRoleAssignmentRepository(
			c.stores.UserRoleAssignments,
			c.stores.Users,
			c.stores.Roles,
			c.stores.Projects,
			&c.stores.mu,
			c.changeTracker,
			db.UserRoleAssignmentEntityType,
		)
	}
	return c.userRoleAssignments
}

func (c *Context) Users() repositories.UserRepository {
	if c.users == nil {
		c.users = memrepos.NewUserRepository(c.stores.Users, &c.stores.mu, c.changeTracker, db.UserEntityType)
	}
	return c.users
}

func (c *Context) VirtualServers() repositories.VirtualServerRepository {
	if c.virtualServers == nil {
		c.virtualServers = memrepos.NewVirtualServerRepository(c.stores.VirtualServers, &c.stores.mu, c.changeTracker, db.VirtualServerEntityType)
	}
	return c.virtualServers
}

func (c *Context) SaveChanges(_ context.Context) error {
	c.stores.mu.Lock()
	defer c.stores.mu.Unlock()

	for _, ch := range c.changeTracker.GetChanges() {
		if err := c.applyChange(ch); err != nil {
			return fmt.Errorf("applying change: %w", err)
		}
	}

	c.changeTracker.Clear()
	return nil
}

func (c *Context) applyChange(ch *change.Entry) error {
	switch ch.GetItemType() {
	case db.ApplicationEntityType:
		return applyChange(c.stores.Applications, ch, func(e *repositories.Application) { e.SetVersion(incrementVersion(e.GetVersion())); e.ClearChanges() })

	case db.ApplicationUserMetadataEntityType:
		return applyChange(c.stores.ApplicationUserMetadata, ch, func(e *repositories.ApplicationUserMetadata) {
			e.SetVersion(incrementVersion(e.GetVersion()))
			e.ClearChanges()
		})

	case db.AuditLogEntityType:
		return applyInsertOnly(c.stores.AuditLogs, ch, func(e *repositories.AuditLog) { e.SetVersion(1) })

	case db.CredentialEntityType:
		return applyChange(c.stores.Credentials, ch, func(e *repositories.Credential) { e.SetVersion(incrementVersion(e.GetVersion())); e.ClearChanges() })

	case db.FileEntityType:
		return applyInsertOnly(c.stores.Files, ch, func(e *repositories.File) { e.SetVersion(1) })

	case db.GroupRoleEntityType:
		return fmt.Errorf("unsupported change type for group role: %v", ch.GetChangeType())

	case db.GroupEntityType:
		return applyChange(c.stores.Groups, ch, func(e *repositories.Group) { e.SetVersion(incrementVersion(e.GetVersion())); e.ClearChanges() })

	case db.OutboxMessageEntityType:
		return applyOutboxMessageChange(c.stores.OutboxMessages, ch)

	case db.PasswordRuleEntityType:
		return applyChange(c.stores.PasswordRules, ch, func(e *repositories.PasswordRule) { e.SetVersion(incrementVersion(e.GetVersion())); e.ClearChanges() })

	case db.ProjectEntityType:
		return applyChange(c.stores.Projects, ch, func(e *repositories.Project) { e.SetVersion(incrementVersion(e.GetVersion())); e.ClearChanges() })

	case db.ResourceServerEntityType:
		return applyChange(c.stores.ResourceServers, ch, func(e *repositories.ResourceServer) { e.SetVersion(incrementVersion(e.GetVersion())); e.ClearChanges() })

	case db.ResourceServerScopeEntityType:
		return applyChange(c.stores.ResourceServerScopes, ch, func(e *repositories.ResourceServerScope) {
			e.SetVersion(incrementVersion(e.GetVersion()))
			e.ClearChanges()
		})

	case db.RoleEntityType:
		return applyChange(c.stores.Roles, ch, func(e *repositories.Role) { e.SetVersion(incrementVersion(e.GetVersion())); e.ClearChanges() })

	case db.SessionEntityType:
		return applySessionChange(c.stores.Sessions, ch)

	case db.TemplateEntityType:
		return applyInsertOnly(c.stores.Templates, ch, func(e *repositories.Template) { e.SetVersion(1) })

	case db.UserRoleAssignmentEntityType:
		return applyInsertOnly(c.stores.UserRoleAssignments, ch, func(e *repositories.UserRoleAssignment) { e.SetVersion(1) })

	case db.UserEntityType:
		return applyChange(c.stores.Users, ch, func(e *repositories.User) { e.SetVersion(incrementVersion(e.GetVersion())); e.ClearChanges() })

	case db.VirtualServerEntityType:
		return applyChange(c.stores.VirtualServers, ch, func(e *repositories.VirtualServer) { e.SetVersion(incrementVersion(e.GetVersion())); e.ClearChanges() })

	default:
		return fmt.Errorf("unsupported item type: %v", ch.GetItemType())
	}
}

func incrementVersion(v any) int {
	if v == nil {
		return 1
	}
	if i, ok := v.(int); ok {
		return i + 1
	}
	return 1
}

type entityWithId interface {
	Id() uuid.UUID
}

func applyChange[T entityWithId](store map[uuid.UUID]T, ch *change.Entry, onWrite func(T)) error {
	switch ch.GetChangeType() {
	case change.Added:
		entity := ch.GetItem().(T)
		onWrite(entity)
		store[entity.Id()] = entity
		return nil

	case change.Updated:
		entity := ch.GetItem().(T)
		onWrite(entity)
		store[entity.Id()] = entity
		return nil

	case change.Deleted:
		id := ch.GetItem().(uuid.UUID)
		delete(store, id)
		return nil

	default:
		return fmt.Errorf("unsupported change type: %v", ch.GetChangeType())
	}
}

func applyInsertOnly[T entityWithId](store map[uuid.UUID]T, ch *change.Entry, onInsert func(T)) error {
	switch ch.GetChangeType() {
	case change.Added:
		entity := ch.GetItem().(T)
		onInsert(entity)
		store[entity.Id()] = entity
		return nil

	default:
		return fmt.Errorf("unsupported change type: %v", ch.GetChangeType())
	}
}

func applyOutboxMessageChange(store map[uuid.UUID]*repositories.OutboxMessage, ch *change.Entry) error {
	switch ch.GetChangeType() {
	case change.Added:
		entity := ch.GetItem().(*repositories.OutboxMessage)
		entity.SetVersion(1)
		store[entity.Id()] = entity
		return nil

	case change.Deleted:
		id := ch.GetItem().(uuid.UUID)
		delete(store, id)
		return nil

	default:
		return fmt.Errorf("unsupported change type: %v", ch.GetChangeType())
	}
}

func applySessionChange(store map[uuid.UUID]*repositories.Session, ch *change.Entry) error {
	switch ch.GetChangeType() {
	case change.Added:
		entity := ch.GetItem().(*repositories.Session)
		entity.SetVersion(1)
		store[entity.Id()] = entity
		return nil

	case change.Deleted:
		id := ch.GetItem().(uuid.UUID)
		delete(store, id)
		return nil

	default:
		return fmt.Errorf("unsupported change type: %v", ch.GetChangeType())
	}
}
