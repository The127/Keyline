package behaviours

import (
	"context"
	"fmt"
	"github.com/The127/Keyline/internal/database"
	"github.com/The127/Keyline/internal/middlewares"

	"github.com/The127/ioc"
	"github.com/The127/mediatr"
)

func SaveChangesBehaviour(ctx context.Context, _ Policy, next mediatr.Next) (any, error) {
	response, err := next()
	if err != nil {
		return nil, err
	}

	scope := middlewares.GetScope(ctx)
	dbContext := ioc.GetDependency[database.Context](scope)

	if saveErr := dbContext.SaveChanges(ctx); saveErr != nil {
		return nil, fmt.Errorf("saving changes: %w", saveErr)
	}

	return response, nil
}
