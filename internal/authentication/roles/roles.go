package roles

import (
	"Keyline/internal/authentication/permissions"
)

type Role string

const (
	SystemUser         Role = "system_user"
	SystemAdmin        Role = "system_admin"
	VirtualServerAdmin Role = "admin"
)

var SystemUserPermissions = []permissions.Permission{
	permissions.SystemUser,
}

var SystemAdminPermissions = []permissions.Permission{
	permissions.VirtualServerCreate,
}

var VirtualServerAdminPermissions = []permissions.Permission{
	permissions.VirtualServerUpdate,
	permissions.VirtualServerView,

	permissions.ProjectCreate,
	permissions.ProjectUpdate,
	permissions.ProjectView,

	permissions.ResourceServerCreate,
	permissions.ResourceServerUpdate,
	permissions.ResourceServerView,

	permissions.ResourceServerScopeCreate,
	permissions.ResourceServerScopeUpdate,
	permissions.ResourceServerScopeView,

	permissions.AuditView,

	permissions.ApplicationCreate,
	permissions.ApplicationDelete,
	permissions.ApplicationUpdate,
	permissions.ApplicationView,

	permissions.GroupView,

	permissions.RoleCreate,
	permissions.RoleUpdate,
	permissions.RoleAssign,
	permissions.RoleView,

	permissions.UserCreate,
	permissions.UserUpdate,
	permissions.UserResetPassword,
	permissions.UserView,

	permissions.UserMetadataUpdate,
	permissions.UserMetadataView,

	permissions.AppMetadataUpdateAny,

	permissions.ServiceUserCreate,
	permissions.ServiceUserAssociateKey,
	permissions.ServiceUserRemoveKey,

	permissions.TemplateView,
}

var AllRoles = map[Role][]permissions.Permission{
	SystemUser:         SystemUserPermissions,
	SystemAdmin:        SystemAdminPermissions,
	VirtualServerAdmin: VirtualServerAdminPermissions,
}
