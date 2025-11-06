package setup

import (
	"Keyline/internal/behaviours"
	"Keyline/internal/caching"
	"Keyline/internal/config"
	"Keyline/internal/middlewares"
	"Keyline/internal/password"
	"Keyline/internal/services"
	"Keyline/internal/services/audit"
	"Keyline/internal/services/claimsMapping"
	"Keyline/internal/services/keyValue"
	"github.com/The127/ioc"
)

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
