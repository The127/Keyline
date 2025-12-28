package postgres

import (
	"Keyline/internal/change"
	"Keyline/internal/logging"
	"Keyline/internal/repositories"
	"Keyline/internal/repositories/postgres/pghelpers"
	"Keyline/utils"
	"context"
	"database/sql"
	"fmt"

	"github.com/google/uuid"
	"github.com/lib/pq"

	"github.com/huandu/go-sqlbuilder"
)

type postgresAuditLog struct {
	postgresBaseModel
	virtualServerId uuid.UUID
	userId          *uuid.UUID
	requestType     string
	request         string
	response        *pq.ByteaArray
	allowed         bool
	allowReasonType string
	allowReason     *string
}

func mapAuditLog(auditLog *repositories.AuditLog) *postgresAuditLog {
	return &postgresAuditLog{
		postgresBaseModel: mapBase(auditLog.BaseModel),
	}
}

func (a *postgresAuditLog) Map() *repositories.AuditLog {
	return repositories.NewAuditLogFromDB(
		a.MapBase(),
		a.virtualServerId,
		a.userId,
		a.requestType,
		a.request,
		nil, // TODO
		a.allowed,
		&a.allowReasonType,
		a.allowReason,
	)
}

func (a *postgresAuditLog) scan(row pghelpers.Row, additionalPtrs ...any) error {
	ptrs := []any{
		&a.id,
		&a.auditCreatedAt,
		&a.auditUpdatedAt,
		&a.xmin,
		&a.virtualServerId,
		&a.userId,
		&a.requestType,
		&a.request,
		&a.response,
		&a.allowed,
		&a.allowReasonType,
		&a.allowReason,
	}

	ptrs = append(ptrs, additionalPtrs...)

	return row.Scan(ptrs...)
}

type AuditLogRepository struct {
	db            *sql.DB
	changeTracker *change.Tracker
	entityType    int
}

func NewAuditLogRepository(db *sql.DB, changeTracker *change.Tracker, entityType int) *AuditLogRepository {
	return &AuditLogRepository{
		db:            db,
		changeTracker: changeTracker,
		entityType:    entityType,
	}
}

func (r *AuditLogRepository) selectQuery(filter *repositories.AuditLogFilter) *sqlbuilder.SelectBuilder {
	s := sqlbuilder.Select(
		"id",
		"audit_created_at",
		"audit_updated_at",
		"xmin",
		"virtual_server_id",
		"user_id",
		"request_type",
		"request",
		"response",
		"allowed",
		"allow_reason_type",
		"allow_reason",
	).From("audit_logs")

	if filter.HasVirtualServerId() {
		s.Where(s.Equal("virtual_server_id", filter.GetVirtualServerId()))
	}

	if filter.HasUserId() {
		s.Where(s.Equal("user_id", filter.GetUserId()))
	}

	if filter.HasPagination() {
		filter.GetPagingInfo().Apply(s)
	}

	if filter.HasOrder() {
		filter.GetOrderInfo().Apply(s)
	}

	return s
}

func (r *AuditLogRepository) List(ctx context.Context, filter *repositories.AuditLogFilter) ([]*repositories.AuditLog, int, error) {
	s := r.selectQuery(filter)
	s.SelectMore("count(*) over()")

	query, args := s.Build()
	logging.Logger.Debug("executing sql: ", query)
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("querying db: %w", err)
	}
	defer utils.PanicOnError(rows.Close, "closing rows")

	var auditLogs []*repositories.AuditLog
	var totalCount int
	for rows.Next() {
		auditLog := &postgresAuditLog{}
		err := auditLog.scan(rows, &totalCount)
		if err != nil {
			return nil, 0, fmt.Errorf("scanning row: %w", err)
		}

		auditLogs = append(auditLogs, auditLog.Map())
	}

	return auditLogs, totalCount, nil
}

func (r *AuditLogRepository) Insert(auditLog *repositories.AuditLog) {
	r.changeTracker.Add(change.NewEntry(change.Added, r.entityType, auditLog))
}

func (r *AuditLogRepository) ExecuteInsert(ctx context.Context, tx *sql.Tx, auditLog *repositories.AuditLog) error {
	mapped := mapAuditLog(auditLog)

	s := sqlbuilder.InsertInto("audit_logs").
		Cols("virtual_server_id", "user_id", "request_type", "request", "response", "allowed", "allow_reason_type", "allow_reason").
		Values(
			mapped.virtualServerId,
			mapped.userId,
			mapped.requestType,
			mapped.request,
			mapped.response,
			mapped.allowed,
			mapped.allowReasonType,
			mapped.allowReason,
		).
		Returning("xmin")

	query, args := s.Build()
	logging.Logger.Debug("executing sql: ", query)
	row := tx.QueryRowContext(ctx, query, args...)

	var xmin uint32
	err := row.Scan(&xmin)
	if err != nil {
		return fmt.Errorf("scanning row: %w", err)
	}

	auditLog.SetVersion(xmin)
	return nil
}
