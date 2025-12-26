package commands

import (
	"Keyline/internal/authentication/permissions"
	"Keyline/internal/behaviours"
	"Keyline/internal/database"
	"Keyline/internal/middlewares"
	"Keyline/internal/repositories"
	"Keyline/utils"
	"context"
	"encoding/json"
	"fmt"

	"github.com/The127/ioc"

	"github.com/google/uuid"
)

type PatchUserAppMetadata struct {
	VirtualServerName string
	UserId            uuid.UUID
	ApplicationId     uuid.UUID
	Metadata          map[string]any
}

func (a PatchUserAppMetadata) LogRequest() bool {
	return true
}

func (a PatchUserAppMetadata) LogResponse() bool {
	return true
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

	applicationUserMetadataFilter := repositories.NewApplicationUserMetadataFilter().
		ApplicationId(application.Id()).
		UserId(user.Id())
	applicationUserMetadata, err := dbContext.ApplicationUserMetadata().Single(ctx, applicationUserMetadataFilter)
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

	dbContext.ApplicationUserMetadata().Update(applicationUserMetadata)
	return &PatchUserAppMetadataResponse{}, nil
}
