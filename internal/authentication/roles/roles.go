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
	permissions.ApplicationCreate,
}

var AllRoles = map[Role][]permissions.Permission{
	Admin: AdminPermissions,
}
