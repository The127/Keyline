package behaviours

import (
	"Keyline/logging"
	"Keyline/mediator"
	"Keyline/utils"
	"context"
	"fmt"
)

type Policy interface {
	IsAllowed(ctx context.Context) (bool, error)
}

func PolicyBehaviour(ctx context.Context, request Policy, next mediator.Next) error {
	logging.Logger.Infof("request: %v", request)
	allowed, err := request.IsAllowed(ctx)
	if err != nil {
		return fmt.Errorf("failed to check if request is allowed: %w", err)
	}

	if !allowed {
		logging.Logger.Infof("request not allowed")
		return fmt.Errorf("request not allowed: %w", utils.ErrHttpUnauthorized)
	}

	logging.Logger.Infof("request allowed")
	return next()
}
