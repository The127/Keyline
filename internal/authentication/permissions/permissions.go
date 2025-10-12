package permissions

type Permission string

const (
	SystemUser Permission = "system_user"

	ApplicationCreate Permission = "application:create"

	RoleCreate Permission = "role:create"
	RoleAssign Permission = "role:assign"

	ServiceUserAssociateKey Permission = "service_user:associate_key"
)
