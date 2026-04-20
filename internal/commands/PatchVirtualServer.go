package commands

import (
	"context"
	"fmt"
	"github.com/The127/Keyline/config"
	"github.com/The127/Keyline/internal/authentication/permissions"
	"github.com/The127/Keyline/internal/behaviours"
	"github.com/The127/Keyline/internal/database"
	"github.com/The127/Keyline/internal/middlewares"
	"github.com/The127/Keyline/internal/repositories"
	"github.com/The127/Keyline/internal/services"

	"github.com/The127/go-clock"
	"github.com/The127/ioc"
)

type PatchVirtualServer struct {
	VirtualServerName string
	DisplayName       *string

	EnableRegistration       *bool
	Require2fa               *bool
	RequireEmailVerification *bool

	PrimarySigningAlgorithm     *config.SigningAlgorithm
	AdditionalSigningAlgorithms *[]config.SigningAlgorithm
}

func (a PatchVirtualServer) LogRequest() bool {
	return true
}

func (a PatchVirtualServer) LogResponse() bool {
	return true
}

func (a PatchVirtualServer) IsAllowed(ctx context.Context) (behaviours.PolicyResult, error) {
	return behaviours.PermissionBasedPolicy(ctx, permissions.VirtualServerUpdate)
}

func (a PatchVirtualServer) GetRequestName() string {
	return "PatchVirtualServer"
}

type PatchVirtualServerResponse struct{}

func HandlePatchVirtualServer(ctx context.Context, command PatchVirtualServer) (*PatchVirtualServerResponse, error) {
	scope := middlewares.GetScope(ctx)
	dbContext := ioc.GetDependency[database.Context](scope)

	virtualServerFilter := repositories.NewVirtualServerFilter().Name(command.VirtualServerName)
	virtualServer, err := dbContext.VirtualServers().FirstOrErr(ctx, virtualServerFilter)
	if err != nil {
		return nil, fmt.Errorf("getting virtual server: %w", err)
	}

	if command.DisplayName != nil {
		virtualServer.SetDisplayName(*command.DisplayName)
	}

	if command.EnableRegistration != nil {
		virtualServer.SetEnableRegistration(*command.EnableRegistration)
	}

	if command.Require2fa != nil {
		virtualServer.SetRequire2fa(*command.Require2fa)
	}

	if command.RequireEmailVerification != nil {
		virtualServer.SetRequireEmailVerification(*command.RequireEmailVerification)
	}

	if command.PrimarySigningAlgorithm != nil {
		virtualServer.SetPrimarySigningAlgorithm(*command.PrimarySigningAlgorithm)
	}

	if command.AdditionalSigningAlgorithms != nil {
		virtualServer.SetAdditionalSigningAlgorithms(*command.AdditionalSigningAlgorithms)
	}

	dbContext.VirtualServers().Update(virtualServer)

	if command.PrimarySigningAlgorithm != nil || command.AdditionalSigningAlgorithms != nil {
		keyStore := ioc.GetDependency[services.KeyStore](scope)
		keyService := ioc.GetDependency[services.KeyService](scope)
		clockService := ioc.GetDependency[clock.Service](scope)

		for _, alg := range virtualServer.AllSigningAlgorithms() {
			existing, err := keyStore.GetAllForAlgorithm(command.VirtualServerName, alg)
			if err != nil {
				return nil, fmt.Errorf("checking keys for algorithm %s: %w", alg, err)
			}
			if len(existing) > 0 {
				continue
			}
			_, err = keyService.Generate(clockService, command.VirtualServerName, alg)
			if err != nil {
				return nil, fmt.Errorf("generating key for algorithm %s: %w", alg, err)
			}
		}
	}

	return &PatchVirtualServerResponse{}, nil
}
