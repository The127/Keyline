package outbox

import (
	"context"
	"github.com/The127/Keyline/internal/repositories"
)

type noopDeliveryService struct{}

func NewNoopDeliveryService() DeliveryService {
	return &noopDeliveryService{}
}

func (n *noopDeliveryService) Deliver(_ context.Context, _ *repositories.OutboxMessage) error {
	return nil
}
