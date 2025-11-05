// @title       Keyline API
// @description Open source OIDC/IDP server.
// @BasePath    /

// Security schemes for the "Authorize" button (Swagger 2.0):
// @securityDefinitions.basic  BasicAuth
// @securityDefinitions.apikey BearerAuth
// @in                         header
// @name                       Authorization
package main

import (
	"Keyline/internal/authentication"
	"Keyline/internal/clock"
	"Keyline/internal/commands"
	"Keyline/internal/config"
	"Keyline/internal/database"
	"Keyline/internal/jobs"
	"Keyline/internal/logging"
	"Keyline/internal/metrics"
	"Keyline/internal/middlewares"
	"Keyline/internal/queries"
	"Keyline/internal/quorum"
	"Keyline/internal/repositories"
	"Keyline/internal/server"
	"Keyline/internal/setup"
	"Keyline/ioc"
	"Keyline/mediator"
	"Keyline/utils"
	"context"
	"fmt"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"Keyline/docs"

	"github.com/huandu/go-sqlbuilder"
)

func tryFiveTimes(f func() error, msg string) {
	var err error
	for i := 0; i < 5; i++ {
		err = f()
		if err == nil {
			return
		}

		logging.Logger.Infof(msg+": %v", err)
		logging.Logger.Infof("Retrying in 5 seconds (attempt %d/5)", i+1)
		time.Sleep(5 * time.Second)
	}

	panic(err)
}

func main() {
	config.Init()
	configureSwaggerFromConfig()

	sqlbuilder.DefaultFlavor = sqlbuilder.PostgreSQL

	logging.Init()
	metrics.Init()

	tryFiveTimes(func() error {
		return database.Migrate(config.C.Database.Postgres)
	}, "failed to migrate database")

	dc := ioc.NewDependencyCollection()

	ioc.RegisterSingleton(dc, func(dp *ioc.DependencyProvider) clock.Service {
		return clock.NewClockService()
	})

	setup.OutboxDelivery(dc, config.QueueModeInProcess)
	setup.KeyServices(dc, config.C.KeyStore.Mode)
	setup.Caching(dc, config.C.Cache.Mode)
	setup.Services(dc)
	setup.Repositories(dc, config.C.Database.Mode, config.C.Database.Postgres)
	setup.Mediator(dc)
	dp := dc.BuildProvider()

	initApplication(dp)

	var jobManager jobs.JobManager
	leaderElection := quorum.NewLeaderElectionFactory().
		OnLeaderChange(func(isLeader bool) {
			if isLeader {
				jobManager = jobs.NewJobManager(jobs.WithOnError(func(err error) {
					logging.Logger.Errorf("an error happened while running a job: %v", err)
				}))

				jobManager.QueueJob(
					jobs.OutboxSendingJob(dp),
					time.Second*10,
					jobs.WithName("outbox_sender"),
					jobs.WithStartImmediate(),
				)

				jobManager.QueueJob(
					jobs.KeyRotateJob(),
					time.Hour,
					jobs.WithName("signing_key_rotation"),
					jobs.WithStartImmediate(),
				)

				logging.Logger.Info("Starting job manager")
				jobManager.Start(middlewares.ContextWithScope(context.Background(), dp))
			} else {
				logging.Logger.Info("Stopping job manager")
				if jobManager != nil {
					jobManager.Stop()
				}
			}
		}).
		Build(config.C.LeaderElection)
	err := leaderElection.Start(middlewares.ContextWithScope(context.Background(), dp))
	if err != nil {
		panic(fmt.Errorf("failed to start leader election: %s", err.Error()))
	}

	server.Serve(dp, config.C.Server)

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c
}

