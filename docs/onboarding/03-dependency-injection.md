# Dependency Injection with IoC Container

This guide explains how Keyline uses its custom IoC (Inversion of Control) container for dependency injection, making the application flexible, testable, and maintainable.

## Table of Contents
- [What is Dependency Injection?](#what-is-dependency-injection)
- [What is an IoC Container?](#what-is-an-ioc-container)
- [Service Lifetimes](#service-lifetimes)
- [How to Register Dependencies](#how-to-register-dependencies)
- [How to Resolve Dependencies](#how-to-resolve-dependencies)
- [Scopes and Request Handling](#scopes-and-request-handling)
- [Real-World Examples](#real-world-examples)
- [Best Practices](#best-practices)

## What is Dependency Injection?

Dependency Injection (DI) is a design pattern where objects receive their dependencies from external sources rather than creating them internally.

### Without DI (Bad)
```go
type UserService struct {
    repo UserRepository
}

func NewUserService() *UserService {
    // Creating dependencies internally - tightly coupled
    db := sql.Open("postgres", "...")
    repo := NewUserRepository(db)
    
    return &UserService{
        repo: repo,
    }
}
```

**Problems**:
- Hard to test (can't mock the database)
- Tight coupling to concrete implementations
- Hard to change implementations
- Creates dependencies it doesn't own

### With DI (Good)
```go
type UserService struct {
    repo UserRepository
}

func NewUserService(repo UserRepository) *UserService {
    // Dependencies are injected - loosely coupled
    return &UserService{
        repo: repo,
    }
}
```

**Benefits**:
- Easy to test (inject mocks)
- Depends on interfaces, not implementations
- Easy to swap implementations
- Clear dependencies

## What is an IoC Container?

An **IoC (Inversion of Control) Container** automates dependency injection. Instead of manually creating and passing dependencies, the container:

1. Knows how to create all objects
2. Manages object lifetimes
3. Automatically resolves dependencies
4. Handles cleanup

### The Keyline IoC Container

Keyline uses a custom IoC container with Go generics for type safety.

**Key Features**:
- Type-safe registration and resolution
- Three lifetime types: Transient, Scoped, Singleton
- Automatic dependency graph resolution
- Scope hierarchy for request handling
- Resource cleanup with close handlers

## Service Lifetimes

The container supports three lifetime types:

### 1. Transient

**A new instance every time.**

```go
RegisterTransient(dc, func(dp *DependencyProvider) EmailSender {
    return &SMTPEmailSender{
        host: "smtp.example.com",
    }
})
```

**Use for**:
- Lightweight objects
- Stateless services that are cheap to create
- Objects that should not be shared

**Example**: Email senders, password hashers, validators

### 2. Scoped

**One instance per scope (typically per HTTP request).**

```go
RegisterScoped(dc, func(dp *DependencyProvider) *RequestContext {
    return &RequestContext{
        RequestID: uuid.New(),
        StartTime: time.Now(),
    }
})
```

**Use for**:
- Per-request state
- Request-specific caching
- Database transactions
- Request logging context

**Example**: HTTP request context, database transactions, request-scoped caches

### 3. Singleton

**One instance for the entire application lifetime.**

```go
RegisterSingleton(dc, func(dp *DependencyProvider) *Database {
    return &Database{
        conn: sql.Open(...),
    }
})
```

**Use for**:
- Expensive to create objects
- Thread-safe shared state
- Connection pools
- Configuration

**Example**: Database connections, mediator, repositories, configuration

## How to Register Dependencies

Dependencies are registered during application startup in `internal/setup/setup.go`.

### Basic Registration

```go
func SetupServices(dc *ioc.DependencyCollection) {
    // Singleton: Database connection
    ioc.RegisterSingleton(dc, func(dp *ioc.DependencyProvider) *sql.DB {
        config := ioc.GetDependency[*config.Config](dp)
        db, err := sql.Open("postgres", config.Database.ConnectionString())
        if err != nil {
            panic(err)
        }
        return db
    })
    
    // Singleton: User repository
    ioc.RegisterSingleton(dc, func(dp *ioc.DependencyProvider) repositories.UserRepository {
        db := ioc.GetDependency[*sql.DB](dp)
        return postgres.NewUserRepository(db)
    })
    
    // Singleton: User service
    ioc.RegisterSingleton(dc, func(dp *ioc.DependencyProvider) *services.UserService {
        repo := ioc.GetDependency[repositories.UserRepository](dp)
        return services.NewUserService(repo)
    })
}
```

### Registration with Multiple Dependencies

```go
ioc.RegisterSingleton(dc, func(dp *ioc.DependencyProvider) *TokenService {
    config := ioc.GetDependency[*config.Config](dp)
    keyService := ioc.GetDependency[*KeyService](dp)
    auditService := ioc.GetDependency[*AuditService](dp)
    
    return &TokenService{
        config:       config,
        keyService:   keyService,
        auditService: auditService,
    }
})
```

### Registration Order

The IoC container automatically resolves dependencies in the correct order. You can register in any order:

```go
// This works even though UserService is registered before UserRepository
ioc.RegisterSingleton(dc, func(dp *ioc.DependencyProvider) *UserService {
    repo := ioc.GetDependency[repositories.UserRepository](dp) // Will be resolved
    return NewUserService(repo)
})

ioc.RegisterSingleton(dc, func(dp *ioc.DependencyProvider) repositories.UserRepository {
    db := ioc.GetDependency[*sql.DB](dp)
    return postgres.NewUserRepository(db)
})
```

## How to Resolve Dependencies

### During Startup (Building the Provider)

```go
func main() {
    // 1. Create dependency collection
    dc := ioc.NewDependencyCollection()
    
    // 2. Register all dependencies
    setup.RegisterDatabase(dc)
    setup.RegisterRepositories(dc)
    setup.RegisterServices(dc)
    setup.RegisterHandlers(dc)
    
    // 3. Build the provider (resolves singletons)
    provider := dc.BuildProvider()
    
    // 4. Get singleton dependencies
    mediator := ioc.GetDependency[mediatr.Mediator](provider)
    router := ioc.GetDependency[*mux.Router](provider)
    
    // 5. Start server
    http.ListenAndServe(":8080", router)
}
```

### During Request Handling (Creating Scopes)

```go
func RequestScopeMiddleware(provider *ioc.DependencyProvider) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            // Create a new scope for this request
            scope := provider.NewScope()
            defer scope.Close() // Clean up scoped resources
            
            // Store scope in request context
            ctx := context.WithValue(r.Context(), "scope", scope)
            
            // Continue with scoped context
            next.ServeHTTP(w, r.WithContext(ctx))
        })
    }
}
```

### In Handlers

```go
func GetUserHandler(w http.ResponseWriter, r *http.Request) {
    // Get scope from context
    scope := r.Context().Value("scope").(*ioc.DependencyProvider)
    
    // Resolve scoped or singleton dependencies
    mediator := ioc.GetDependency[mediatr.Mediator](scope)
    
    // Use the dependency
    result, err := mediatr.Send[GetUserResult](r.Context(), GetUserQuery{...})
    // ...
}
```

## Scopes and Request Handling

Scopes enable per-request dependency management. This is crucial for:

### Request Isolation

Each HTTP request gets its own scope:

```
Request 1 → Scope 1 → Scoped instances for Request 1
Request 2 → Scope 2 → Scoped instances for Request 2
Request 3 → Scope 3 → Scoped instances for Request 3
```

### Scope Hierarchy

```
Root Provider (Singletons)
    │
    ├─ Scope 1 (Request 1)
    │   └─ Scoped instances for Request 1
    │
    ├─ Scope 2 (Request 2)
    │   └─ Scoped instances for Request 2
    │
    └─ Scope 3 (Request 3)
        └─ Scoped instances for Request 3
```

### Resource Cleanup

Scoped dependencies are automatically cleaned up when the scope closes:

```go
// Register a scoped database transaction
ioc.RegisterScoped(dc, func(dp *ioc.DependencyProvider) *sql.Tx {
    db := ioc.GetDependency[*sql.DB](dp)
    tx, _ := db.Begin()
    return tx
})

// Register cleanup handler
ioc.RegisterCloseHandler(dc, func(tx *sql.Tx) error {
    // Rollback if not committed
    return tx.Rollback()
})

// Usage in request
scope := provider.NewScope()
tx := ioc.GetDependency[*sql.Tx](scope)
// ... use transaction
tx.Commit()
scope.Close() // Cleanup called here
```

## Real-World Examples

### Example 1: Complete Setup in main.go

```go
package main

import (
	"Keyline/internal/setup"

	"github.com/The127/ioc"
)

func main() {
	// Create dependency collection
	dc := ioc.NewDependencyCollection()

	// Register configuration (singleton)
	setup.Configuration(dc)

	// Register database (singleton)
	setup.Database(dc)

	// Register repositories (singleton)
	setup.Repositories(dc, config.DatabaseModePostgres, postgresConfig)

	// Register services (singleton)
	setup.Services(dc)

	// Register mediator (singleton)
	setup.Mediator(dc)

	// Register command handlers (singleton)
	setup.Commands(dc)

	// Register query handlers (singleton)
	setup.Queries(dc)

	// Register behaviors (singleton)
	setup.Behaviours(dc)

	// Register event handlers (singleton)
	setup.Events(dc)

	// Build provider
	provider := dc.BuildProvider()

	// Start server
	router := ioc.GetDependency[*mux.Router](provider)
	http.ListenAndServe(":8080", router)
}
```

### Example 2: Registering Repositories

From `internal/setup/setup.go`:

```go
func Repositories(dc *ioc.DependencyCollection, mode config.DatabaseMode, c any) {
    switch mode {
    case config.DatabaseModePostgres:
        pc := c.(config.PostgresConfig)
        
        // User repository
        ioc.RegisterSingleton(dc, func(dp *ioc.DependencyProvider) repositories.UserRepository {
            db := ioc.GetDependency[*sql.DB](dp)
            return postgres.NewUserRepository(db)
        })
        
        // Application repository
        ioc.RegisterSingleton(dc, func(dp *ioc.DependencyProvider) repositories.ApplicationRepository {
            db := ioc.GetDependency[*sql.DB](dp)
            return postgres.NewApplicationRepository(db)
        })
        
        // Role repository
        ioc.RegisterSingleton(dc, func(dp *ioc.DependencyProvider) repositories.RoleRepository {
            db := ioc.GetDependency[*sql.DB](dp)
            return postgres.NewRoleRepository(db)
        })
        
        // ... more repositories
    }
}
```

### Example 3: Registering Commands with Mediator

```go
func Commands(dc *ioc.DependencyCollection) {
    // CreateUser command
    ioc.RegisterSingleton(dc, func(dp *ioc.DependencyProvider) any {
        m := ioc.GetDependency[mediatr.Mediator](dp)
        userRepo := ioc.GetDependency[repositories.UserRepository](dp)
        emailService := ioc.GetDependency[services.EmailService](dp)
        
        handler := &commands.CreateUserHandler{
            userRepo:     userRepo,
            emailService: emailService,
            mediator:     m,
        }
        
        // Register with mediator
        mediatr.RegisterHandler(m, handler.Handle)
        
        return handler
    })
    
    // UpdateUser command
    ioc.RegisterSingleton(dc, func(dp *ioc.DependencyProvider) any {
        m := ioc.GetDependency[mediatr.Mediator](dp)
        userRepo := ioc.GetDependency[repositories.UserRepository](dp)
        
        handler := &commands.UpdateUserHandler{
            userRepo: userRepo,
        }
        
        mediatr.RegisterHandler(m, handler.Handle)
        return handler
    })
}
```

### Example 4: Scoped Request Context

```go
// Register scoped request context
ioc.RegisterScoped(dc, func(dp *ioc.DependencyProvider) *RequestContext {
    return &RequestContext{
        RequestID:   uuid.New(),
        StartTime:   time.Now(),
        UserID:      uuid.Nil, // Set by auth middleware
        VirtualServerID: uuid.Nil, // Set by middleware
    }
})

// Use in middleware
func AuthMiddleware(provider *ioc.DependencyProvider) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            scope := provider.NewScope()
            defer scope.Close()
            
            // Get scoped request context
            reqCtx := ioc.GetDependency[*RequestContext](scope)
            
            // Set user from token
            userID := extractUserFromToken(r)
            reqCtx.UserID = userID
            
            // Store scope in context
            ctx := context.WithValue(r.Context(), "scope", scope)
            next.ServeHTTP(w, r.WithContext(ctx))
        })
    }
}
```

## Best Practices

### 1. Register by Interface, Resolve by Interface

```go
// Good: Depend on interface
ioc.RegisterSingleton(dc, func(dp *ioc.DependencyProvider) repositories.UserRepository {
    db := ioc.GetDependency[*sql.DB](dp)
    return postgres.NewUserRepository(db) // Returns interface
})

// Usage
userRepo := ioc.GetDependency[repositories.UserRepository](dp)
```

### 2. Use Singletons for Stateless Services

```go
// Repositories are typically singleton
ioc.RegisterSingleton(dc, func(dp *ioc.DependencyProvider) repositories.UserRepository {
    // ...
})

// Services are typically singleton
ioc.RegisterSingleton(dc, func(dp *ioc.DependencyProvider) *TokenService {
    // ...
})
```

### 3. Use Scoped for Request-Specific State

```go
// Database transactions per request
ioc.RegisterScoped(dc, func(dp *ioc.DependencyProvider) *sql.Tx {
    db := ioc.GetDependency[*sql.DB](dp)
    tx, _ := db.Begin()
    return tx
})

// Request context per request
ioc.RegisterScoped(dc, func(dp *ioc.DependencyProvider) *RequestContext {
    return &RequestContext{}
})
```

### 4. Avoid Transient for Expensive Objects

```go
// Bad: Database connections are expensive
ioc.RegisterTransient(dc, func(dp *ioc.DependencyProvider) *sql.DB {
    return sql.Open(...) // Creates new connection every time!
})

// Good: Use singleton
ioc.RegisterSingleton(dc, func(dp *ioc.DependencyProvider) *sql.DB {
    return sql.Open(...) // Creates once, reuses
})
```

### 5. Use Close Handlers for Resource Cleanup

```go
// Register resource
ioc.RegisterScoped(dc, func(dp *ioc.DependencyProvider) *DatabaseConnection {
    return OpenConnection()
})

// Register cleanup
ioc.RegisterCloseHandler(dc, func(conn *DatabaseConnection) error {
    return conn.Close()
})
```

### 6. Keep Registration Logic in setup Package

```go
// Good: Organized in setup package
// internal/setup/repositories.go
func Repositories(dc *ioc.DependencyCollection) {
    // Register all repositories here
}

// internal/setup/services.go
func Services(dc *ioc.DependencyCollection) {
    // Register all services here
}

// Bad: Registration scattered across packages
```

### 7. Don't Create Circular Dependencies

```go
// Bad: A depends on B, B depends on A
ioc.RegisterSingleton(dc, func(dp *ioc.DependencyProvider) *ServiceA {
    b := ioc.GetDependency[*ServiceB](dp) // ServiceB depends on ServiceA!
    return &ServiceA{b: b}
})

// Good: Introduce an interface or mediator to break the cycle
```

## Testing with the IoC Container

### Unit Tests (Without IoC)

```go
func TestUserService(t *testing.T) {
    // Create mocks
    mockRepo := &MockUserRepository{}
    
    // Create service directly
    service := NewUserService(mockRepo)
    
    // Test
    user, err := service.GetUser("user123")
    assert.NoError(t, err)
}
```

### Integration Tests (With IoC)

```go
func TestUserServiceIntegration(t *testing.T) {
    // Setup IoC container with test dependencies
    dc := ioc.NewDependencyCollection()
    
    // Register test database
    ioc.RegisterSingleton(dc, func(dp *ioc.DependencyProvider) *sql.DB {
        return setupTestDatabase()
    })
    
    // Register real repositories
    setup.Repositories(dc, config.DatabaseModePostgres, testConfig)
    
    // Register services
    setup.Services(dc)
    
    // Build provider
    provider := dc.BuildProvider()
    
    // Test
    service := ioc.GetDependency[*UserService](provider)
    user, err := service.GetUser("user123")
    assert.NoError(t, err)
}
```

## Troubleshooting

### Dependency Not Registered

```
Error: dependency not registered for type *UserService
```

**Solution**: Register the dependency in `internal/setup/setup.go`:
```go
ioc.RegisterSingleton(dc, func(dp *ioc.DependencyProvider) *UserService {
    return NewUserService(...)
})
```

### Circular Dependency

```
Error: circular dependency detected
```

**Solution**: Introduce an interface or use the mediator pattern to break the cycle.

### Wrong Lifetime

```
Problem: Scoped dependency used in singleton
```

**Solution**: Singletons cannot depend on scoped or transient dependencies. Restructure to use correct lifetimes.

## Next Steps

Now that you understand dependency injection:

1. **Start developing** → [Development Workflow](04-development-workflow.md)
2. **See practical examples** → [Common Patterns and Examples](05-common-patterns.md)

## Additional Resources

- [Dependency Injection Principles](https://martinfowler.com/articles/injection.html)
- [SOLID Principles](https://en.wikipedia.org/wiki/SOLID)
