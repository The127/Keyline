package commands

import (
	"Keyline/internal/authentication/permissions"
	"Keyline/internal/behaviours"
	"Keyline/internal/middlewares"
	"Keyline/internal/repositories"
	"Keyline/ioc"
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
)

type UpdateUserMetadata struct {
	VirtualServerName string
	UserId            uuid.UUID
	Metadata          map[string]any
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

	virtualServerRepository := ioc.GetDependency[repositories.VirtualServerRepository](scope)
	virtualServerFilter := repositories.NewVirtualServerFilter().Name(command.VirtualServerName)
	virtualServer, err := virtualServerRepository.Single(ctx, virtualServerFilter)
	if err != nil {
		return nil, fmt.Errorf("getting virtual server: %w", err)
	}

	userRepository := ioc.GetDependency[repositories.UserRepository](scope)
	userFilter := repositories.NewUserFilter().Id(command.UserId).VirtualServerId(virtualServer.Id())
	user, err := userRepository.Single(ctx, userFilter)
	if err != nil {
		return nil, fmt.Errorf("getting user: %w", err)
	}

	jsonString, err := json.Marshal(command.Metadata)
	if err != nil {
		return nil, fmt.Errorf("marshaling metadata: %w", err)
	}

	user.SetMetadata(string(jsonString))
	err = userRepository.Update(ctx, user)
	if err != nil {
		return nil, fmt.Errorf("updating user: %w", err)
	}

	return &UpdateUserMetadataResponse{}, nil
}
