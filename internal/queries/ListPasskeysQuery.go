package queries

import (
	"Keyline/internal/authentication/permissions"
	"Keyline/internal/behaviours"
	"Keyline/internal/database"
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
	dbContext := ioc.GetDependency[database.Context](scope)

	virtualServerFilter := repositories.NewVirtualServerFilter().Name(query.VirtualServerName)
	virtualServer, err := dbContext.VirtualServers().FirstOrErr(ctx, virtualServerFilter)
	if err != nil {
		return nil, fmt.Errorf("getting virtual server: %w", err)
	}

	userFilter := repositories.NewUserFilter().
		VirtualServerId(virtualServer.Id()).
		Id(query.UserId)
	user, err := dbContext.Users().Single(ctx, userFilter)
	if err != nil {
		return nil, fmt.Errorf("getting user: %w", err)
	}

	credentialFilter := repositories.NewCredentialFilter().
		UserId(user.Id()).
		Type(repositories.CredentialTypeWebauthn)
	credentials, err := dbContext.Credentials().List(ctx, credentialFilter)
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
