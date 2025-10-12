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

type PatchUserAppMetadata struct {
	VirtualServerName string
	UserId            uuid.UUID
	ApplicationId     uuid.UUID
	Metadata          map[string]any
}

func (a PatchUserAppMetadata) IsAllowed(ctx context.Context) (behaviours.PolicyResult, error) {
	return behaviours.PermissionBasedPolicy(ctx, permissions.AppMetadataUpdateAny)
}

func (a PatchUserAppMetadata) GetRequestName() string {
	return "PatchUserAppMetadata"
}

type PatchUserAppMetadataResponse struct{}

func HandlePatchUserAppMetadata(ctx context.Context, command PatchUserAppMetadata) (*PatchUserAppMetadataResponse, error) {
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

	applicationRepository := ioc.GetDependency[repositories.ApplicationRepository](scope)
	applicationFilter := repositories.NewApplicationFilter().Id(command.ApplicationId).VirtualServerId(virtualServer.Id())
	application, err := applicationRepository.Single(ctx, applicationFilter)
	if err != nil {
		return nil, fmt.Errorf("getting application: %w", err)
	}

	applicationUserMetadataRepository := ioc.GetDependency[repositories.ApplicationUserMetadataRepository](scope)
	applicationUserMetadataFilter := repositories.NewApplicationUserMetadataFilter().
		ApplicationId(application.Id()).
		UserId(user.Id())
	applicationUserMetadata, err := applicationUserMetadataRepository.Single(ctx, applicationUserMetadataFilter)
	if err != nil {
		return nil, fmt.Errorf("getting application user metadata: %w", err)
	}

	metadata := make(map[string]any)
	metadataString := applicationUserMetadata.Metadata()
	if metadataString != "" {
		err = json.Unmarshal([]byte(metadataString), &metadata)
		if err != nil {
			return nil, fmt.Errorf("unmarshaling metadata: %w", err)
		}
	}

	utils.JsonMergePatch(metadata, command.Metadata)

	jsonString, err := json.Marshal(metadata)
	if err != nil {
		return nil, fmt.Errorf("marshaling metadata: %w", err)
	}

	applicationUserMetadata.SetMetadata(string(jsonString))
	err = applicationUserMetadataRepository.Update(ctx, applicationUserMetadata)
	if err != nil {
		return nil, fmt.Errorf("updating application user metadata: %w", err)
	}

	return &PatchUserAppMetadataResponse{}, nil
}
