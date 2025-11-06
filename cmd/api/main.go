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
	"Keyline/utils"
	"context"
	"fmt"
	"net/url"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/The127/ioc"
	"github.com/The127/mediatr"

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
	m := ioc.GetDependency[mediatr.Mediator](scope)

	// check if there are no virtual servers
	existsResult, err := mediatr.Send[*queries.AnyVirtualServerExistsResult](ctx, m, queries.AnyVirtualServerExists{})
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

	var adminConfig *commands.CreateVirtualServerAdmin = nil
	if config.C.InitialVirtualServer.CreateAdmin {
		adminConfig = &commands.CreateVirtualServerAdmin{
			Username:     config.C.InitialVirtualServer.Admin.Username,
			DisplayName:  config.C.InitialVirtualServer.Admin.DisplayName,
			PrimaryEmail: config.C.InitialVirtualServer.Admin.PrimaryEmail,
			PasswordHash: config.C.InitialVirtualServer.Admin.PasswordHash,
		}
	}

	var serviceUsers []commands.CreateVirtualServerServiceUser = nil //nolint:prealloc
	for _, serviceUser := range config.C.InitialVirtualServer.ServiceUsers {
		serviceUsers = append(serviceUsers, commands.CreateVirtualServerServiceUser{
			Username:  serviceUser.Username,
			Roles:     serviceUser.Roles,
			PublicKey: serviceUser.PublicKey,
		})
	}

	var projects []commands.CreateVirtualServerProject = nil //nolint:prealloc
	for _, project := range config.C.InitialVirtualServer.Projects {
		var apps []commands.CreateVirtualServerProjectApplication = nil
		for _, app := range project.Applications {
			apps = append(apps, commands.CreateVirtualServerProjectApplication{
				Name:           app.Name,
				DisplayName:    app.DisplayName,
				Type:           app.Type,
				HashedSecret:   app.HashedSecret,
				RedirectUris:   app.RedirectUris,
				PostLogoutUris: app.PostLogoutRedirectUris,
			})
		}

		var roles []commands.CreateVirtualServerProjectRole = nil
		for _, role := range project.Roles {
			roles = append(roles, commands.CreateVirtualServerProjectRole{
				Name:        role.Name,
				Description: role.Description,
			})
		}

		var resourceServers []commands.CreateVirtualServerProjectResourceServer = nil
		for _, resourceServer := range project.ResourceServers {
			resourceServers = append(resourceServers, commands.CreateVirtualServerProjectResourceServer{
				Name:        resourceServer.Name,
				Slug:        resourceServer.Slug,
				Description: resourceServer.Description,
			})
		}

		projects = append(projects, commands.CreateVirtualServerProject{
			Slug:            project.Slug,
			Name:            project.Name,
			Description:     project.Description,
			Applications:    apps,
			Roles:           roles,
			ResourceServers: resourceServers,
		})
	}

	_, err = mediatr.Send[*commands.CreateVirtualServerResponse](ctx, m, commands.CreateVirtualServer{
		Name:               config.C.InitialVirtualServer.Name,
		DisplayName:        config.C.InitialVirtualServer.DisplayName,
		EnableRegistration: config.C.InitialVirtualServer.EnableRegistration,
		SigningAlgorithm:   config.C.InitialVirtualServer.SigningAlgorithm,

		CreateSystemAdminRole: config.C.InitialVirtualServer.CreateSystemAdminRole,

		Admin:        adminConfig,
		ServiceUsers: serviceUsers,
		Projects:     projects,
	})
	if err != nil {
		logging.Logger.Fatalf("failed to create initial virtual server: %v", err)
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
