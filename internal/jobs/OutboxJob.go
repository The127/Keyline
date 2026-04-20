package jobs

import (
	"context"
	"fmt"
	"github.com/The127/Keyline/internal/database"
	"github.com/The127/Keyline/internal/logging"
	"github.com/The127/Keyline/internal/middlewares"
	"github.com/The127/Keyline/internal/repositories"
	"github.com/The127/Keyline/internal/services/outbox"
	"github.com/The127/Keyline/utils"

	"github.com/The127/ioc"
)

func OutboxSendingJob(dp *ioc.DependencyProvider) JobFn {
	return func(ctx context.Context) error {
		scope := dp.NewScope()
		defer utils.PanicOnError(scope.Close, "failed to close scope")
		ctx = middlewares.ContextWithScope(ctx, scope)
		dbContext := ioc.GetDependency[database.Context](scope)

		filter := repositories.NewOutboxMessageFilter()
		outboxMessages, err := dbContext.OutboxMessages().List(ctx, filter)
		if err != nil {
			return fmt.Errorf("failed to list outbox messages: %w", err)
		}

		for _, message := range outboxMessages {
			err = handleMessage(ctx, message, dbContext)
			if err != nil {
				// we don't want to stop the whole job if one message fails
				// failed messages will be retried later
				logging.Logger.Errorf("failed handling message: %v", err)
			}
		}

		return nil
	}
}

func handleMessage(ctx context.Context, message *repositories.OutboxMessage, dbContext database.Context) error {
	scope := middlewares.GetScope(ctx)
	delivery := ioc.GetDependency[outbox.DeliveryService](scope)

	err := delivery.Deliver(ctx, message)
	if err != nil {
		return fmt.Errorf("failed to handle message: %w", err)
	}

	dbContext.OutboxMessages().Delete(message.Id())

	return nil
}
