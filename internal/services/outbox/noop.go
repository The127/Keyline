package outbox

import (
	"Keyline/internal/repositories"
	"context"
)

type noopDeliveryService struct{}

func NewNoopDeliveryService() DeliveryService {
	return &noopDeliveryService{}
}

func (n *noopDeliveryService) Deliver(_ context.Context, _ *repositories.OutboxMessage) error {
	return nil
}
