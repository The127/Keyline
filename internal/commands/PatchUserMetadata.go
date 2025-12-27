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

type PatchUserMetadata struct {
	VirtualServerName string
	UserId            uuid.UUID
	Metadata          map[string]any
}

func (a PatchUserMetadata) LogRequest() bool {
	return true
}

func (a PatchUserMetadata) LogResponse() bool {
	return true
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
	dbContext := ioc.GetDependency[database.Context](scope)

	virtualServerFilter := repositories.NewVirtualServerFilter().Name(command.VirtualServerName)
	virtualServer, err := dbContext.VirtualServers().FirstOrErr(ctx, virtualServerFilter)
	if err != nil {
		return nil, fmt.Errorf("getting virtual server: %w", err)
	}

	userFilter := repositories.NewUserFilter().
		Id(command.UserId).
		VirtualServerId(virtualServer.Id()).
		IncludeMetadata()
	user, err := dbContext.Users().Single(ctx, userFilter)
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

	dbContext.Users().Update(user)
	return &PatchUserMetadataResponse{}, nil
}
