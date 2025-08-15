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
	"time"
)

type OutboxMessageType string

const (
	SendMailOutboxMessageType OutboxMessageType = "send_mail"
)

type OutboxMessageDetails interface {
	OutboxMessageType() OutboxMessageType
}

type OutboxMessage struct {
	id uuid.UUID

	auditCreatedAt time.Time
	auditUpdatedAt time.Time

	_type   OutboxMessageType
	details OutboxMessageDetails
}

func NewOutboxMessage(details OutboxMessageDetails) *OutboxMessage {
	return &OutboxMessage{
		_type:   details.OutboxMessageType(),
		details: details,
	}
}

type OutboxMessageFilter struct {
}

func NewOutboxMessageFilter() OutboxMessageFilter {
	return OutboxMessageFilter{}
}

type OutboxMessageRepository struct {
}

func (r *OutboxMessageRepository) Insert(ctx context.Context, outboxMessage *OutboxMessage) error {
	scope := middlewares.GetScope(ctx)
	dbService := ioc.GetDependency[*database.DbService](scope)

	tx, err := dbService.GetTx()
	if err != nil {
		return fmt.Errorf("failed to open tx: %w", err)
	}

	s := sqlbuilder.InsertInto("outbox_messages").
		Cols("type", "details").
		Values(
			outboxMessage._type,
			outboxMessage.details,
		).Returning("id", "audit_created_at", "audit_updated_at")

	query, args := s.Build()
	logging.Logger.Debug("sql: %s", query)
	row := tx.QueryRow(query, args...)

	err = row.Scan(&outboxMessage.id, &outboxMessage.auditCreatedAt, &outboxMessage.auditUpdatedAt)
	if err != nil {
		return fmt.Errorf("scanning row: %w", err)
	}

	return nil
}
