package postgres

import (
	"Keyline/internal/database"
	"Keyline/internal/logging"
	"Keyline/internal/middlewares"
	"Keyline/internal/repositories"
	"Keyline/utils"
	"context"
	"fmt"
	"github.com/The127/ioc"

	"github.com/huandu/go-sqlbuilder"
)

type auditLogRepository struct{}

func NewAuditLogRepository() repositories.AuditLogRepository {
	return &auditLogRepository{}
}

func (r *auditLogRepository) selectQuery(filter repositories.AuditLogFilter) *sqlbuilder.SelectBuilder {
	s := sqlbuilder.Select(
		"id",
		"audit_created_at",
		"audit_updated_at",
		"version",
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

func (r *auditLogRepository) List(ctx context.Context, filter repositories.AuditLogFilter) ([]*repositories.AuditLog, int, error) {
	scope := middlewares.GetScope(ctx)
	dbService := ioc.GetDependency[database.DbService](scope)

	tx, err := dbService.GetTx()
	if err != nil {
		return nil, 0, fmt.Errorf("failed to open tx: %w", err)
	}

	s := r.selectQuery(filter)
	s.SelectMore("count(*) over()")

	query, args := s.Build()
	logging.Logger.Debug("executing sql: ", query)
	rows, err := tx.Query(query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("querying db: %w", err)
	}
	defer utils.PanicOnError(rows.Close, "closing rows")

	var auditLogs []*repositories.AuditLog
	var totalCount int
	for rows.Next() {
		auditLog := repositories.AuditLog{
			BaseModel: repositories.NewModelBase(),
		}
		err = rows.Scan(append(auditLog.GetScanPointers(), &totalCount)...)
		if err != nil {
			return nil, 0, fmt.Errorf("scanning row: %w", err)
		}

		auditLogs = append(auditLogs, &auditLog)
	}

	return auditLogs, totalCount, nil
}

func (r *auditLogRepository) Insert(ctx context.Context, auditLog *repositories.AuditLog) error {
	scope := middlewares.GetScope(ctx)
	dbService := ioc.GetDependency[database.DbService](scope)

	tx, err := dbService.GetTx()
	if err != nil {
		return fmt.Errorf("failed to open tx: %w", err)
	}

	s := sqlbuilder.InsertInto("audit_logs").
		Cols("virtual_server_id", "user_id", "request_type", "request", "response", "allowed", "allow_reason_type", "allow_reason").
		Values(
			auditLog.VirtualServerId(),
			auditLog.UserId(),
			auditLog.RequestType(),
			auditLog.Request(),
			auditLog.Response(),
			auditLog.Allowed(),
			auditLog.AllowReasonType(),
			auditLog.AllowReason(),
		).Returning("id", "audit_created_at", "audit_updated_at", "version")

	query, args := s.Build()
	logging.Logger.Debug("executing sql: ", query)
	row := tx.QueryRowContext(ctx, query, args...)

	err = row.Scan(auditLog.InsertPointers()...)
	if err != nil {
		return fmt.Errorf("scanning row: %w", err)
	}

	auditLog.ClearChanges()
	return nil
}
