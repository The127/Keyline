package memory

import (
	"context"
	"github.com/The127/Keyline/internal/change"
	"github.com/The127/Keyline/internal/repositories"
	"sync"

	"github.com/google/uuid"
)

type UserRoleAssignmentRepository struct {
	store        map[uuid.UUID]*repositories.UserRoleAssignment
	usersStore   map[uuid.UUID]*repositories.User
	rolesStore   map[uuid.UUID]*repositories.Role
	projectStore map[uuid.UUID]*repositories.Project
	mu           *sync.RWMutex

	changeTracker *change.Tracker
	entityType    int
}

func NewUserRoleAssignmentRepository(
	store map[uuid.UUID]*repositories.UserRoleAssignment,
	usersStore map[uuid.UUID]*repositories.User,
	rolesStore map[uuid.UUID]*repositories.Role,
	projectStore map[uuid.UUID]*repositories.Project,
	mu *sync.RWMutex,
	changeTracker *change.Tracker,
	entityType int,
) *UserRoleAssignmentRepository {
	return &UserRoleAssignmentRepository{
		store:         store,
		usersStore:    usersStore,
		rolesStore:    rolesStore,
		projectStore:  projectStore,
		mu:            mu,
		changeTracker: changeTracker,
		entityType:    entityType,
	}
}

func (r *UserRoleAssignmentRepository) matches(ura *repositories.UserRoleAssignment, filter *repositories.UserRoleAssignmentFilter) bool {
	if filter.HasUserId() && ura.UserId() != filter.GetUserId() {
		return false
	}
	if filter.HasRoleId() && ura.RoleId() != filter.GetRoleId() {
		return false
	}
	if filter.HasGroupId() {
		gid := filter.GetGroupId()
		if ura.GroupId() == nil || *ura.GroupId() != gid {
			return false
		}
	}
	return true
}

func (r *UserRoleAssignmentRepository) enrich(ura *repositories.UserRoleAssignment, filter *repositories.UserRoleAssignmentFilter) *repositories.UserRoleAssignment {
	var userInfo *repositories.UserRoleAssignmentUserInfo
	var roleInfo *repositories.UserRoleAssignmentRoleInfo

	if filter.GetIncludeUser() {
		if u, ok := r.usersStore[ura.UserId()]; ok {
			userInfo = &repositories.UserRoleAssignmentUserInfo{
				Username:    u.Username(),
				DisplayName: u.DisplayName(),
			}
		}
	}

	if filter.GetIncludeRole() {
		if role, ok := r.rolesStore[ura.RoleId()]; ok {
			projectSlug := ""
			if p, ok := r.projectStore[role.ProjectId()]; ok {
				projectSlug = p.Slug()
			}
			roleInfo = &repositories.UserRoleAssignmentRoleInfo{
				ProjectSlug: projectSlug,
				Name:        role.Name(),
			}
		}
	}

	if userInfo == nil && roleInfo == nil {
		return ura
	}

	return repositories.NewUserRoleAssignmentFromDB(
		ura.BaseModel,
		ura.UserId(),
		ura.RoleId(),
		ura.GroupId(),
		userInfo,
		roleInfo,
	)
}

func (r *UserRoleAssignmentRepository) List(_ context.Context, filter *repositories.UserRoleAssignmentFilter) ([]*repositories.UserRoleAssignment, int, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var items []*repositories.UserRoleAssignment
	for _, ura := range r.store {
		if r.matches(ura, filter) {
			items = append(items, r.enrich(ura, filter))
		}
	}
	return items, len(items), nil
}

func (r *UserRoleAssignmentRepository) Insert(userRoleAssignment *repositories.UserRoleAssignment) {
	r.changeTracker.Add(change.NewEntry(change.Added, r.entityType, userRoleAssignment))
}
