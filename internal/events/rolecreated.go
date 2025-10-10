package events

import "github.com/google/uuid"

type RoleCreatedEvent struct {
	RoleId uuid.UUID
}
