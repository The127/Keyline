package postgres

import (
	"Keyline/internal/database"
	"Keyline/internal/logging"
	"Keyline/internal/middlewares"
	"Keyline/internal/repositories"
	"Keyline/ioc"
	"Keyline/utils"
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/huandu/go-sqlbuilder"
)

type outboxMessageRepository struct {
}

func NewOutboxMessageRepository() repositories.OutboxMessageRepository {
	return &outboxMessageRepository{}
}

func (r *outboxMessageRepository) selectQuery(filter repositories.OutboxMessageFilter) *sqlbuilder.SelectBuilder {
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

func (r *outboxMessageRepository) List(ctx context.Context, filter repositories.OutboxMessageFilter) ([]*repositories.OutboxMessage, error) {
	scope := middlewares.GetScope(ctx)
	dbService := ioc.GetDependency[database.DbService](scope)

	tx, err := dbService.GetTx()
	if err != nil {
		return nil, fmt.Errorf("failed to open tx: %w", err)
	}

	s := r.selectQuery(filter)

	query, args := s.Build()
	logging.Logger.Debug("executing sql: ", query)
	rows, err := tx.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("querying db: %w", err)
	}
	defer utils.PanicOnError(rows.Close, "closing rows")

	var outboxMessages []*repositories.OutboxMessage
	for rows.Next() {
		outboxMessage := repositories.OutboxMessage{
			ModelBase: repositories.NewModelBase(),
		}
		err = rows.Scan(outboxMessage.GetScanPointers()...)
		if err != nil {
			return nil, fmt.Errorf("scanning row: %w", err)
		}
		outboxMessages = append(outboxMessages, &outboxMessage)
	}

	return outboxMessages, nil
}

func (r *outboxMessageRepository) Insert(ctx context.Context, outboxMessage *repositories.OutboxMessage) error {
	scope := middlewares.GetScope(ctx)
	dbService := ioc.GetDependency[database.DbService](scope)

	tx, err := dbService.GetTx()
	if err != nil {
		return fmt.Errorf("failed to open tx: %w", err)
	}

	s := sqlbuilder.InsertInto("outbox_messages").
		Cols("type", "details").
		Values(
			outboxMessage.Type(),
			outboxMessage.Details(),
		).Returning("id", "audit_created_at", "audit_updated_at", "version")

	query, args := s.Build()
	logging.Logger.Debug("executing sql: ", query)
	row := tx.QueryRowContext(ctx, query, args...)

	err = row.Scan(outboxMessage.InsertPointers()...)
	if err != nil {
		return fmt.Errorf("scanning row: %w", err)
	}

	outboxMessage.ClearChanges()
	return nil
}

func (r *outboxMessageRepository) Delete(ctx context.Context, id uuid.UUID) error {
	scope := middlewares.GetScope(ctx)
	dbService := ioc.GetDependency[database.DbService](scope)

	tx, err := dbService.GetTx()
	if err != nil {
		return fmt.Errorf("failed to open tx: %w", err)
	}

	s := sqlbuilder.DeleteFrom("outbox_messages")

	s.Where(s.Equal("id", id))

	query, args := s.Build()
	logging.Logger.Debug("executing sql: ", query)
	_, err = tx.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("executing delete: %w", err)
	}

	return nil
}
