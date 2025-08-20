package jobs

import (
	"Keyline/ioc"
	"Keyline/middlewares"
	"Keyline/repositories"
	"Keyline/utils"
	"context"
)

func OutboxSendingJob(dp *ioc.DependencyProvider) JobFn {
	return func(ctx context.Context) error {
		scope := dp.NewScope()
		defer utils.PanicOnError(scope.Close, "failed to close scope")
		ctx = middlewares.ContextWithScope(ctx, scope)

		outboxMessageRepository := ioc.GetDependency[*repositories.OutboxMessageRepository](scope)
		outboxMessageRepository.List(ctx)

		return nil
	}
}
