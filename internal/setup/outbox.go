package setup

import (
	"github.com/The127/Keyline/config"
	"github.com/The127/Keyline/internal/services/outbox"

	"github.com/The127/ioc"
)

func OutboxDelivery(dc *ioc.DependencyCollection, queueMode config.QueueMode) {
	switch queueMode {
	case config.QueueModeNoop:
		ioc.RegisterSingleton(dc, func(_ *ioc.DependencyProvider) outbox.DeliveryService {
			return outbox.NewNoopDeliveryService()
		})

	case config.QueueModeInProcess:
		ioc.RegisterSingleton(dc, func(_ *ioc.DependencyProvider) outbox.DeliveryService {
			return outbox.NewInProcessDeliveryService()
		})

	default:
		panic("queue mode missing or not supported")
	}

	ioc.RegisterTransient(dc, func(dp *ioc.DependencyProvider) outbox.MessageBroker {
		return outbox.NewMessageBroker()
	})
}
