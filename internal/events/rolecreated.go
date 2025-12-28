package events

import (
	"Keyline/internal/repositories"
)

type RoleCreatedEvent struct {
	Role *repositories.Role
}
