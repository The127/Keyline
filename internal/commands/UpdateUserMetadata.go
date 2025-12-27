package commands

import (
	"Keyline/internal/authentication/permissions"
	"Keyline/internal/behaviours"
	"Keyline/internal/database"
	"Keyline/internal/middlewares"
	"Keyline/internal/repositories"
	"context"
	"encoding/json"
	"fmt"

	"github.com/The127/ioc"

	"github.com/google/uuid"
)

type UpdateUserMetadata struct {
	VirtualServerName string
	UserId            uuid.UUID
	Metadata          map[string]any
}

func (a UpdateUserMetadata) LogRequest() bool {
	return true
}

func (a UpdateUserMetadata) LogResponse() bool {
	return true
}

func (a UpdateUserMetadata) IsAllowed(ctx context.Context) (behaviours.PolicyResult, error) {
	return behaviours.PermissionBasedPolicy(ctx, permissions.UserMetadataUpdate)
}

func (a UpdateUserMetadata) GetRequestName() string {
	return "UpdateUserMetadata"
}

type UpdateUserMetadataResponse struct{}

func HandleUpdateUserMetadata(ctx context.Context, command UpdateUserMetadata) (*UpdateUserMetadataResponse, error) {
	scope := middlewares.GetScope(ctx)
	dbContext := ioc.GetDependency[database.Context](scope)

	virtualServerFilter := repositories.NewVirtualServerFilter().Name(command.VirtualServerName)
	virtualServer, err := dbContext.VirtualServers().FirstOrErr(ctx, virtualServerFilter)
	if err != nil {
		return nil, fmt.Errorf("getting virtual server: %w", err)
	}

	userFilter := repositories.NewUserFilter().Id(command.UserId).VirtualServerId(virtualServer.Id())
	user, err := dbContext.Users().Single(ctx, userFilter)
	if err != nil {
		return nil, fmt.Errorf("getting user: %w", err)
	}

	jsonString, err := json.Marshal(command.Metadata)
	if err != nil {
		return nil, fmt.Errorf("marshaling metadata: %w", err)
	}

	user.SetMetadata(string(jsonString))

	dbContext.Users().Update(user)
	return &UpdateUserMetadataResponse{}, nil
}
