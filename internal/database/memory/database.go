package memory

import (
	db "github.com/The127/Keyline/internal/database"
	"github.com/The127/Keyline/internal/repositories"
	"context"
	"sync"

	"github.com/google/uuid"
)

// Stores holds all in-memory maps shared across context instances.
type Stores struct {
	mu sync.RWMutex

	Applications            map[uuid.UUID]*repositories.Application
	ApplicationUserMetadata map[uuid.UUID]*repositories.ApplicationUserMetadata
	AuditLogs               map[uuid.UUID]*repositories.AuditLog
	Credentials             map[uuid.UUID]*repositories.Credential
	Files                   map[uuid.UUID]*repositories.File
	GroupRoles              map[uuid.UUID]*repositories.GroupRole
	Groups                  map[uuid.UUID]*repositories.Group
	OutboxMessages          map[uuid.UUID]*repositories.OutboxMessage
	PasswordRules           map[uuid.UUID]*repositories.PasswordRule
	Projects                map[uuid.UUID]*repositories.Project
	ResourceServers         map[uuid.UUID]*repositories.ResourceServer
	ResourceServerScopes    map[uuid.UUID]*repositories.ResourceServerScope
	Roles                   map[uuid.UUID]*repositories.Role
	Sessions                map[uuid.UUID]*repositories.Session
	Templates               map[uuid.UUID]*repositories.Template
	UserRoleAssignments     map[uuid.UUID]*repositories.UserRoleAssignment
	Users                   map[uuid.UUID]*repositories.User
	VirtualServers          map[uuid.UUID]*repositories.VirtualServer
}

func newStores() *Stores {
	return &Stores{
		Applications:            make(map[uuid.UUID]*repositories.Application),
		ApplicationUserMetadata: make(map[uuid.UUID]*repositories.ApplicationUserMetadata),
		AuditLogs:               make(map[uuid.UUID]*repositories.AuditLog),
		Credentials:             make(map[uuid.UUID]*repositories.Credential),
		Files:                   make(map[uuid.UUID]*repositories.File),
		GroupRoles:              make(map[uuid.UUID]*repositories.GroupRole),
		Groups:                  make(map[uuid.UUID]*repositories.Group),
		OutboxMessages:          make(map[uuid.UUID]*repositories.OutboxMessage),
		PasswordRules:           make(map[uuid.UUID]*repositories.PasswordRule),
		Projects:                make(map[uuid.UUID]*repositories.Project),
		ResourceServers:         make(map[uuid.UUID]*repositories.ResourceServer),
		ResourceServerScopes:    make(map[uuid.UUID]*repositories.ResourceServerScope),
		Roles:                   make(map[uuid.UUID]*repositories.Role),
		Sessions:                make(map[uuid.UUID]*repositories.Session),
		Templates:               make(map[uuid.UUID]*repositories.Template),
		UserRoleAssignments:     make(map[uuid.UUID]*repositories.UserRoleAssignment),
		Users:                   make(map[uuid.UUID]*repositories.User),
		VirtualServers:          make(map[uuid.UUID]*repositories.VirtualServer),
	}
}

type memoryDatabase struct {
	stores *Stores
}

func NewMemoryDatabase() db.Database {
	return &memoryDatabase{
		stores: newStores(),
	}
}

func (d *memoryDatabase) Migrate(_ context.Context) error {
	return nil
}

func (d *memoryDatabase) NewDbContext(_ context.Context) (db.Context, error) {
	return newContext(d.stores), nil
}

func (d *memoryDatabase) Close() error {
	return nil
}
