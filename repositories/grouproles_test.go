package repositories

import (
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestGroupRoleFilter(t *testing.T) {
	// arrange
	f := NewGroupRoleFilter()
	groupId := uuid.New()
	roleId := uuid.New()

	// act
	f = f.GroupId(groupId)
	f = f.RoleId(roleId)

	// assert
	assert.Equal(t, &groupId, f.groupId)
	assert.Equal(t, &roleId, f.roleId)
}
