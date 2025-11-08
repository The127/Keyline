package commands

import (
	"Keyline/internal/authentication/permissions"
	"Keyline/internal/behaviours"
	"Keyline/internal/middlewares"
	"Keyline/internal/repositories"
	"context"
	"fmt"

	"github.com/The127/ioc"

	"github.com/google/uuid"
)

type AssociateServiceUserPublicKey struct {
	Kid               *string
	VirtualServerName string
	ServiceUserId     uuid.UUID
	PublicKey         string
}

func (a AssociateServiceUserPublicKey) LogRequest() bool {
	return true
}

func (a AssociateServiceUserPublicKey) LogResponse() bool {
	return true
}

func (a AssociateServiceUserPublicKey) IsAllowed(ctx context.Context) (behaviours.PolicyResult, error) {
	return behaviours.PermissionBasedPolicy(ctx, permissions.ServiceUserAssociateKey)
}

func (a AssociateServiceUserPublicKey) GetRequestName() string {
	return "AssociateServiceUserPublicKey"
}

type AssociateServiceUserPublicKeyResponse struct {
	Id  uuid.UUID
	Kid string
}

func HandleAssociateServiceUserPublicKey(ctx context.Context, command AssociateServiceUserPublicKey) (*AssociateServiceUserPublicKeyResponse, error) {
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
		Id(command.ServiceUserId).
		ServiceUser(true)
	user, err := userRepository.Single(ctx, userFilter)
	if err != nil {
		return nil, fmt.Errorf("getting user: %w", err)
	}

	var kid string
	if command.Kid == nil {
		kid = uuid.New().String()
	} else {
		kid = *command.Kid
	}

	credentialRepository := ioc.GetDependency[repositories.CredentialRepository](scope)
	credential := repositories.NewCredential(user.Id(), &repositories.CredentialServiceUserKey{
		Kid:       kid,
		PublicKey: command.PublicKey,
	})
	err = credentialRepository.Insert(ctx, credential)
	if err != nil {
		return nil, fmt.Errorf("inserting credential: %w", err)
	}

	return &AssociateServiceUserPublicKeyResponse{
		Id:  credential.Id(),
		Kid: kid,
	}, nil
}
