package repositories

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
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
