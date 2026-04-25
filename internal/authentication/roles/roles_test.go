package roles

import (
	"testing"

	"github.com/The127/Keyline/internal/authentication/permissions"
	"github.com/stretchr/testify/assert"
)

// system-admin must be a strict superset of admin, with virtual_server:create
// the one and only extra. Encoded as a test so the two enumerated lists in
// roles.go can't drift apart silently when a new permission is added.
func TestSystemAdminIsAdminPlusVirtualServerCreate(t *testing.T) {
	t.Parallel()

	sysAdmin := permissionSet(SystemAdminPermissions)
	admin := permissionSet(VirtualServerAdminPermissions)

	for p := range admin {
		assert.Containsf(t, sysAdmin, p, "system-admin missing permission held by admin: %q", p)
	}

	extras := make(map[permissions.Permission]struct{})
	for p := range sysAdmin {
		if _, ok := admin[p]; !ok {
			extras[p] = struct{}{}
		}
	}
	assert.Equal(t, map[permissions.Permission]struct{}{
		permissions.VirtualServerCreate: {},
	}, extras, "system-admin must add exactly virtual_server:create on top of admin")
}

// permissions.SystemUser is the wildcard sentinel that PolicyBehaviour
// short-circuits on. If it leaks into admin or system-admin, every per-request
// permission check is bypassed for those roles too — almost certainly not the
// intent.
func TestSystemUserSentinelStaysOutOfOperatorRoles(t *testing.T) {
	t.Parallel()

	assert.NotContains(t, SystemAdminPermissions, permissions.SystemUser,
		"SystemUser is the wildcard sentinel; only system-user may hold it")
	assert.NotContains(t, VirtualServerAdminPermissions, permissions.SystemUser,
		"SystemUser is the wildcard sentinel; only system-user may hold it")
}

func TestRolePermissionListsHaveNoDuplicates(t *testing.T) {
	t.Parallel()

	cases := map[string][]permissions.Permission{
		"system-user":  SystemUserPermissions,
		"system-admin": SystemAdminPermissions,
		"admin":        VirtualServerAdminPermissions,
	}
	for name, perms := range cases {
		seen := make(map[permissions.Permission]struct{}, len(perms))
		for _, p := range perms {
			_, dup := seen[p]
			assert.Falsef(t, dup, "role %q has duplicate permission %q", name, p)
			seen[p] = struct{}{}
		}
	}
}

// AllRoles is what AuthenticationMiddleware.assignPermissionsToUser looks
// roles up in. Any Role constant that isn't registered here silently grants
// no permissions — easy to miss when adding a new role.
func TestAllRolesRegistersEveryRoleConstant(t *testing.T) {
	t.Parallel()

	for _, role := range []Role{SystemUser, SystemAdmin, VirtualServerAdmin} {
		_, ok := AllRoles[role]
		assert.Truef(t, ok, "role %q is defined as a constant but not registered in AllRoles", role)
	}
}

func permissionSet(perms []permissions.Permission) map[permissions.Permission]struct{} {
	set := make(map[permissions.Permission]struct{}, len(perms))
	for _, p := range perms {
		set[p] = struct{}{}
	}
	return set
}
