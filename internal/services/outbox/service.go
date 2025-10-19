package outbox

import (
	"Keyline/internal/logging"
	"Keyline/internal/repositories"
	"context"
)

//go:generate mockgen -destination=./mocks/mock_deliveryService.go -package=mocks . DeliveryService
type DeliveryService interface {
	Deliver(ctx context.Context, message *repositories.OutboxMessage) error
}

//go:generate mockgen -destination=./mocks/mock_messageBroker.go -package=mocks . MessageBroker
type MessageBroker interface {
	Distribute(ctx context.Context, message *repositories.OutboxMessage) error
}

type messageBroker struct {
}

func NewMessageBroker() MessageBroker {
	return &messageBroker{}
}

func (m *messageBroker) Distribute(ctx context.Context, message *repositories.OutboxMessage) error {
	logging.Logger.Info("Distributing message", "message", message)
	return nil
}
