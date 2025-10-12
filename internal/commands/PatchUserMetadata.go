package commands

import (
	"Keyline/internal/authentication/permissions"
	"Keyline/internal/behaviours"
	"Keyline/internal/middlewares"
	"Keyline/internal/repositories"
	"Keyline/ioc"
	"Keyline/utils"
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
)

type PatchUserMetadata struct {
	VirtualServerName string
	UserId            uuid.UUID
	Metadata          map[string]any
}

func (a PatchUserMetadata) IsAllowed(ctx context.Context) (behaviours.PolicyResult, error) {
	return behaviours.PermissionBasedPolicy(ctx, permissions.UserMetadataUpdate)
}

func (a PatchUserMetadata) GetRequestName() string {
	return "PatchUserMetadata"
}

type PatchUserMetadataResponse struct{}

func HandlePatchUserMetadata(ctx context.Context, command PatchUserMetadata) (*PatchUserMetadataResponse, error) {
	scope := middlewares.GetScope(ctx)

	virtualServerRepository := ioc.GetDependency[repositories.VirtualServerRepository](scope)
	virtualServerFilter := repositories.NewVirtualServerFilter().Name(command.VirtualServerName)
	virtualServer, err := virtualServerRepository.Single(ctx, virtualServerFilter)
	if err != nil {
		return nil, fmt.Errorf("getting virtual server: %w", err)
	}

	userRepository := ioc.GetDependency[repositories.UserRepository](scope)
	userFilter := repositories.NewUserFilter().
		Id(command.UserId).
		VirtualServerId(virtualServer.Id()).
		IncludeMetadata()
	user, err := userRepository.Single(ctx, userFilter)
	if err != nil {
		return nil, fmt.Errorf("getting user: %w", err)
	}

	metadata := make(map[string]any)
	if user.Metadata() != "" {
		err = json.Unmarshal([]byte(user.Metadata()), &metadata)
		if err != nil {
			return nil, fmt.Errorf("unmarshaling metadata: %w", err)
		}
	}

	utils.JsonMergePatch(metadata, command.Metadata)

	jsonString, err := json.Marshal(metadata)
	if err != nil {
		return nil, fmt.Errorf("marshaling metadata: %w", err)
	}

	user.SetMetadata(string(jsonString))
	err = userRepository.Update(ctx, user)
	if err != nil {
		return nil, fmt.Errorf("updating user: %w", err)
	}

	return &PatchUserMetadataResponse{}, nil
}
