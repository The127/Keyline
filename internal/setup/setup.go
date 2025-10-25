package setup

import (
	"Keyline/internal/behaviours"
	"Keyline/internal/caching"
	"Keyline/internal/commands"
	"Keyline/internal/config"
	"Keyline/internal/database"
	"Keyline/internal/events"
	"Keyline/internal/middlewares"
	"Keyline/internal/password"
	"Keyline/internal/queries"
	"Keyline/internal/repositories"
	"Keyline/internal/repositories/postgres"
	"Keyline/internal/services"
	"Keyline/internal/services/audit"
	"Keyline/internal/services/claimsMapping"
	"Keyline/internal/services/keyValue"
	"Keyline/internal/services/outbox"
	"Keyline/ioc"
	"Keyline/mediator"
	"database/sql"
)

func Repositories(dc *ioc.DependencyCollection, mode config.DatabaseMode, c any) {
	switch mode {
	case config.DatabaseModeSqlite:
		panic("not implemented")

	case config.DatabaseModePostgres:
		pc, ok := c.(config.PostgresConfig)
		if !ok {
			panic("required postgres config missing")
		}
		postgresRepositories(dc, pc)

	default:
		panic("database mode missing or not supported")
	}
}

func postgresRepositories(dc *ioc.DependencyCollection, pc config.PostgresConfig) {
	ioc.RegisterSingleton(dc, func(dp *ioc.DependencyProvider) *sql.DB {
		return database.ConnectToDatabase(pc)
	})

	ioc.RegisterScoped(dc, func(dp *ioc.DependencyProvider) database.DbService {
		return database.NewDbService(dp)
	})
	ioc.RegisterCloseHandler(dc, func(dbService database.DbService) error {
		return dbService.Close()
	})

	ioc.RegisterScoped(dc, func(dp *ioc.DependencyProvider) repositories.UserRepository {
		return postgres.NewUserRepository()
	})
	ioc.RegisterScoped(dc, func(dp *ioc.DependencyProvider) repositories.VirtualServerRepository {
		return postgres.NewVirtualServerRepository()
	})
	ioc.RegisterScoped(dc, func(dp *ioc.DependencyProvider) repositories.CredentialRepository {
		return postgres.NewCredentialRepository()
	})
	ioc.RegisterScoped(dc, func(dp *ioc.DependencyProvider) repositories.OutboxMessageRepository {
		return postgres.NewOutboxMessageRepository()
	})
	ioc.RegisterScoped(dc, func(dp *ioc.DependencyProvider) repositories.FileRepository {
		return postgres.NewFileRepository()
	})
	ioc.RegisterScoped(dc, func(dp *ioc.DependencyProvider) repositories.TemplateRepository {
		return postgres.NewTemplateRepository()
	})
	ioc.RegisterScoped(dc, func(dp *ioc.DependencyProvider) repositories.RoleRepository {
		return postgres.NewRoleRepository()
	})
	ioc.RegisterScoped(dc, func(dp *ioc.DependencyProvider) repositories.GroupRepository {
		return postgres.NewGroupRepository()
	})
	ioc.RegisterScoped(dc, func(dp *ioc.DependencyProvider) repositories.GroupRoleRepository {
		return postgres.NewGroupRoleRepository()
	})
	ioc.RegisterScoped(dc, func(dp *ioc.DependencyProvider) repositories.UserRoleAssignmentRepository {
		return postgres.NewUserRoleAssignmentRepository()
	})
	ioc.RegisterScoped(dc, func(dp *ioc.DependencyProvider) repositories.ApplicationRepository {
		return postgres.NewApplicationRepository()
	})
	ioc.RegisterScoped(dc, func(dp *ioc.DependencyProvider) repositories.SessionRepository {
		return postgres.NewSessionRepository()
	})
	ioc.RegisterScoped(dc, func(dp *ioc.DependencyProvider) repositories.ApplicationUserMetadataRepository {
		return postgres.NewApplicationUserMetadataRepository()
	})
	ioc.RegisterScoped(dc, func(dp *ioc.DependencyProvider) repositories.AuditLogRepository {
		return postgres.NewAuditLogRepository()
	})
	ioc.RegisterScoped(dc, func(dp *ioc.DependencyProvider) repositories.PasswordRuleRepository {
		return postgres.NewPasswordRuleRepository()
	})
}

