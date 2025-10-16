package repositories

import (
	"Keyline/utils"
	"context"

	"github.com/google/uuid"
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

func (m *OutboxMessage) GetScanPointers() []any {
	return []any{
		&m.id,
		&m.auditCreatedAt,
		&m.auditUpdatedAt,
		&m.version,
		&m._type,
		&m.details,
	}
}

func (m *OutboxMessage) Type() OutboxMessageType {
	return m._type
}

func (m *OutboxMessage) Details() OutboxMessageDetails {
	return m.details
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

func (f OutboxMessageFilter) HasId() bool {
	return f.id != nil
}

func (f OutboxMessageFilter) GetId() uuid.UUID {
	return utils.ZeroIfNil(f.id)
}

//go:generate mockgen -destination=./mocks/outboxmessage_repository.go -package=mocks Keyline/internal/repositories OutboxMessageRepository
type OutboxMessageRepository interface {
	List(ctx context.Context, filter OutboxMessageFilter) ([]*OutboxMessage, error)
	Insert(ctx context.Context, outboxMessage *OutboxMessage) error
	Delete(ctx context.Context, id uuid.UUID) error
}
