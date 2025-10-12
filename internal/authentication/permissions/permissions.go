package permissions

type Permission string

const (
	SystemUser        Permission = "system_user"
	ApplicationCreate Permission = "application:create"
)
