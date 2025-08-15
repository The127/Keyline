package repositories

import (
	"Keyline/database"
	"Keyline/ioc"
	"Keyline/logging"
	"Keyline/middlewares"
	"context"
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
	"github.com/huandu/go-sqlbuilder"
	"time"
)

type OutboxMessageType string

const (
	DummyOutboxMessageType OutboxMessageType = "dummy"
)

type OutboxMessageDetails interface {
	OutboxMessageType() OutboxMessageType
}

type DummyOutboxMessageDetails struct {
	Foo string
}

func (d *DummyOutboxMessageDetails) OutboxMessageType() OutboxMessageType {
	return DummyOutboxMessageType
}

func (d *DummyOutboxMessageDetails) Value() (driver.Value, error) {
	return json.Marshal(d)
}

func (d *DummyOutboxMessageDetails) Scan(value any) error {
	bytes, ok := value.([]byte)
	if !ok {
		return fmt.Errorf("type assertion for outbox message failed")
	}

	return json.Unmarshal(bytes, &d)
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
