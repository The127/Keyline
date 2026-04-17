package events

import (
	"github.com/The127/Keyline/internal/repositories"
)

type RoleCreatedEvent struct {
	Role *repositories.Role
}
