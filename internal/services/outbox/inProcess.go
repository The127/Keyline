package outbox

import (
	"Keyline/internal/repositories"
	"context"
)

type inProcessDeliveryEnqueuer struct{}

func NewInProcessDeliveryEnqueuer() DeliveryEnqueuer {
	return &inProcessDeliveryEnqueuer{}
}

func (s *inProcessDeliveryEnqueuer) Enqueue(ctx context.Context, message *repositories.OutboxMessage) error {
	panic("implement me")
}
