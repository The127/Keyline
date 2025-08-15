package repositories

import (
	"github.com/google/uuid"
	"time"
)

type ModelBase struct {
	id uuid.UUID

	auditCreatedAt time.Time
	auditUpdatedAt time.Time

	changes map[string]any
}

func NewModelBase() ModelBase {
	return ModelBase{
		changes: make(map[string]any),
	}
}

func (m *ModelBase) TrackChange(fieldName string, value any) {
	m.changes[fieldName] = value
}
