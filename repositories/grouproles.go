package repositories

import "github.com/google/uuid"

type GroupRole struct {
	ModelBase

	groupId uuid.UUID
	roleId  uuid.UUID
}

func NewGroupRole(groupId uuid.UUID, roleId uuid.UUID) *GroupRole {
	return &GroupRole{
		ModelBase: NewModelBase(),
		groupId:   groupId,
		roleId:    roleId,
	}
}

func (g *GroupRole) GroupId() uuid.UUID {
	return g.groupId
}

func (g *GroupRole) RoleId() uuid.UUID {
	return g.roleId
}

type GroupRoleFilter struct {
	groupId *uuid.UUID
	roleId  *uuid.UUID
}

func NewGroupRoleFilter() GroupRoleFilter {
	return GroupRoleFilter{}
}

func (f GroupRoleFilter) Clone() GroupRoleFilter {
	return f
}

func (f GroupRoleFilter) GroupId(groupId uuid.UUID) GroupRoleFilter {
	filter := f.Clone()
	filter.groupId = &groupId
	return filter
}

func (f GroupRoleFilter) RoleId(roleId uuid.UUID) GroupRoleFilter {
	filter := f.Clone()
	filter.roleId = &roleId
	return filter
}

//go:generate mockgen -destination=./mocks/grouprole_repository.go -package=mocks Keyline/repositories GroupRoleRepository
type GroupRoleRepository interface {
}

type groupRoleRepository struct {
}

func NewGroupRoleRepository() GroupRoleRepository {
	return &groupRoleRepository{}
}
