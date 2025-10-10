package roles

import "Keyline/authentication/permissions"

type Role string

const (
	Admin Role = "admin"
)

var AdminPermissions = []permissions.Permission{
	permissions.ApplicationCreate,
}
