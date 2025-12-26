package repositories

import (
	"Keyline/utils"
	"context"

	"github.com/google/uuid"
)

type UserRoleAssignment struct {
	BaseModel

	userId  uuid.UUID
	roleId  uuid.UUID
	groupId *uuid.UUID

	userInfo UserRoleAssignmentUserInfo
	roleInfo UserRoleAssignmentRoleInfo
}

type UserRoleAssignmentUserInfo struct {
	Username    string
	DisplayName string
}

type UserRoleAssignmentRoleInfo struct {
	ProjectSlug string
	Name        string
}

func NewUserRoleAssignment(userId uuid.UUID, roleId uuid.UUID, groupId *uuid.UUID) *UserRoleAssignment {
	return &UserRoleAssignment{
		BaseModel: NewBaseModel(),
		userId:    userId,
		roleId:    roleId,
		groupId:   groupId,
	}
}

func (u *UserRoleAssignment) UserId() uuid.UUID {
	return u.userId
}

func (u *UserRoleAssignment) UserInfo() UserRoleAssignmentUserInfo {
	return u.userInfo
}

func (u *UserRoleAssignment) RoleInfo() UserRoleAssignmentRoleInfo {
	return u.roleInfo
}

func (u *UserRoleAssignment) RoleId() uuid.UUID {
	return u.roleId
}

func (u *UserRoleAssignment) GroupId() *uuid.UUID {
	return u.groupId
}

func (u *UserRoleAssignment) GetScanPointers(filter UserRoleAssignmentFilter) []any {
	ptrs := []any{
		&u.id,
		&u.auditCreatedAt,
		&u.auditUpdatedAt,
		&u.version,
		&u.userId,
		&u.roleId,
		&u.groupId,
	}

	if filter.includeUser {
		ptrs = append(ptrs,
			&u.userInfo.Username,
			&u.userInfo.DisplayName,
		)
	}

	if filter.includeRole {
		ptrs = append(ptrs,
			&u.roleInfo.Name,
			&u.roleInfo.ProjectSlug,
		)
	}

	return ptrs
}

type UserRoleAssignmentFilter struct {
	userId      *uuid.UUID
	roleId      *uuid.UUID
	groupId     *uuid.UUID
	includeUser bool
	includeRole bool
}

func NewUserRoleAssignmentFilter() UserRoleAssignmentFilter {
	return UserRoleAssignmentFilter{}
}

func (f UserRoleAssignmentFilter) Clone() UserRoleAssignmentFilter {
	return f
}

func (f UserRoleAssignmentFilter) UserId(userId uuid.UUID) UserRoleAssignmentFilter {
	filter := f.Clone()
	filter.userId = &userId
	return filter
}

func (f UserRoleAssignmentFilter) HasUserId() bool {
	return f.userId != nil
}

func (f UserRoleAssignmentFilter) GetUserId() uuid.UUID {
	return utils.ZeroIfNil(f.userId)
}

func (f UserRoleAssignmentFilter) RoleId(roleId uuid.UUID) UserRoleAssignmentFilter {
	filter := f.Clone()
	filter.roleId = &roleId
	return filter
}

func (f UserRoleAssignmentFilter) HasRoleId() bool {
	return f.roleId != nil
}

func (f UserRoleAssignmentFilter) GetRoleId() uuid.UUID {
	return utils.ZeroIfNil(f.roleId)
}

func (f UserRoleAssignmentFilter) GroupId(groupId uuid.UUID) UserRoleAssignmentFilter {
	filter := f.Clone()
	filter.groupId = &groupId
	return filter
}

func (f UserRoleAssignmentFilter) HasGroupId() bool {
	return f.groupId != nil
}

func (f UserRoleAssignmentFilter) GetGroupId() uuid.UUID {
	return utils.ZeroIfNil(f.groupId)
}

func (f UserRoleAssignmentFilter) IncludeUser() UserRoleAssignmentFilter {
	filter := f.Clone()
	filter.includeUser = true
	return filter
}

func (f UserRoleAssignmentFilter) GetIncludeUser() bool {
	return f.includeUser
}

func (f UserRoleAssignmentFilter) IncludeRole() UserRoleAssignmentFilter {
	filter := f.Clone()
	filter.includeRole = true
	return filter
}

func (f UserRoleAssignmentFilter) GetIncludeRole() bool {
	return f.includeRole
}

//go:generate mockgen -destination=./mocks/userroleassignment_repository.go -package=mocks Keyline/internal/repositories UserRoleAssignmentRepository
type UserRoleAssignmentRepository interface {
	List(ctx context.Context, filter UserRoleAssignmentFilter) ([]*UserRoleAssignment, int, error)
	Insert(userRoleAssignment *UserRoleAssignment)
}
