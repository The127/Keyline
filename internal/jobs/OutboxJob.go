package jobs

import (
	"Keyline/internal/middlewares"
	"Keyline/internal/repositories"
	"Keyline/ioc"
	"Keyline/logging"
	"Keyline/utils"
	"context"
	"fmt"
)

func OutboxSendingJob(dp *ioc.DependencyProvider) JobFn {
	return func(ctx context.Context) error {
		scope := dp.NewScope()
		defer utils.PanicOnError(scope.Close, "failed to close scope")
		ctx = middlewares.ContextWithScope(ctx, scope)

		outboxMessageRepository := ioc.GetDependency[repositories.OutboxMessageRepository](scope)
		filter := repositories.NewOutboxMessageFilter()
		outboxMessages, err := outboxMessageRepository.List(ctx, filter)
		if err != nil {
			return fmt.Errorf("failed to list outbox messages: %w", err)
		}

		for _, message := range outboxMessages {
			err = handleMessage(ctx, message, outboxMessageRepository)
			if err != nil {
				logging.Logger.Errorf("failed handling message: %v", err)
			}
		}

		return nil
	}
}

func handleMessage(ctx context.Context, message *repositories.OutboxMessage, repository repositories.OutboxMessageRepository) error {
	// TODO: send to rabbitmq

	err := repository.Delete(ctx, message.Id())
	if err != nil {
		return fmt.Errorf("failed to delete message in database: %w", err)
	}

	return nil
}
