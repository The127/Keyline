package permissions

type Permission string

const (
	SystemUser Permission = "system_user"

	ApplicationCreate Permission = "application:create"
	ApplicationDelete Permission = "application:delete"
	ApplicationUpdate Permission = "application:update"
	ApplicationView   Permission = "application:view"

	RoleCreate Permission = "role:create"
	RoleAssign Permission = "role:assign"

	UserCreate        Permission = "user:create"
	UserUpdate        Permission = "user:update"
	UserResetPassword Permission = "user:reset_password"

	UserMetadataUpdate Permission = "user_metadata:update"

	AppMetadataUpdateAny Permission = "app_metadata:update:any"

	ServiceUserCreate       Permission = "service_user:create"
	ServiceUserAssociateKey Permission = "service_user:associate_key"

	VirtualServerCreate Permission = "virtual_server:create"
	VirtualServerUpdate Permission = "virtual_server:update"
)
