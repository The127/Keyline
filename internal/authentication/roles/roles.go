package roles

import (
	"Keyline/internal/authentication/permissions"
)

type Role string

const (
	Admin Role = "admin"
)

var AdminPermissions = []permissions.Permission{
	permissions.ApplicationCreate,
}
