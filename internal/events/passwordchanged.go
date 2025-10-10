package events

import "github.com/google/uuid"

type PasswordChangedEvent struct {
	UserId uuid.UUID
}
