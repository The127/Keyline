package repositories

import (
	"Keyline/utils"
	"fmt"
	"github.com/google/uuid"
	"time"
)

var ErrVersionMismatch = fmt.Errorf("version mismatch: %w", utils.ErrHttpConflict)

type ModelBase struct {
	id uuid.UUID

	auditCreatedAt time.Time
	auditUpdatedAt time.Time

	version int64

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

func (m *ModelBase) clearChanges() {
	m.changes = make(map[string]any)
}

func (m *ModelBase) Id() uuid.UUID {
	return m.id
}

func (m *ModelBase) AuditCreatedAt() time.Time {
	return m.auditCreatedAt
}

func (m *ModelBase) AuditUpdatedAt() time.Time {
	return m.auditUpdatedAt
}

func (m *ModelBase) Version() int64 {
	return m.version
}
