package repositories

import (
	"Keyline/utils"
	"fmt"
	"time"

	"github.com/google/uuid"
)

var ErrVersionMismatch = fmt.Errorf("version mismatch: %w", utils.ErrHttpConflict)

type BaseModel struct {
	id uuid.UUID

	auditCreatedAt time.Time
	auditUpdatedAt time.Time

	version any
}

func NewBaseModel() BaseModel {
	return BaseModel{
		version: nil,
	}
}

func NewBaseModelFromDB(id uuid.UUID, auditCreatedAt time.Time, auditUpdatedAt time.Time, version any) BaseModel {
	return BaseModel{
		id:             id,
		auditCreatedAt: auditCreatedAt,
		auditUpdatedAt: auditUpdatedAt,
		version:        version,
	}
}

func (m *BaseModel) Id() uuid.UUID {
	return m.id
}

func (m *BaseModel) AuditCreatedAt() time.Time {
	return m.auditCreatedAt
}

func (m *BaseModel) AuditUpdatedAt() time.Time {
	return m.auditUpdatedAt
}

func (b *BaseModel) GetVersion() any {
	return b.version
}

// SetVersion is used to update the version of the model.
// This is used to prevent concurrent updates.
// This function should only be called by the repositories.
func (b *BaseModel) SetVersion(version any) {
	b.version = version
}

// Mock is a test helper function that sets the model to a mock state.
func (m *BaseModel) Mock(now time.Time) {
	m.id = uuid.New()
	m.auditCreatedAt = now
	m.auditUpdatedAt = now
	m.version = 0
}
