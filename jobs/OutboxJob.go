package jobs

import (
	"Keyline/ioc"
	"Keyline/logging"
	"context"
	"time"
)

func OutboxSendingJob(dp *ioc.DependencyProvider) JobFn {
	return func(ctx context.Context) error {
		logging.Logger.Infof("ayooo")
		time.Sleep(time.Second * 5)
		return nil
	}
}
