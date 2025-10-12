package queries

import (
	"Keyline/internal/middlewares"
	"Keyline/internal/repositories"
	"Keyline/ioc"
	"context"
	"fmt"

	"github.com/google/uuid"
)

type GetUserMetadata struct {
	VirtualServerName             string
	UserId                        uuid.UUID
	IncludeGlobalMetadata         bool
	IncludeAllApplicationMetadata bool
	ApplicationIds                *[]uuid.UUID
}

type GetUserMetadataResult struct {
	Metadata            string
	ApplicationMetadata map[string]string
}

func HandleGetUserMetadata(ctx context.Context, query GetUserMetadata) (*GetUserMetadataResult, error) {
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
		Id(query.UserId).
		IncludeMetadata()
	user, err := userRepository.Single(ctx, userFilter)
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
		applicationRepository := ioc.GetDependency[repositories.ApplicationRepository](scope)
		applicationFilter := repositories.NewApplicationFilter()

		if query.ApplicationIds != nil {
			applicationFilter = applicationFilter.Ids(*query.ApplicationIds)
		}

		applications, _, err := applicationRepository.List(ctx, applicationFilter)
		if err != nil {
			return nil, fmt.Errorf("searching applications: %w", err)
		}

		appIds := make([]uuid.UUID, len(applications))
		for i, application := range applications {
			appIds[i] = application.Id()
		}

		applicationUserMetadataRepository := ioc.GetDependency[repositories.ApplicationUserMetadataRepository](scope)
		applicationUserMetadataFilter := repositories.NewApplicationUserMetadataFilter().
			ApplicationIds(appIds).
			UserId(query.UserId)
		applicationMetadata, _, err := applicationUserMetadataRepository.List(ctx, applicationUserMetadataFilter)
		if err != nil {
			return nil, fmt.Errorf("searching application user metadata: %w", err)
		}

		for _, metadata := range applicationMetadata {
			var application *repositories.Application
			for _, application = range applications {
				if application.Id() == metadata.ApplicationId() {
					application = application
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
