package roles

import (
	"Keyline/internal/authentication/permissions"
)

type Role string

const (
	SystemUser Role = "system_user"
	Admin      Role = "admin"
)

var SystemUserPermissions = []permissions.Permission{
	permissions.SystemUser,
}

var AdminPermissions = []permissions.Permission{
	permissions.VirtualServerCreate,

	permissions.ApplicationCreate,
	permissions.ApplicationDelete,
	permissions.ApplicationUpdate,

	permissions.RoleCreate,
	permissions.RoleAssign,

	permissions.UserCreate,
	permissions.UserUpdate,

	permissions.AppMetadataUpdateAny,

	permissions.ServiceUserCreate,
	permissions.ServiceUserAssociateKey,
}

var AllRoles = map[Role][]permissions.Permission{
	SystemUser: SystemUserPermissions,
	Admin:      AdminPermissions,
}
