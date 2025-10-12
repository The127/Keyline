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
	permissions.VirtualServerUpdate,

	permissions.ApplicationCreate,
	permissions.ApplicationDelete,
	permissions.ApplicationUpdate,
	permissions.ApplicationView,

	permissions.RoleCreate,
	permissions.RoleAssign,
	permissions.RoleView,

	permissions.UserCreate,
	permissions.UserUpdate,
	permissions.UserResetPassword,

	permissions.UserMetadataUpdate,

	permissions.AppMetadataUpdateAny,

	permissions.ServiceUserCreate,
	permissions.ServiceUserAssociateKey,

	permissions.TemplateView,
}

var AllRoles = map[Role][]permissions.Permission{
	SystemUser: SystemUserPermissions,
	Admin:      AdminPermissions,
}
