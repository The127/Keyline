package outbox

import (
	"context"
	"fmt"

	"github.com/The127/Keyline/internal/middlewares"
	"github.com/The127/Keyline/internal/repositories"
)

type inProcessDeliveryService struct{}

func NewInProcessDeliveryService() DeliveryService {
	return &inProcessDeliveryService{}
}

func (s *inProcessDeliveryService) Deliver(ctx context.Context, message *repositories.OutboxMessage) error {
	scope := middlewares.GetScope(ctx)
	deliveryDequeuer := scope.GetDependency[MessageBroker]()
	err := deliveryDequeuer.Distribute(ctx, message)
	if err != nil {
		return fmt.Errorf("failed to handle message: %w", err)
	}
	return nil
}
