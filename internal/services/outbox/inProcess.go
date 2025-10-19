package outbox

import (
	"Keyline/internal/middlewares"
	"Keyline/internal/repositories"
	"Keyline/ioc"
	"context"
	"fmt"
)

type inProcessDeliveryService struct{}

func NewInProcessDeliveryService() DeliveryService {
	return &inProcessDeliveryService{}
}

func (s *inProcessDeliveryService) Deliver(ctx context.Context, message *repositories.OutboxMessage) error {
	scope := middlewares.GetScope(ctx)
	deliveryDequeuer := ioc.GetDependency[MessageBroker](scope)
	err := deliveryDequeuer.Distribute(ctx, message)
	if err != nil {
		return fmt.Errorf("failed to handle message: %w", err)
	}
	return nil
}
