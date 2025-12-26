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
	"github.com/huandu/go-sqlbuilder"
)

type postgresOutboxMessage struct {
	postgresBaseModel
	type_   string
	details []byte
}

func mapOutboxMessage(m *repositories.OutboxMessage) *postgresOutboxMessage {
	return &postgresOutboxMessage{
		postgresBaseModel: mapBase(m.BaseModel),
		type_:             string(m.Type()),
		details:           m.Details(),
	}
}

func (m *postgresOutboxMessage) Map() *repositories.OutboxMessage {
	return repositories.NewOutboxMessageFromDB(
		m.MapBase(),
		repositories.OutboxMessageType(m.type_),
		m.details,
	)
}

func (m *postgresOutboxMessage) scan(row pghelpers.Row) error {
	return row.Scan(
		&m.id,
		&m.auditCreatedAt,
		&m.auditUpdatedAt,
		&m.xmin,
		&m.type_,
		&m.details,
	)
}

type OutboxMessageRepository struct {
	db            *sql.DB
	changeTracker *change.Tracker
	entityType    int
}

func NewOutboxMessageRepository(db *sql.DB, changeTracker change.Tracker, entityType int) repositories.OutboxMessageRepository {
	return &OutboxMessageRepository{
		db:            db,
		changeTracker: &changeTracker,
		entityType:    entityType,
	}
}

func (r *OutboxMessageRepository) selectQuery(filter *repositories.OutboxMessageFilter) *sqlbuilder.SelectBuilder {
	s := sqlbuilder.Select(
		"id",
		"audit_created_at",
		"audit_updated_at",
		"xmin",
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

	query, args := s.Build()
	logging.Logger.Debug("executing sql: ", query)
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("querying db: %w", err)
	}
	defer utils.PanicOnError(rows.Close, "closing rows")

	var outboxMessages []*repositories.OutboxMessage
	for rows.Next() {
		outboxMessage := &postgresOutboxMessage{}
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
		Returning("xmin")

	query, args := s.Build()
	logging.Logger.Debug("executing sql: ", query)
	row := tx.QueryRowContext(ctx, query, args...)

	var xmin uint32
	err := row.Scan(&xmin)
	if err != nil {
		return fmt.Errorf("scanning row: %w", err)
	}

	outboxMessage.SetVersion(xmin)
	return nil
}

func (r *OutboxMessageRepository) Delete(id uuid.UUID) {
	r.changeTracker.Add(change.NewEntry(change.Deleted, r.entityType, id))
}

func (r *OutboxMessageRepository) ExecuteDelete(ctx context.Context, tx *sql.Tx, id uuid.UUID) error {
	s := sqlbuilder.DeleteFrom("outbox_messages")

	s.Where(s.Equal("id", id))

	query, args := s.Build()
	logging.Logger.Debug("executing sql: ", query)
	_, err := tx.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("executing delete: %w", err)
	}

	return nil
}