func Mediator(dc *ioc.DependencyCollection) {
	m := mediator.NewMediator()

	mediator.RegisterHandler(m, queries.HandleAnyVirtualServerExists)
	mediator.RegisterHandler(m, queries.HandleGetVirtualServerPublicInfo)
	mediator.RegisterHandler(m, queries.HandleGetVirtualServerQuery)
	mediator.RegisterHandler(m, commands.HandleCreateVirtualServer)

	mediator.RegisterHandler(m, queries.HandleListPasswordRules)

	mediator.RegisterHandler(m, queries.HandleListTemplates)
	mediator.RegisterHandler(m, queries.HandleGetTemplate)

	mediator.RegisterHandler(m, commands.HandleRegisterUser)
	mediator.RegisterHandler(m, commands.HandleCreateUser)
	mediator.RegisterHandler(m, commands.HandleVerifyEmail)
	mediator.RegisterHandler(m, commands.HandleResetPassword)
	mediator.RegisterHandler(m, queries.HandleGetUserQuery)
	mediator.RegisterHandler(m, commands.HandlePatchUser)
	mediator.RegisterHandler(m, queries.HandleListUsers)
	mediator.RegisterHandler(m, commands.HandleCreateServiceUser)
	mediator.RegisterHandler(m, commands.HandleAssociateServiceUserPublicKey)
	mediator.RegisterHandler(m, queries.HandleGetUserMetadata)
	mediator.RegisterHandler(m, commands.HandleUpdateUserMetadata)
	mediator.RegisterHandler(m, commands.HandleUpdateUserAppMetadata)
	mediator.RegisterHandler(m, commands.HandlePatchUserMetadata)
	mediator.RegisterHandler(m, commands.HandlePatchUserAppMetadata)

	mediator.RegisterHandler(m, commands.HandleCreateApplication)
	mediator.RegisterHandler(m, queries.HandleListApplications)
	mediator.RegisterHandler(m, queries.HandleGetApplication)
	mediator.RegisterHandler(m, commands.HandlePatchApplication)
	mediator.RegisterHandler(m, commands.HandleDeleteApplication)

	mediator.RegisterHandler(m, queries.HandleListRoles)
	mediator.RegisterHandler(m, queries.HandleGetRole)
	mediator.RegisterHandler(m, commands.HandleCreateRole)
	mediator.RegisterHandler(m, commands.HandleAssignRoleToUser)
	mediator.RegisterHandler(m, queries.HandleListUsersInRole)

	mediator.RegisterHandler(m, queries.HandleListGroups)

	mediator.RegisterHandler(m, queries.HandleListAuditEntries)

	mediator.RegisterEventHandler(m, events.QueueEmailVerificationJobOnUserCreatedEvent)

	mediator.RegisterBehaviour(m, behaviours.PolicyBehaviour)

	ioc.RegisterSingleton(dc, func(dp *ioc.DependencyProvider) mediator.Mediator {
		return m
	})
}

func OutboxDelivery(dc *ioc.DependencyCollection, queueMode config.QueueMode) {
	switch queueMode {
	case config.QueueModeNoop:
		ioc.RegisterSingleton(dc, func(_ *ioc.DependencyProvider) outbox.DeliveryService {
			return outbox.NewNoopDeliveryService()
		})

	case config.QueueModeInProcess:
		ioc.RegisterSingleton(dc, func(_ *ioc.DependencyProvider) outbox.DeliveryService {
			return outbox.NewInProcessDeliveryService()
		})

	default:
		panic("queue mode missing or not supported")
	}

	ioc.RegisterTransient(dc, func(dp *ioc.DependencyProvider) outbox.MessageBroker {
		return outbox.NewMessageBroker()
	})
}

func KeyServices(dc *ioc.DependencyCollection, keyStoreMode config.KeyStoreMode) {
	ioc.RegisterSingleton(dc, func(dp *ioc.DependencyProvider) services.KeyCache {
		return caching.NewMemoryCache[string, services.KeyPair]()
	})
	ioc.RegisterSingleton(dc, func(dp *ioc.DependencyProvider) services.KeyStore {
		switch keyStoreMode {
		case config.KeyStoreModeMemory:
			return services.NewMemoryKeyStore()

		case config.KeyStoreModeDirectory:
			return services.NewDirectoryKeyStore()

		case config.KeyStoreModeOpenBao:
			panic("not implemented yet")

		default:
			panic("not implemented")
		}
	})
	ioc.RegisterSingleton(dc, func(dp *ioc.DependencyProvider) services.KeyService {
		return services.NewKeyService(
			ioc.GetDependency[services.KeyCache](dp),
			ioc.GetDependency[services.KeyStore](dp),
		)
	})
}

func Services(dc *ioc.DependencyCollection) {
	ioc.RegisterSingleton(dc, func(dp *ioc.DependencyProvider) claimsMapping.ClaimsMapper {
		return claimsMapping.NewClaimsMapper()
	})
	ioc.RegisterSingleton(dc, func(dp *ioc.DependencyProvider) services.MailService {
		return services.NewMailService()
	})
	ioc.RegisterSingleton(dc, func(dp *ioc.DependencyProvider) services.TemplateService {
		return services.NewTemplateService()
	})
	ioc.RegisterSingleton(dc, func(dp *ioc.DependencyProvider) services.TokenService {
		return services.NewTokenService()
	})
	ioc.RegisterSingleton(dc, func(dp *ioc.DependencyProvider) middlewares.SessionService {
		return services.NewSessionService()
	})
	ioc.RegisterSingleton(dc, func(dp *ioc.DependencyProvider) behaviours.AuditLogger {
		return audit.NewDbAuditLogger()
	})
	ioc.RegisterScoped(dc, func(_ *ioc.DependencyProvider) password.Validator {
		return password.NewValidator()
	})
}

func Caching(dc *ioc.DependencyCollection, mode config.CacheMode) {
	ioc.RegisterSingleton(dc, func(dp *ioc.DependencyProvider) keyValue.Store {
		switch mode {
		case config.CacheModeMemory:
			return keyValue.NewMemoryStore()

		case config.CacheModeRedis:
			return keyValue.NewRedisStore()

		default:
			panic("cache mode missing or not supported")
		}
	})
}
