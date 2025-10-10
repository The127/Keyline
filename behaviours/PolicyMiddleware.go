package behaviours

import (
	"Keyline/logging"
	"Keyline/mediator"
	"Keyline/utils"
	"context"
	"fmt"

	"github.com/google/uuid"
)

type PolicyResult struct {
	allowed bool
	userId  uuid.UUID
}

func Allowed(userId uuid.UUID) PolicyResult {
	return PolicyResult{
		allowed: true,
		userId:  userId,
	}
}

func Denied(userId uuid.UUID) PolicyResult {
	return PolicyResult{
		allowed: false,
		userId:  userId,
	}
}

type Policy interface {
	IsAllowed(ctx context.Context) (PolicyResult, error)
}

func PolicyBehaviour(ctx context.Context, request Policy, next mediator.Next) error {
	logging.Logger.Infof("request: %v", request)
	policyResult, err := request.IsAllowed(ctx)
	if err != nil {
		return fmt.Errorf("failed to check if request is allowed: %w", err)
	}

	if !policyResult.allowed {
		logging.Logger.Infof("request not allowed")
		return fmt.Errorf("request not allowed: %w", utils.ErrHttpUnauthorized)
	}

	logging.Logger.Infof("request allowed")
	return next()
}
