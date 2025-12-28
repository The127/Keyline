package postgres

import (
	"Keyline/internal/repositories"
	"time"

	"github.com/google/uuid"
)

type postgresBaseModel struct {
	id             uuid.UUID
	auditCreatedAt time.Time
	auditUpdatedAt time.Time
	xmin           uint
}

func (b *postgresBaseModel) MapBase() repositories.BaseModel {
	return repositories.NewBaseModelFromDB(b.id, b.auditCreatedAt, b.auditUpdatedAt, b.xmin)
}

func mapBase(baseModel repositories.BaseModel) postgresBaseModel {
	var xmin uint = 0
	if baseModel.GetVersion() != nil {
		xmin = baseModel.GetVersion().(uint)
	}

	return postgresBaseModel{
		id:             baseModel.Id(),
		auditCreatedAt: baseModel.AuditCreatedAt(),
		auditUpdatedAt: baseModel.AuditUpdatedAt(),
		xmin:           xmin,
	}
}
