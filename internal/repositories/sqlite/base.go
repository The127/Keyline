package sqlite

import (
	"fmt"
	"github.com/The127/Keyline/internal/repositories"
	"time"

	"github.com/google/uuid"
)

type sqliteBaseModel struct {
	id             uuid.UUID
	auditCreatedAt time.Time
	auditUpdatedAt time.Time
	version        uint
}

func (b *sqliteBaseModel) MapBase() repositories.BaseModel {
	return repositories.NewBaseModelFromDB(b.id, b.auditCreatedAt, b.auditUpdatedAt, b.version)
}

func mapBase(baseModel repositories.BaseModel) sqliteBaseModel {
	return sqliteBaseModel{
		id:             baseModel.Id(),
		auditCreatedAt: baseModel.AuditCreatedAt(),
		auditUpdatedAt: baseModel.AuditUpdatedAt(),
		version:        versionToUint(baseModel.GetVersion()),
	}
}

func versionToUint(version any) uint {
	switch v := version.(type) {
	case nil:
		return 0
	case uint:
		return v
	case uint32:
		return uint(v)
	case uint64:
		return uint(v)
	case int:
		return uint(v)
	case int64:
		return uint(v)
	default:
		panic(fmt.Sprintf("unsupported version type %T", version))
	}
}
