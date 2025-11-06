package queries

import (
	"Keyline/internal/authentication/permissions"
	"Keyline/internal/behaviours"
	"Keyline/internal/middlewares"
	"Keyline/internal/repositories"
	"Keyline/utils"
	"context"
	"fmt"
	"github.com/The127/ioc"

	"github.com/google/uuid"
)

type ListPasskeys struct {
	VirtualServerName string
	UserId            uuid.UUID
}

func (a ListPasskeys) LogRequest() bool {
	return false
}

func (a ListPasskeys) LogResponse() bool {
	return false
}

func (a ListPasskeys) IsAllowed(ctx context.Context) (behaviours.PolicyResult, error) {
	return behaviours.PermissionBasedPolicy(ctx, permissions.UserView)
}

func (a ListPasskeys) GetRequestName() string {
	return "ListPasskeys"
}

type ListPasskeysResponse struct {
	PagedResponse[ListPasskeysResponseItem]
}

type ListPasskeysResponseItem struct {
	Id uuid.UUID
}

func HandleListPasskeys(ctx context.Context, query ListPasskeys) (*ListPasskeysResponse, error) {
	scope := middlewares.GetScope(ctx)

	virtualServerRepository := ioc.GetDependency[repositories.VirtualServerRepository](scope)
	virtualServerFilter := repositories.NewVirtualServerFilter().Name(query.VirtualServerName)
	virtualServer, err := virtualServerRepository.Single(ctx, virtualServerFilter)
	if err != nil {
		return nil, fmt.Errorf("getting virtual server: %w", err)
	}

	userRepository := ioc.GetDependency[repositories.UserRepository](scope)
	userFilter := repositories.NewUserFilter().
		VirtualServerId(virtualServer.Id()).
		Id(query.UserId)
	user, err := userRepository.Single(ctx, userFilter)
	if err != nil {
		return nil, fmt.Errorf("getting user: %w", err)
	}

	credentialRepository := ioc.GetDependency[repositories.CredentialRepository](scope)
	credentialFilter := repositories.NewCredentialFilter().
		UserId(user.Id()).
		Type(repositories.CredentialTypeWebauthn)
	credentials, err := credentialRepository.List(ctx, credentialFilter)
	if err != nil {
		return nil, fmt.Errorf("getting credentials: %w", err)
	}

	items := utils.MapSlice(credentials, func(x *repositories.Credential) ListPasskeysResponseItem {
		return ListPasskeysResponseItem{
			Id: x.Id(),
		}
	})

	return &ListPasskeysResponse{
		PagedResponse: NewPagedResponse(items, len(credentials)),
	}, nil
}
