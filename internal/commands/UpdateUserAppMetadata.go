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

type UpdateUserAppMetadata struct {
	VirtualServerName string
	UserId            uuid.UUID
	ApplicationId     uuid.UUID
	Metadata          map[string]any
}

func (a UpdateUserAppMetadata) LogRequest() bool {
	return true
}

func (a UpdateUserAppMetadata) LogResponse() bool {
	return true
}

func (a UpdateUserAppMetadata) IsAllowed(ctx context.Context) (behaviours.PolicyResult, error) {
	return behaviours.PermissionBasedPolicy(ctx, permissions.AppMetadataUpdateAny)
}

func (a UpdateUserAppMetadata) GetRequestName() string {
	return "UpdateUserAppMetadata"
}

type UpdateUserAppMetadataResponse struct{}

func HandleUpdateUserAppMetadata(ctx context.Context, command UpdateUserAppMetadata) (*UpdateUserAppMetadataResponse, error) {
	scope := middlewares.GetScope(ctx)
	dbContext := ioc.GetDependency[database.Context](scope)

	virtualServerFilter := repositories.NewVirtualServerFilter().Name(command.VirtualServerName)
	virtualServer, err := dbContext.VirtualServers().Single(ctx, virtualServerFilter)
	if err != nil {
		return nil, fmt.Errorf("getting virtual server: %w", err)
	}

	userFilter := repositories.NewUserFilter().Id(command.UserId).VirtualServerId(virtualServer.Id())
	user, err := dbContext.Users().Single(ctx, userFilter)
	if err != nil {
		return nil, fmt.Errorf("getting user: %w", err)
	}

	applicationFilter := repositories.NewApplicationFilter().Id(command.ApplicationId).VirtualServerId(virtualServer.Id())
	application, err := dbContext.Applications().Single(ctx, applicationFilter)
	if err != nil {
		return nil, fmt.Errorf("getting application: %w", err)
	}

	applicationUserMetadatFilter := repositories.NewApplicationUserMetadataFilter().
		ApplicationId(application.Id()).
		UserId(user.Id())
	metadata, err := dbContext.ApplicationUserMetadata().FirstOrNil(ctx, applicationUserMetadatFilter)
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

		dbContext.ApplicationUserMetadata().Insert(metadata)
	} else {
		metadata.SetMetadata(string(jsonString))
		dbContext.ApplicationUserMetadata().Update(metadata)
	}

	return &UpdateUserAppMetadataResponse{}, nil
}
