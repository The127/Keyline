package repositories

import (
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestUserRoleAssignmentFilter(t *testing.T) {
	// arrange
	f := NewUserRoleAssignmentFilter()
	userId := uuid.New()
	roleId := uuid.New()
	groupId := uuid.New()

	// act
	f = f.UserId(userId)
	f = f.RoleId(roleId)
	f = f.GroupId(groupId)

	// assert
	assert.Equal(t, &userId, f.userId)
	assert.Equal(t, &roleId, f.roleId)
	assert.Equal(t, &groupId, f.groupId)
}
