package permissions

type Permission string

const (
	SystemUser Permission = "system_user"

	ApplicationCreate Permission = "application:create"
	ApplicationDelete Permission = "application:delete"

	RoleCreate Permission = "role:create"
	RoleAssign Permission = "role:assign"

	UserCreate Permission = "user:create"

	ServiceUserCreate       Permission = "service_user:create"
	ServiceUserAssociateKey Permission = "service_user:associate_key"

	VirtualServerCreate Permission = "virtual_server:create"
)
