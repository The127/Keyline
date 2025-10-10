package behaviours

import (
	"Keyline/logging"
	"Keyline/mediator"
	"context"
)

type Policy interface {
	IsAllowed(ctx context.Context) (bool, error)
}

func PolicyBehaviour(ctx context.Context, request Policy, next mediator.Next) {
	logging.Logger.Infof("request: %v", request)
	allowed, err := request.IsAllowed(ctx)
	if err != nil {
		logging.Logger.Fatalf("failed to check if request is allowed: %v", err)
		return
	}

	if !allowed {
		logging.Logger.Infof("request not allowed")
		return
	}

	logging.Logger.Infof("request allowed")
	next()
}
