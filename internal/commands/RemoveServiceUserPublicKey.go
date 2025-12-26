package commands

import (
	"Keyline/internal/authentication/permissions"
	"Keyline/internal/behaviours"
	"Keyline/internal/database"
	"Keyline/internal/middlewares"
	"Keyline/internal/repositories"
	"context"
	"fmt"

	"github.com/The127/ioc"

	"github.com/google/uuid"
)

type RemoveServiceUserPublicKey struct {
	VirtualServerName string
	ServiceUserId     uuid.UUID
	PublicKey         string
}

func (a RemoveServiceUserPublicKey) LogRequest() bool {
	return true
}

func (a RemoveServiceUserPublicKey) LogResponse() bool {
	return true
}

func (a RemoveServiceUserPublicKey) IsAllowed(ctx context.Context) (behaviours.PolicyResult, error) {
	return behaviours.PermissionBasedPolicy(ctx, permissions.ServiceUserRemoveKey)
}

func (a RemoveServiceUserPublicKey) GetRequestName() string {
	return "RemoveServiceUserPublicKey"
}

type RemoveServiceUserPublicKeyResponse struct{}

func HandleRemoveServiceUserPublicKey(ctx context.Context, command RemoveServiceUserPublicKey) (*RemoveServiceUserPublicKeyResponse, error) {
	scope := middlewares.GetScope(ctx)
	dbContext := ioc.GetDependency[database.Context](scope)

	virtualServerFilter := repositories.NewVirtualServerFilter().Name(command.VirtualServerName)
	virtualServer, err := dbContext.VirtualServers().Single(ctx, virtualServerFilter)
	if err != nil {
		return nil, fmt.Errorf("getting virtual server: %w", err)
	}

	userFilter := repositories.NewUserFilter().
		VirtualServerId(virtualServer.Id()).
		ServiceUser(true).
		Id(command.ServiceUserId)
	user, err := dbContext.Users().Single(ctx, userFilter)
	if err != nil {
		return nil, fmt.Errorf("getting user: %w", err)
	}

	credentialFilter := repositories.NewCredentialFilter().
		UserId(user.Id()).
		Type(repositories.CredentialTypeServiceUserKey).
		DetailPublicKey(command.PublicKey)
	credential, err := dbContext.Credentials().Single(ctx, credentialFilter)
	if err != nil {
		return nil, fmt.Errorf("getting credential: %w", err)
	}

	dbContext.Credentials().Delete(credential.Id())
	return &RemoveServiceUserPublicKeyResponse{}, nil
}
