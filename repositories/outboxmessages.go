package repositories

import (
	"Keyline/database"
	"Keyline/ioc"
	"Keyline/logging"
	"Keyline/middlewares"
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/huandu/go-sqlbuilder"
)

type OutboxMessageType string

const (
	SendMailOutboxMessageType OutboxMessageType = "send_mail"
)

type OutboxMessageDetails interface {
	OutboxMessageType() OutboxMessageType
}

type OutboxMessage struct {
	ModelBase

	_type   OutboxMessageType
	details OutboxMessageDetails
}

func (m *OutboxMessage) getScanPointers() []any {
	return []any{
		&m.id,
		&m.auditCreatedAt,
		&m.auditUpdatedAt,
		&m.version,
		&m._type,
		&m.details,
	}
}

func NewOutboxMessage(details OutboxMessageDetails) *OutboxMessage {
	return &OutboxMessage{
		ModelBase: NewModelBase(),
		_type:     details.OutboxMessageType(),
		details:   details,
	}
}

type OutboxMessageFilter struct {
	id *uuid.UUID
}

func NewOutboxMessageFilter() OutboxMessageFilter {
	return OutboxMessageFilter{}
}

func (f OutboxMessageFilter) Clone() OutboxMessageFilter {
	return f
}

func (f OutboxMessageFilter) Id(id uuid.UUID) OutboxMessageFilter {
	filter := f.Clone()
	filter.id = &id
	return filter
}

//go:generate mockgen -destination=./mocks/outboxmessage_repository.go -package=mocks Keyline/repositories OutboxMessageRepository
type OutboxMessageRepository interface {
	List(ctx context.Context, filter OutboxMessageFilter) ([]*OutboxMessage, error)
	Insert(ctx context.Context, outboxMessage *OutboxMessage) error
	Delete(ctx context.Context, filter OutboxMessageFilter) error
}

type outboxMessageRepository struct {
}

func NewOutboxMessageRepository() OutboxMessageRepository {
	return &outboxMessageRepository{}
}

func (r *outboxMessageRepository) selectQuery(filter OutboxMessageFilter) *sqlbuilder.SelectBuilder {
	s := sqlbuilder.Select(
		"id",
		"audit_created_at",
		"audit_updated_at",
		"version",
		"type",
		"details",
	).From("outbox_messages")

	if filter.id != nil {
		s.Where(s.Equal("id", filter.id))
	}

	return s
}

func (r *outboxMessageRepository) List(ctx context.Context, filter OutboxMessageFilter) ([]*OutboxMessage, error) {
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
	defer rows.Close()

	var outboxMessages []*OutboxMessage
	for rows.Next() {
		outboxMessage := OutboxMessage{
			ModelBase: NewModelBase(),
		}
		err = rows.Scan(outboxMessage.getScanPointers()...)
		if err != nil {
			return nil, fmt.Errorf("scanning row: %w", err)
		}
		outboxMessages = append(outboxMessages, &outboxMessage)
	}

	return outboxMessages, nil
}

func (r *outboxMessageRepository) Insert(ctx context.Context, outboxMessage *OutboxMessage) error {
	scope := middlewares.GetScope(ctx)
	dbService := ioc.GetDependency[database.DbService](scope)

	tx, err := dbService.GetTx()
	if err != nil {
		return fmt.Errorf("failed to open tx: %w", err)
	}

	s := sqlbuilder.InsertInto("outbox_messages").
		Cols("type", "details").
		Values(
			outboxMessage._type,
			outboxMessage.details,
		).Returning("id", "audit_created_at", "audit_updated_at", "version")

	query, args := s.Build()
	logging.Logger.Debug("executing sql: ", query)
	row := tx.QueryRowContext(ctx, query, args...)

	err = row.Scan(&outboxMessage.id, &outboxMessage.auditCreatedAt, &outboxMessage.auditUpdatedAt, &outboxMessage.version)
	if err != nil {
		return fmt.Errorf("scanning row: %w", err)
	}
	if err != nil {
		return fmt.Errorf("scanning row: %w", err)
	}

	outboxMessage.clearChanges()
	return nil
}

func (r *outboxMessageRepository) Delete(ctx context.Context, filter OutboxMessageFilter) error {
	scope := middlewares.GetScope(ctx)
	dbService := ioc.GetDependency[database.DbService](scope)

	tx, err := dbService.GetTx()
	if err != nil {
		return fmt.Errorf("failed to open tx: %w", err)
	}

	s := sqlbuilder.DeleteFrom("outbox_messages")

	if filter.id != nil {
		s.Where(s.Equal("id", filter.id))
	}

	query, args := s.Build()
	logging.Logger.Debug("executing sql: ", query)
	_, err = tx.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("executing delete: %w", err)
	}

	return nil
}
