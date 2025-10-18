package jobs

import (
	"Keyline/internal/clock"
	"Keyline/internal/config"
	"Keyline/internal/middlewares"
	"Keyline/internal/repositories"
	"Keyline/internal/services"
	"Keyline/ioc"
	"Keyline/utils"
	"context"
	"fmt"
	"time"
)

func KeyRotateJob() JobFn {
	return func(ctx context.Context) error {
		scope := middlewares.GetScope(ctx).NewScope()
		defer utils.PanicOnError(scope.Close, "failed to close scope")

		virtualServerRepository := ioc.GetDependency[repositories.VirtualServerRepository](scope)
		virtualServerFilter := repositories.NewVirtualServerFilter()
		virtualServers, _, err := virtualServerRepository.List(ctx, virtualServerFilter)
		if err != nil {
			return fmt.Errorf("listing virtual servers: %w", err)
		}

		for _, virtualServer := range virtualServers {
			err = rotateKeysForVirtualServer(scope, virtualServer)
			if err != nil {
				// TODO: we don't want to stop the whole job if one virtual server fails
				return fmt.Errorf("rotating keys for virtual server %s: %w", virtualServer.Name(), err)
			}
		}

		return nil
	}
}

func rotateKeysForVirtualServer(dp *ioc.DependencyProvider, server *repositories.VirtualServer) error {
	keyStore := ioc.GetDependency[services.KeyStore](dp)
	keyPairs, err := keyStore.GetAll(server.Name())
	if err != nil {
		return fmt.Errorf("getting key pairs: %w", err)
	}

	clockService := ioc.GetDependency[clock.Service](dp)

	err = deleteExpiredKeys(keyPairs, keyStore, server.Name(), clockService.Now())
	if err != nil {
		return fmt.Errorf("deleting expired key: %w", err)
	}

	keyService := ioc.GetDependency[services.KeyService](dp)
	err = generateNewKeys(keyPairs, keyService, server, clockService)
	if err != nil {
		return fmt.Errorf("generating new key: %w", err)
	}

	return nil
}

func deleteExpiredKeys(
	keyPairs []services.KeyPair,
	keyStore services.KeyStore,
	virtualServerName string,
	now time.Time,
) error {
	for _, keyPair := range keyPairs {
		if keyPair.ExpiresAt().Before(now) {
			err := keyStore.Remove(virtualServerName, keyPair.Algorithm(), keyPair.GetKid())
			if err != nil {
				return fmt.Errorf("removing key pair: %w", err)
			}
			continue
		}
	}
	return nil
}

func generateNewKeys(
	keyPairs []services.KeyPair,
	keyService services.KeyService,
	server *repositories.VirtualServer,
	clockService clock.Service,
) error {
	algorithmsToRotate := make(map[config.SigningAlgorithm]bool)
	for _, keyPair := range keyPairs {
		if keyPair.RotatesAt().Before(clockService.Now()) {
			algorithmsToRotate[keyPair.Algorithm()] = true
		}
	}

	for alg := range algorithmsToRotate {
		_, err := keyService.Generate(clockService, server.Name(), alg)
		if err != nil {
			return fmt.Errorf("generating key pair: %w", err)
		}

	}
	return nil
}
