package repositories

import (
	"Keyline/utils"

	"github.com/google/uuid"
)

type GroupRole struct {
	BaseModel

	groupId uuid.UUID
	roleId  uuid.UUID
}

func NewGroupRole(groupId uuid.UUID, roleId uuid.UUID) *GroupRole {
	return &GroupRole{
		BaseModel: NewModelBase(),
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

func (f GroupRoleFilter) HasGroupId() bool {
	return f.groupId != nil
}

func (f GroupRoleFilter) GetGroupId() uuid.UUID {
	return utils.ZeroIfNil(f.groupId)
}

func (f GroupRoleFilter) RoleId(roleId uuid.UUID) GroupRoleFilter {
	filter := f.Clone()
	filter.roleId = &roleId
	return filter
}

func (f GroupRoleFilter) HasRoleId() bool {
	return f.roleId != nil
}

func (f GroupRoleFilter) GetRoleId() uuid.UUID {
	return utils.ZeroIfNil(f.roleId)
}

//go:generate mockgen -destination=./mocks/grouprole_repository.go -package=mocks Keyline/internal/repositories GroupRoleRepository
type GroupRoleRepository interface {
}
