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
)

type PatchVirtualServer struct {
	VirtualServerName string
	DisplayName       *string

	EnableRegistration       *bool
	Require2fa               *bool
	RequireEmailVerification *bool
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
	virtualServer, err := dbContext.VirtualServers().Single(ctx, virtualServerFilter)
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

	dbContext.VirtualServers().Update(virtualServer)
	return &PatchVirtualServerResponse{}, nil
}
