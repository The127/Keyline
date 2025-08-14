package events

import "github.com/google/uuid"

type UserCreatedEvent struct {
	UserId uuid.UUID
}
