package commands

import (
	"context"
	"fmt"
	"github.com/The127/Keyline/internal/authentication/permissions"
	"github.com/The127/Keyline/internal/behaviours"
	"github.com/The127/Keyline/internal/database"
	"github.com/The127/Keyline/internal/middlewares"
	"github.com/The127/Keyline/internal/repositories"

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
	dbContext := ioc.GetDependency[database.Context](scope)

	virtualServerFilter := repositories.NewVirtualServerFilter().Name(command.VirtualServerName)
	virtualServer, err := dbContext.VirtualServers().FirstOrErr(ctx, virtualServerFilter)
	if err != nil {
		return nil, fmt.Errorf("getting virtual server: %w", err)
	}

	userFilter := repositories.NewUserFilter().
		VirtualServerId(virtualServer.Id()).
		Id(command.ServiceUserId).
		ServiceUser(true)
	user, err := dbContext.Users().FirstOrErr(ctx, userFilter)
	if err != nil {
		return nil, fmt.Errorf("getting user: %w", err)
	}

	var kid string
	if command.Kid == nil {
		kid = uuid.New().String()
	} else {
		kid = *command.Kid
	}

	credential := repositories.NewCredential(user.Id(), &repositories.CredentialServiceUserKey{
		Kid:       kid,
		PublicKey: command.PublicKey,
	})
	dbContext.Credentials().Insert(credential)

	return &AssociateServiceUserPublicKeyResponse{
		Id:  credential.Id(),
		Kid: kid,
	}, nil
}
