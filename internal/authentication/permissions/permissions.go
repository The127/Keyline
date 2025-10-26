package permissions

type Permission string

const (
	SystemUser Permission = "system_user"

	AuditView Permission = "audit:view"

	ApplicationCreate Permission = "application:create"
	ApplicationDelete Permission = "application:delete"
	ApplicationUpdate Permission = "application:update"
	ApplicationView   Permission = "application:view"

	GroupView Permission = "group:view"

	RoleCreate Permission = "role:create"
	RoleAssign Permission = "role:assign"
	RoleView   Permission = "role:view"

	UserCreate        Permission = "user:create"
	UserUpdate        Permission = "user:update"
	UserResetPassword Permission = "user:reset_password"
	UserView          Permission = "user:view"

	UserMetadataUpdate Permission = "user_metadata:update"
	UserMetadataView   Permission = "user_metadata:view"

	AppMetadataUpdateAny Permission = "app_metadata:update:any"

	ServiceUserCreate       Permission = "service_user:create"
	ServiceUserAssociateKey Permission = "service_user:associate_key"

	VirtualServerCreate Permission = "virtual_server:create"
	VirtualServerUpdate Permission = "virtual_server:update"
	VirtualServerView   Permission = "virtual_server:view"

	TemplateView Permission = "template:view"

	ProjectCreate Permission = "project:create"
	ProjectUpdate Permission = "project:update"
	ProjectView   Permission = "project:view"

	ResourceServerCreate Permission = "resource_server:create"
	ResourceServerUpdate Permission = "resource_server:update"
	ResourceServerView   Permission = "resource_server:view"

	ResourceServerScopeCreate Permission = "resource_server_scope:create"
)
