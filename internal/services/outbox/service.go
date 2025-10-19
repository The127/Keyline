package outbox

import (
	"Keyline/internal/repositories"
	"context"
)

//go:generate mockgen -destination=./mocks/mock_deliveryEnqueuer.go -package=mocks . DeliveryEnqueuer
type DeliveryEnqueuer interface {
	Enqueue(ctx context.Context, message *repositories.OutboxMessage) error
}
