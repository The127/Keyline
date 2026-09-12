package commands

import (
	"context"
	"fmt"
	"github.com/The127/Keyline/internal/authentication/permissions"
	"github.com/The127/Keyline/internal/behaviours"
	"github.com/The127/Keyline/internal/database"
	"github.com/The127/Keyline/internal/middlewares"
	"github.com/The127/Keyline/internal/repositories"
	"github.com/The127/Keyline/utils"

	"github.com/The127/ioc"

	"github.com/google/uuid"
)

type AddApplicationKey struct {
	VirtualServerName string
	ProjectSlug       string
	ApplicationId     uuid.UUID
	Kid               *string
	PublicKey         string
}

func (a AddApplicationKey) LogRequest() bool {
	return true
}

func (a AddApplicationKey) LogResponse() bool {
	return true
}

func (a AddApplicationKey) IsAllowed(ctx context.Context) (behaviours.PolicyResult, error) {
	return behaviours.PermissionBasedPolicy(ctx, permissions.ApplicationAddKey)
}

func (a AddApplicationKey) GetRequestName() string {
	return "AddApplicationKey"
}

type AddApplicationKeyResponse struct {
	Id  uuid.UUID
	Kid string
}

func HandleAddApplicationKey(ctx context.Context, command AddApplicationKey) (*AddApplicationKeyResponse, error) {
	scope := middlewares.GetScope(ctx)
	dbContext := ioc.GetDependency[database.Context](scope)

	virtualServerFilter := repositories.NewVirtualServerFilter().Name(command.VirtualServerName)
	virtualServer, err := dbContext.VirtualServers().FirstOrErr(ctx, virtualServerFilter)
	if err != nil {
		return nil, fmt.Errorf("getting virtual server: %w", err)
	}

	projectFilter := repositories.NewProjectFilter().VirtualServerId(virtualServer.Id()).Slug(command.ProjectSlug)
	project, err := dbContext.Projects().FirstOrErr(ctx, projectFilter)
	if err != nil {
		return nil, fmt.Errorf("getting project: %w", err)
	}

	applicationFilter := repositories.NewApplicationFilter().
		VirtualServerId(virtualServer.Id()).
		ProjectId(project.Id()).
		Id(command.ApplicationId)
	application, err := dbContext.Applications().FirstOrErr(ctx, applicationFilter)
	if err != nil {
		return nil, fmt.Errorf("getting application: %w", err)
	}

	if !application.AuthenticatesWith(repositories.TokenEndpointAuthMethodPrivateKeyJwt) {
		return nil, fmt.Errorf("application does not authenticate with private_key_jwt: %w", utils.ErrHttpBadRequest)
	}

	_, err = utils.ParseAndValidatePublicKeyPem(command.PublicKey)
	if err != nil {
		return nil, fmt.Errorf("parsing public key: %w", err)
	}

	kid := uuid.New().String()
	if command.Kid != nil {
		kid = *command.Kid
	}

	applicationKeyFilter := repositories.NewApplicationKeyFilter().
		ApplicationId(application.Id()).
		Kid(kid)
	existingKey, err := dbContext.ApplicationKeys().FirstOrNil(ctx, applicationKeyFilter)
	if err != nil {
		return nil, fmt.Errorf("getting application key: %w", err)
	}
	if existingKey != nil {
		return nil, fmt.Errorf("application key with kid %s already exists: %w", kid, utils.ErrHttpConflict)
	}

	applicationKey := repositories.NewApplicationKey(application.Id(), kid, command.PublicKey)
	dbContext.ApplicationKeys().Insert(applicationKey)

	return &AddApplicationKeyResponse{
		Id:  applicationKey.Id(),
		Kid: kid,
	}, nil
}
