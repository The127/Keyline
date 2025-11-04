package commands

import (
	"Keyline/internal/authentication/permissions"
	"Keyline/internal/behaviours"
	"Keyline/internal/middlewares"
	"Keyline/internal/repositories"
	"Keyline/ioc"
	"context"
	"fmt"

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

	virtualServerRepository := ioc.GetDependency[repositories.VirtualServerRepository](scope)
	virtualServerFilter := repositories.NewVirtualServerFilter().Name(command.VirtualServerName)
	virtualServer, err := virtualServerRepository.Single(ctx, virtualServerFilter)
	if err != nil {
		return nil, fmt.Errorf("getting virtual server: %w", err)
	}

	userRepository := ioc.GetDependency[repositories.UserRepository](scope)
	userFilter := repositories.NewUserFilter().
		VirtualServerId(virtualServer.Id()).
		ServiceUser(true).
		Id(command.ServiceUserId)
	user, err := userRepository.Single(ctx, userFilter)
	if err != nil {
		return nil, fmt.Errorf("getting user: %w", err)
	}

	credentialRepository := ioc.GetDependency[repositories.CredentialRepository](scope)
	credentialFilter := repositories.NewCredentialFilter().
		UserId(user.Id()).
		Type(repositories.CredentialTypeServiceUserKey).
		DetailPublicKey(command.PublicKey)
	credential, err := credentialRepository.Single(ctx, credentialFilter)
	if err != nil {
		return nil, fmt.Errorf("getting credential: %w", err)
	}

	err = credentialRepository.Delete(ctx, credential.Id())
	if err != nil {
		return nil, fmt.Errorf("deleting credential: %w", err)
	}

	return &RemoveServiceUserPublicKeyResponse{}, nil
}
