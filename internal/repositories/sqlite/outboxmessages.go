package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/The127/Keyline/internal/change"
	"github.com/The127/Keyline/internal/logging"
	"github.com/The127/Keyline/internal/repositories"
	"github.com/The127/Keyline/internal/repositories/sqlite/sqlitehelpers"
	"github.com/The127/Keyline/utils"

	"github.com/google/uuid"
	"github.com/huandu/go-sqlbuilder"
)

type sqliteOutboxMessage struct {
	sqliteBaseModel
	type_   string
	details []byte
}

func mapOutboxMessage(m *repositories.OutboxMessage) *sqliteOutboxMessage {
	return &sqliteOutboxMessage{
		sqliteBaseModel: mapBase(m.BaseModel),
		type_:           string(m.Type()),
		details:         m.Details(),
	}
}

func (m *sqliteOutboxMessage) Map() *repositories.OutboxMessage {
	return repositories.NewOutboxMessageFromDB(
		m.MapBase(),
		repositories.OutboxMessageType(m.type_),
		m.details,
	)
}

func (m *sqliteOutboxMessage) scan(row sqlitehelpers.Row, additionalPtrs ...any) error {
	ptrs := []any{
		&m.id,
		&m.auditCreatedAt,
		&m.auditUpdatedAt,
		&m.version,
		&m.type_,
		&m.details,
	}

	ptrs = append(ptrs, additionalPtrs...)

	return row.Scan(ptrs...)
}

type OutboxMessageRepository struct {
	db            *sql.DB
	changeTracker *change.Tracker
	entityType    int
}

func NewOutboxMessageRepository(db *sql.DB, changeTracker *change.Tracker, entityType int) *OutboxMessageRepository {
	return &OutboxMessageRepository{
		db:            db,
		changeTracker: changeTracker,
		entityType:    entityType,
	}
}

func (r *OutboxMessageRepository) selectQuery(filter *repositories.OutboxMessageFilter) *sqlbuilder.SelectBuilder {
	s := sqlbuilder.Select(
		"id",
		"audit_created_at",
		"audit_updated_at",
		"version",
		"type",
		"details",
	).From("outbox_messages")

	if filter.HasId() {
		s.Where(s.Equal("id", filter.GetId()))
	}

	return s
}

func (r *OutboxMessageRepository) List(ctx context.Context, filter *repositories.OutboxMessageFilter) ([]*repositories.OutboxMessage, error) {
	s := r.selectQuery(filter)

	query, args := s.BuildWithFlavor(sqlbuilder.SQLite)
	logging.Logger.Debug("executing sql: ", query)
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("querying db: %w", err)
	}
	defer utils.PanicOnError(rows.Close, "closing rows")

	var outboxMessages []*repositories.OutboxMessage
	for rows.Next() {
		outboxMessage := &sqliteOutboxMessage{}
		err := outboxMessage.scan(rows)
		if err != nil {
			return nil, fmt.Errorf("scanning row: %w", err)
		}
		outboxMessages = append(outboxMessages, outboxMessage.Map())
	}

	return outboxMessages, nil
}

func (r *OutboxMessageRepository) Insert(outboxMessage *repositories.OutboxMessage) {
	r.changeTracker.Add(change.NewEntry(change.Added, r.entityType, outboxMessage))
}

func (r *OutboxMessageRepository) ExecuteInsert(ctx context.Context, tx *sql.Tx, outboxMessage *repositories.OutboxMessage) error {
	mapped := mapOutboxMessage(outboxMessage)

	s := sqlbuilder.InsertInto("outbox_messages").
		Cols(
			"id",
			"audit_created_at",
			"audit_updated_at",
			"type",
			"details",
		).
		Values(
			mapped.id,
			mapped.auditCreatedAt,
			mapped.auditUpdatedAt,
			mapped.type_,
			mapped.details,
		).
		Returning("version")

	query, args := s.BuildWithFlavor(sqlbuilder.SQLite)
	logging.Logger.Debug("executing sql: ", query)
	row := tx.QueryRowContext(ctx, query, args...)

	var version uint32
	err := row.Scan(&version)
	if err != nil {
		return fmt.Errorf("scanning row: %w", err)
	}

	outboxMessage.SetVersion(version)
	return nil
}

func (r *OutboxMessageRepository) Delete(id uuid.UUID) {
	r.changeTracker.Add(change.NewEntry(change.Deleted, r.entityType, id))
}

func (r *OutboxMessageRepository) ExecuteDelete(ctx context.Context, tx *sql.Tx, id uuid.UUID) error {
	s := sqlbuilder.DeleteFrom("outbox_messages")

	s.Where(s.Equal("id", id))

	query, args := s.BuildWithFlavor(sqlbuilder.SQLite)
	logging.Logger.Debug("executing sql: ", query)
	_, err := tx.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("executing delete: %w", err)
	}

	return nil
}