// initApplication sets up the initial application state on the first startup.
// It creates an initial virtual server and other necessary defaults if none exist.
func initApplication(dp *ioc.DependencyProvider) {
	scope := dp.NewScope()

	ctx := middlewares.ContextWithScope(context.Background(), scope)
	ctx = authentication.ContextWithCurrentUser(ctx, authentication.SystemUser())
	m := ioc.GetDependency[mediator.Mediator](scope)

	// check if there are no virtual servers
	existsResult, err := mediator.Send[*queries.AnyVirtualServerExistsResult](ctx, m, queries.AnyVirtualServerExists{})
	if err != nil {
		logging.Logger.Fatalf("failed to query if any virtual servers exist: %v", err)
	}

	if existsResult.Found {
		return
	}

	logging.Logger.Info("Creating system user")
	userRepository := ioc.GetDependency[repositories.UserRepository](scope)
	systemUser := repositories.NewSystemUser("system-user")
	err = userRepository.Insert(ctx, systemUser)
	if err != nil {
		logging.Logger.Fatalf("failed to create system user: %v", err)
	}

	logging.Logger.Infof("Creating initial virtual server")

	createVirtualServerResponse, err := mediator.Send[*commands.CreateVirtualServerResponse](ctx, m, commands.CreateVirtualServer{
		Name:               config.C.InitialVirtualServer.Name,
		DisplayName:        config.C.InitialVirtualServer.DisplayName,
		EnableRegistration: config.C.InitialVirtualServer.EnableRegistration,
		SigningAlgorithm:   config.C.InitialVirtualServer.SigningAlgorithm,
	})
	if err != nil {
		logging.Logger.Fatalf("failed to create initial virtual server: %v", err)
	}

	for _, projectConfig := range config.C.InitialVirtualServer.Projects {
		_, err := mediator.Send[*commands.CreateProjectResponse](ctx, m, commands.CreateProject{
			VirtualServerName: config.C.InitialVirtualServer.Name,
			Slug:              projectConfig.Slug,
			Name:              projectConfig.Name,
			Description:       projectConfig.Description,
		})
		if err != nil {
			logging.Logger.Fatalf("failed to create initial project: %v", err)
		}

		for _, applicationConfig := range projectConfig.Applications {
			_, err := mediator.Send[*commands.CreateApplicationResponse](ctx, m, commands.CreateApplication{
				VirtualServerName:      config.C.InitialVirtualServer.Name,
				ProjectSlug:            projectConfig.Slug,
				Name:                   applicationConfig.Name,
				DisplayName:            applicationConfig.DisplayName,
				Type:                   repositories.ApplicationType(applicationConfig.Type),
				RedirectUris:           applicationConfig.RedirectUris,
				PostLogoutRedirectUris: applicationConfig.PostLogoutRedirectUris,
				HashedSecret:           applicationConfig.HashedSecret,
			})
			if err != nil {
				logging.Logger.Fatalf("failed to create initial application: %v", err)
			}
		}

		for _, roleConfig := range projectConfig.Roles {
			_, err := mediator.Send[*commands.CreateRoleResponse](ctx, m, commands.CreateRole{
				VirtualServerName: config.C.InitialVirtualServer.Name,
				ProjectSlug:       projectConfig.Slug,
				Name:              roleConfig.Name,
				Description:       roleConfig.Description,
			})
			if err != nil {
				logging.Logger.Fatalf("failed to create initial role: %v", err)
			}
		}

		for _, resourceServerConfig := range projectConfig.ResourceServers {
			_, err := mediator.Send[*commands.CreateResourceServerResponse](ctx, m, commands.CreateResourceServer{
				VirtualServerName: config.C.InitialVirtualServer.Name,
				ProjectSlug:       projectConfig.Slug,
				Slug:              resourceServerConfig.Slug,
				Name:              resourceServerConfig.Name,
				Description:       resourceServerConfig.Description,
			})
			if err != nil {
				logging.Logger.Fatalf("failed to create initial resource server: %v", err)
			}
		}
	}

	if config.C.InitialVirtualServer.CreateAdmin {
		logging.Logger.Infof("Creating initial admin user")

		initialAdminUserInfo, err := mediator.Send[*commands.CreateUserResponse](ctx, m, commands.CreateUser{
			VirtualServerName: config.C.InitialVirtualServer.Name,
			DisplayName:       config.C.InitialVirtualServer.Admin.DisplayName,
			Username:          config.C.InitialVirtualServer.Admin.Username,
			Email:             config.C.InitialVirtualServer.Admin.PrimaryEmail,
			EmailVerified:     true,
		})
		if err != nil {
			logging.Logger.Fatalf("failed to create initial admin user: %v", err)
		}

		credentialRepository := ioc.GetDependency[repositories.CredentialRepository](scope)
		initialAdminCredential := repositories.NewCredential(initialAdminUserInfo.Id, &repositories.CredentialPasswordDetails{
			HashedPassword: config.C.InitialVirtualServer.Admin.PasswordHash,
			Temporary:      false,
		})
		err = credentialRepository.Insert(ctx, initialAdminCredential)
		if err != nil {
			logging.Logger.Fatalf("failed to create initial admin credential: %v", err)
		}

		_, err = mediator.Send[*commands.AssignRoleToUserResponse](ctx, m, commands.AssignRoleToUser{
			VirtualServerName: config.C.InitialVirtualServer.Name,
			ProjectSlug:       createVirtualServerResponse.SystemProjectSlug,
			UserId:            initialAdminUserInfo.Id,
			RoleId:            createVirtualServerResponse.AdminRoleId,
		})
		if err != nil {
			logging.Logger.Fatalf("failed to assign admin role to initial admin user: %v", err)
		}
	}

	for _, serviceUserConfig := range config.C.InitialVirtualServer.ServiceUsers {
		serviceUserResponse, err := mediator.Send[*commands.CreateServiceUserResponse](ctx, m, commands.CreateServiceUser{
			VirtualServerName: config.C.InitialVirtualServer.Name,
			Username:          serviceUserConfig.Username,
		})
		if err != nil {
			logging.Logger.Fatalf("failed to create initial service user: %v", err)
		}

		_, err = mediator.Send[*commands.AssociateServiceUserPublicKeyResponse](ctx, m, commands.AssociateServiceUserPublicKey{
			VirtualServerName: config.C.InitialVirtualServer.Name,
			ServiceUserId:     serviceUserResponse.Id,
			PublicKey:         serviceUserConfig.PublicKey,
		})
		if err != nil {
			logging.Logger.Fatalf("failed to associate initial service user public key: %v", err)
		}

		for _, configuredRole := range serviceUserConfig.Roles {
			if strings.Contains(configuredRole, " ") {
				split := strings.Split(configuredRole, " ")
				projectSlug := split[0]
				roleName := split[1]

				projectRepository := ioc.GetDependency[repositories.ProjectRepository](scope)
				projectFilter := repositories.NewProjectFilter().VirtualServerId(createVirtualServerResponse.Id).Slug(projectSlug)
				project, err := projectRepository.Single(ctx, projectFilter)
				if err != nil {
					logging.Logger.Fatalf("failed to get project: %v", err)
				}

				roleRepository := ioc.GetDependency[repositories.RoleRepository](scope)
				roleFilter := repositories.NewRoleFilter().
					VirtualServerId(createVirtualServerResponse.Id).
					ProjectId(project.Id()).
					Name(roleName)
				role, err := roleRepository.Single(ctx, roleFilter)
				if err != nil {
					logging.Logger.Fatalf("failed to get role: %v", err)
				}

				_, err = mediator.Send[*commands.AssignRoleToUserResponse](ctx, m, commands.AssignRoleToUser{
					VirtualServerName: config.C.InitialVirtualServer.Name,
					ProjectSlug:       createVirtualServerResponse.SystemProjectSlug,
					UserId:            serviceUserResponse.Id,
					RoleId:            role.Id(),
				})
				if err != nil {
					logging.Logger.Fatalf("failed to assign role to service user: %v", err)
				}
			} else {
				roleRepository := ioc.GetDependency[repositories.RoleRepository](scope)
				roleFilter := repositories.NewRoleFilter().
					VirtualServerId(createVirtualServerResponse.Id).
					Name(configuredRole)
				role, err := roleRepository.Single(ctx, roleFilter)
				if err != nil {
					logging.Logger.Fatalf("failed to get role: %v", err)
				}

				_, err = mediator.Send[*commands.AssignRoleToUserResponse](ctx, m, commands.AssignRoleToUser{
					VirtualServerName: config.C.InitialVirtualServer.Name,
					UserId:            serviceUserResponse.Id,
					RoleId:            role.Id(),
				})
				if err != nil {
					logging.Logger.Fatalf("failed to assign role to service user: %v", err)
				}
			}
		}
	}

	utils.PanicOnError(scope.Close, "failed creating scope to init application")
}

func configureSwaggerFromConfig() {
	if config.C.Server.ExternalUrl != "" {
		if u, err := url.Parse(config.C.Server.ExternalUrl); err == nil {
			if u.Host != "" {
				docs.SwaggerInfo.Host = u.Host
			}

			if u.Scheme != "" {
				docs.SwaggerInfo.Schemes = []string{u.Scheme}
			}
		}
	} else {
		docs.SwaggerInfo.Host = fmt.Sprintf("%s:%d", config.C.Server.Host, config.C.Server.Port)
	}

	if len(docs.SwaggerInfo.Schemes) == 0 {
		docs.SwaggerInfo.Schemes = []string{"http"}
	}

	docs.SwaggerInfo.BasePath = "/"
}
