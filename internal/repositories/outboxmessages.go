package repositories

import (
	"Keyline/utils"
	"context"
	"fmt"

	"github.com/google/uuid"
)

type OutboxMessageType string

const (
	SendMailOutboxMessageType OutboxMessageType = "send_mail"
)

type OutboxMessageDetails interface {
	OutboxMessageType() OutboxMessageType
	Serialize() ([]byte, error)
}

type OutboxMessage struct {
	BaseModel

	_type   OutboxMessageType
	details []byte
}

func (m *OutboxMessage) Type() OutboxMessageType {
	return m._type
}

func (m *OutboxMessage) Details() []byte {
	return m.details
}

func NewOutboxMessage(details OutboxMessageDetails) (*OutboxMessage, error) {
	serializedDetails, err := details.Serialize()
	if err != nil {
		return nil, fmt.Errorf("failed to serialize details: %w", err)
	}

	return &OutboxMessage{
		BaseModel: NewBaseModel(),
		_type:     details.OutboxMessageType(),
		details:   serializedDetails,
	}, nil
}

func NewOutboxMessageFromDB(base BaseModel, _type OutboxMessageType, details []byte) *OutboxMessage {
	return &OutboxMessage{
		BaseModel: base,
		_type:     _type,
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
	Insert(outboxMessage *OutboxMessage)
	Delete(id uuid.UUID)
}
