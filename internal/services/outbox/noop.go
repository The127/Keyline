package outbox

import (
	"Keyline/internal/repositories"
	"context"
)

type noopDeliveryEnqueuer struct{}

func NewNoopDeliveryEnqueuer() DeliveryEnqueuer {
	return &noopDeliveryEnqueuer{}
}

func (n *noopDeliveryEnqueuer) Enqueue(_ context.Context, _ *repositories.OutboxMessage) error {
	return nil
}
