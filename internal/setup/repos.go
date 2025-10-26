package setup

import (
	"Keyline/internal/config"
	"Keyline/internal/database"
	"Keyline/internal/repositories"
	"Keyline/internal/repositories/postgres"
	"Keyline/ioc"
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
	ioc.RegisterScoped(dc, func(_ *ioc.DependencyProvider) repositories.ProjectRepository {
		return postgres.NewProjectRepository()
	})
	ioc.RegisterScoped(dc, func(_ *ioc.DependencyProvider) repositories.ResourceServerRepository {
		return postgres.NewResourceServerRepository()
	})
	ioc.RegisterScoped(dc, func(_ *ioc.DependencyProvider) repositories.ResourceServerScopeRepository {
		return postgres.NewResourceServerScopeRepository()
	})
}
