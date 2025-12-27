package queries

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

type GetUserMetadata struct {
	VirtualServerName             string
	UserId                        uuid.UUID
	IncludeGlobalMetadata         bool
	IncludeAllApplicationMetadata bool
	ApplicationIds                *[]uuid.UUID
}

func (a GetUserMetadata) LogRequest() bool {
	return true
}

func (a GetUserMetadata) LogResponse() bool {
	return false
}

func (a GetUserMetadata) IsAllowed(ctx context.Context) (behaviours.PolicyResult, error) {
	return behaviours.PermissionBasedPolicy(ctx, permissions.UserMetadataView)
}

func (a GetUserMetadata) GetRequestName() string {
	return "GetUserMetadata"
}

type GetUserMetadataResult struct {
	Metadata            string
	ApplicationMetadata map[string]string
}

func HandleGetUserMetadata(ctx context.Context, query GetUserMetadata) (*GetUserMetadataResult, error) {
	scope := middlewares.GetScope(ctx)
	dbContext := ioc.GetDependency[database.Context](scope)

	virtualServerFilter := repositories.NewVirtualServerFilter().Name(query.VirtualServerName)
	virtualServer, err := dbContext.VirtualServers().FirstOrErr(ctx, virtualServerFilter)
	if err != nil {
		return nil, fmt.Errorf("getting virtual server: %w", err)
	}

	userFilter := repositories.NewUserFilter().
		VirtualServerId(virtualServer.Id()).
		Id(query.UserId).
		IncludeMetadata()
	user, err := dbContext.Users().Single(ctx, userFilter)
	if err != nil {
		return nil, fmt.Errorf("getting user: %w", err)
	}

	result := GetUserMetadataResult{
		Metadata:            "",
		ApplicationMetadata: make(map[string]string),
	}

	if query.IncludeGlobalMetadata {
		result.Metadata = user.Metadata()
	}

	if query.ApplicationIds != nil || query.IncludeAllApplicationMetadata {
		applicationFilter := repositories.NewApplicationFilter()

		if query.ApplicationIds != nil {
			applicationFilter = applicationFilter.Ids(*query.ApplicationIds)
		}

		applications, _, err := dbContext.Applications().List(ctx, applicationFilter)
		if err != nil {
			return nil, fmt.Errorf("searching applications: %w", err)
		}

		appIds := make([]uuid.UUID, len(applications))
		for i, application := range applications {
			appIds[i] = application.Id()
		}

		applicationUserMetadataFilter := repositories.NewApplicationUserMetadataFilter().
			ApplicationIds(appIds).
			UserId(query.UserId)
		applicationMetadata, _, err := dbContext.ApplicationUserMetadata().List(ctx, applicationUserMetadataFilter)
		if err != nil {
			return nil, fmt.Errorf("searching application user metadata: %w", err)
		}

		for _, metadata := range applicationMetadata {
			var application *repositories.Application
			for _, a := range applications {
				if a.Id() == metadata.ApplicationId() {
					application = a
					break
				}
			}
			if application == nil {
				panic(fmt.Sprintf("application not found: %s", metadata.ApplicationId().String()))
			}

			result.ApplicationMetadata[application.Name()] = metadata.Metadata()
		}
	}

	return &result, nil
}
