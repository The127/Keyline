package commands

import (
	"Keyline/internal/middlewares"
	"Keyline/internal/repositories"
	"Keyline/ioc"
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
)

type UpdateUserAppMetadata struct {
	VirtualServerName string
	UserId            uuid.UUID
	ApplicationId     uuid.UUID
	Metadata          map[string]any
}

type UpdateUserAppMetadataResponse struct{}

func HandleUpdateUserAppMetadata(ctx context.Context, command UpdateUserAppMetadata) (*UpdateUserAppMetadataResponse, error) {
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
	applicationUserMetadatFilter := repositories.NewApplicationUserMetadataFilter().
		ApplicationId(application.Id()).
		UserId(user.Id())
	metadata, err := applicationUserMetadataRepository.First(ctx, applicationUserMetadatFilter)
	if err != nil {
		return nil, fmt.Errorf("getting application user metadata: %w", err)
	}

	jsonString, err := json.Marshal(command.Metadata)
	if err != nil {
		return nil, fmt.Errorf("marshaling metadata: %w", err)
	}

	if metadata == nil {
		metadata = repositories.NewApplicationUserMetadata(
			application.Id(),
			user.Id(),
			string(jsonString),
		)

		err := applicationUserMetadataRepository.Insert(ctx, metadata)
		if err != nil {
			return nil, fmt.Errorf("inserting application user metadata: %w", err)
		}
	} else {
		metadata.SetMetadata(string(jsonString))
		err := applicationUserMetadataRepository.Update(ctx, metadata)
		if err != nil {
			return nil, fmt.Errorf("updating application user metadata: %w", err)
		}
	}

	return &UpdateUserAppMetadataResponse{}, nil
}
