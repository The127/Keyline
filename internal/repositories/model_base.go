package repositories

import (
	"Keyline/utils"
	"fmt"
	"time"

	"github.com/google/uuid"
)

var ErrVersionMismatch = fmt.Errorf("version mismatch: %w", utils.ErrHttpConflict)

type ModelBase struct {
	id uuid.UUID

	auditCreatedAt time.Time
	auditUpdatedAt time.Time

	version int64

	changes map[string]any
}

// UpdatePointers is an internal function that returns the pointers to the auditUpdatedAt and version fields (in that order).
func (m *ModelBase) UpdatePointers() []any {
	return []any{
		&m.auditUpdatedAt,
		&m.version,
	}
}

func NewModelBase() ModelBase {
	return ModelBase{
		changes: make(map[string]any),
	}
}

// Changes is an internal function that returns the changes made to the model.
// The map is empty if no changes have been made.
func (m *ModelBase) Changes() map[string]any {
	return m.changes
}

// TrackChange is an internal function that needs to be called when a field is changed.
func (m *ModelBase) TrackChange(fieldName string, value any) {
	m.changes[fieldName] = value
}

// ClearChanges is an internal function that needs to be called when a model is inserted or updated.
func (m *ModelBase) ClearChanges() {
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

// Mock is a test helper function that sets the model to a mock state.
func (m *ModelBase) Mock(now time.Time) {
	m.id = uuid.New()
	m.auditCreatedAt = now
	m.auditUpdatedAt = now
	m.version = 0
}
