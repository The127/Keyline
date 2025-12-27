# CQRS and Mediator Pattern

This guide explains how Keyline uses CQRS (Command Query Responsibility Segregation) and the Mediator pattern to create a clean, maintainable architecture.

## Table of Contents
- [What is CQRS?](#what-is-cqrs)
- [What is the Mediator Pattern?](#what-is-the-mediator-pattern)
- [How They Work Together](#how-they-work-together)
- [Commands in Detail](#commands-in-detail)
- [Queries in Detail](#queries-in-detail)
- [Behaviors (Middleware)](#behaviors-middleware)
- [Events](#events)
- [Real-World Examples](#real-world-examples)

## What is CQRS?

CQRS stands for **Command Query Responsibility Segregation**. It's a pattern that separates read and write operations into different models.

### The Core Principle

> Commands change state. Queries return data. Never both.

```
┌─────────────────────────────────────────────────┐
│                Application                       │
├─────────────────────────────────────────────────┤
│                                                  │
│  Commands (Write)      Queries (Read)           │
│  ├─ CreateUser         ├─ GetUser               │
│  ├─ UpdateUser         ├─ ListUsers             │
│  ├─ DeleteUser         └─ SearchUsers           │
│  └─ ...                                          │
│      │                      │                    │
│      ▼                      ▼                    │
│  [Write Model]         [Read Model]             │
│                                                  │
└─────────────────────────────────────────────────┘
```

### Why CQRS?

**Benefits**:

1. **Clarity**: It's obvious whether code changes state or just reads it
2. **Optimization**: Read and write operations can be optimized independently
3. **Scalability**: Can scale reads and writes separately
4. **Security**: Easier to control who can read vs. write
5. **Maintainability**: Changes to reads don't affect writes and vice versa

**Example in Keyline**:
```
Command: CreateUser - Writes to database, may send emails, returns minimal data
Query: GetUser - Only reads, no side effects, returns complete user data
```

## What is the Mediator Pattern?

The Mediator pattern defines an object that coordinates communication between components, reducing direct dependencies.

### Without Mediator
```
Controller ──────> UserService
    │                  │
    └──────> EmailService
    │                  │
    └──────> AuditService
```
**Problem**: Controller needs to know about all services

### With Mediator
```
Controller ──> Mediator ──> UserHandler
                  │            │
                  ├────> EmailHandler
                  │            │
                  └────> AuditHandler
```
**Benefit**: Controller only knows about the mediator

### Why Mediator?

**Benefits**:

1. **Decoupling**: Components don't need to know about each other
2. **Single Responsibility**: Each handler does one thing
3. **Cross-Cutting Concerns**: Add validation, logging, auth in one place
4. **Testability**: Test handlers in isolation
5. **Flexibility**: Easy to add/remove/change handlers

## How They Work Together

In Keyline, CQRS and Mediator combine to create a powerful architecture:

```
HTTP Request
     ↓
Handler (thin)
     ↓
mediatr.Send(Command/Query)
     ↓
Behavior Pipeline (validation, auth, logging)
     ↓
Command/Query Handler (business logic)
     ↓
Repository (data access)
     ↓
Response flows back
```

## Commands in Detail

Commands represent **intentions to change state**.

### Command Structure

Every command in Keyline follows this pattern:

```go
// 1. Define the command (the request)
type CreateUser struct {
    VirtualServerName string
    Username          string
    Email             string
    Password          string
}

// 2. Define the response
type CreateUserResponse struct {
    UserID uuid.UUID
}

// 3. Implement the handler function
func HandleCreateUser(
    ctx context.Context,
    cmd CreateUser,
) (*CreateUserResponse, error) {
    // Get scope from context
    scope := middlewares.GetScope(ctx)
    
    // Resolve dependencies
    userRepo := ioc.GetDependency[repositories.UserRepository](scope)
    m := ioc.GetDependency[mediatr.Mediator](scope)
    
    // Validate business rules
    if cmd.Username == "" {
        return nil, errors.New("username required")
    }
    
    // Execute business logic
    user := repositories.NewUser(
        cmd.Username,
        cmd.Username, // displayName
        cmd.Email,
        virtualServerID,
    )
    
    err := userRepo.Insert(ctx, user)
    if err != nil {
        return nil, fmt.Errorf("inserting user: %w", err)
    }
    
    // Emit domain event (fire and forget)
    _ = mediatr.SendEvent(ctx, m, events.UserCreatedEvent{
        UserID: user.Id(),
    })
    
    // Return result
    return &CreateUserResponse{
        UserID: user.Id(),
    }, nil
}
```

### Command Registration

Commands are registered during application startup in `internal/setup/setup.go`:

```go
func Commands(m mediatr.Mediator) {
    // Register handler function directly with mediator
    mediatr.RegisterHandler(m, commands.HandleCreateUser)
    mediatr.RegisterHandler(m, commands.HandleUpdateUser)
    mediatr.RegisterHandler(m, commands.HandleDeleteUser)
    // ... more commands
}
```

### Command Best Practices

1. **Name with imperative verbs**: `CreateUser`, `UpdateUser`, `DeleteApplication`
2. **Include all required data**: Commands should be self-contained
3. **Return minimal data**: Usually just an ID or confirmation
4. **One handler per command**: Each command has exactly one handler
5. **Emit events for side effects**: Don't call other handlers directly

## Queries in Detail

Queries represent **requests for information** without side effects.

### Query Structure

```go
// 1. Define the query (the request)
type GetUser struct {
    UserID uuid.UUID
}

// 2. Define the response
type GetUserResponse struct {
    UserID        uuid.UUID
    Username      string
    Email         string
    DisplayName   string
    EmailVerified bool
    // ... more fields as needed
}

// 3. Implement the handler function
func HandleGetUser(
    ctx context.Context,
    query GetUser,
) (*GetUserResponse, error) {
    // Get scope from context
    scope := middlewares.GetScope(ctx)
    
    // Resolve dependencies
    userRepo := ioc.GetDependency[repositories.UserRepository](scope)
    
    // Retrieve data
    filter := repositories.NewUserFilter().Id(query.UserID)
    user, err := userRepo.Single(ctx, filter)
    if err != nil {
        return nil, fmt.Errorf("getting user: %w", err)
    }
    
    // Map to response
    return &GetUserResponse{
        UserID:        user.Id(),
        Username:      user.Username(),
        Email:         user.PrimaryEmail(),
        DisplayName:   user.DisplayName(),
        EmailVerified: user.EmailVerified(),
    }, nil
}
```

### Query Best Practices

1. **Name with questions**: `GetUser`, `ListUsers`, `FindApplications`
2. **No side effects**: Queries should never modify state
3. **Return complete data**: Include all data the caller needs
4. **Optimize for reads**: Use database indexes, caching as appropriate
5. **DTOs for results**: Don't return internal models directly

## Behaviors (Middleware)

Behaviors are like middleware for commands and queries. They wrap around handlers to add cross-cutting concerns.

### How Behaviors Work

```
Request
  ↓
Behavior 1 (before)
  ↓
Behavior 2 (before)
  ↓
Handler
  ↓
Behavior 2 (after)
  ↓
Behavior 1 (after)
  ↓
Response
```

### Example: Authorization Behavior

```go
// PolicyMiddleware checks if user has permission
type PolicyMiddleware struct {
    userRepo repositories.UserRepository
}

func (p *PolicyMiddleware) Handle(
    ctx context.Context,
    request any,
    next mediatr.Next,
) (any, error) {
    // Before handler executes
    
    // Check if user is authenticated
    userID := authentication.GetUserID(ctx)
    if userID == uuid.Nil {
        return nil, errors.New("unauthorized")
    }
    
    // Check permissions based on command type
    if cmd, ok := request.(commands.CreateApplicationCommand); ok {
        hasPermission := p.checkPermission(ctx, userID, "applications:create")
        if !hasPermission {
            return nil, errors.New("forbidden")
        }
    }
    
    // Call next behavior or handler
    result, err := next()
    
    // After handler executes (can log, modify result, etc.)
    return result, err
}
```

### Registering Behaviors

```go
func Behaviours(dc *ioc.DependencyCollection) {
    ioc.RegisterSingleton(dc, func(dp *ioc.DependencyProvider) any {
        m := ioc.GetDependency[mediatr.Mediator](dp)
        userRepo := ioc.GetDependency[repositories.UserRepository](dp)
        
        behavior := &behaviours.PolicyMiddleware{
            userRepo: userRepo,
        }
        
        // Register behavior - it applies to all requests
        mediatr.RegisterBehaviour(m, behavior.Handle)
        return behavior
    })
}
```

### Common Behaviors

1. **Validation**: Validate commands before processing
2. **Authorization**: Check user permissions
3. **Logging**: Log request/response
4. **Transactions**: Wrap handlers in database transactions
5. **Caching**: Cache query results
6. **Rate Limiting**: Prevent abuse

## Events

Events represent **notifications that something has happened**. Unlike commands and queries, events:

- Can have multiple handlers
- Are processed sequentially
- Don't return values (except errors)
- Are fire-and-forget from the sender's perspective

### Event Structure

```go
// 1. Define the event
type UserCreatedEvent struct {
    UserID uuid.UUID
}

// 2. Implement event handler functions
func QueueEmailVerificationJobOnUserCreatedEvent(
    ctx context.Context,
    evt UserCreatedEvent,
) error {
    scope := middlewares.GetScope(ctx)
    
    // Get dependencies
    userRepo := ioc.GetDependency[repositories.UserRepository](scope)
    emailService := ioc.GetDependency[services.EmailService](scope)
    
    // Get user
    user, err := userRepo.First(ctx, repositories.NewUserFilter().Id(evt.UserID))
    if err != nil {
        return fmt.Errorf("getting user: %w", err)
    }
    
    // Send welcome email
    return emailService.SendWelcomeEmail(user.PrimaryEmail(), user.Username())
}

func CreateAuditLogOnUserCreatedEvent(
    ctx context.Context,
    evt UserCreatedEvent,
) error {
    scope := middlewares.GetScope(ctx)
    auditRepo := ioc.GetDependency[repositories.AuditLogRepository](scope)
    
    // Create audit log
    return auditRepo.Insert(ctx, repositories.NewAuditLog(
        "user_created",
        evt.UserID,
    ))
}
```

### Emitting Events

```go
func HandleCreateUser(
    ctx context.Context,
    cmd CreateUser,
) (*CreateUserResponse, error) {
    scope := middlewares.GetScope(ctx)
    
    // Get dependencies
    userRepo := ioc.GetDependency[repositories.UserRepository](scope)
    m := ioc.GetDependency[mediatr.Mediator](scope)
    
    // Create user...
    user := repositories.NewUser(cmd.Username, cmd.DisplayName, cmd.Email, vsID)
    err := userRepo.Insert(ctx, user)
    if err != nil {
        return nil, fmt.Errorf("inserting user: %w", err)
    }
    
    // Emit event (fire and forget - errors are logged but don't fail the command)
    _ = mediatr.SendEvent(ctx, m, UserCreatedEvent{
        UserID: user.Id(),
    })
    
    return &CreateUserResponse{UserID: user.Id()}, nil
}
```

### Event Best Practices

1. **Past tense names**: `UserCreated`, `ApplicationDeleted`, `RoleAssigned`
2. **Immutable**: Events should not be modified after creation
3. **Complete data**: Include all relevant information
4. **No return values**: Event handlers don't return data to the caller
5. **Eventual consistency**: Events may be processed asynchronously

## Real-World Examples

### Example 1: User Registration Flow

```go
// In HTTP Handler
func (h *UserHandlers) RegisterUser(w http.ResponseWriter, r *http.Request) {
    var dto RegisterUserDTO
    json.NewDecoder(r.Body).Decode(&dto)
    
    scope := middlewares.GetScope(r.Context())
    m := ioc.GetDependency[mediatr.Mediator](scope)
    
    // Send command via mediator
    result, err := mediatr.Send[*commands.RegisterUserResponse](
        r.Context(),
        m,
        commands.RegisterUser{
            VirtualServerName: getVirtualServerName(r),
            Username:          dto.Username,
            Email:             dto.Email,
            Password:          dto.Password,
        },
    )
    if err != nil {
        // Handle error...
        return
    }
    
    w.WriteHeader(http.StatusCreated)
    json.NewEncoder(w).Encode(result)
}

// Command handler function
func HandleRegisterUser(
    ctx context.Context,
    cmd commands.RegisterUser,
) (*commands.RegisterUserResponse, error) {
    scope := middlewares.GetScope(ctx)
    
    vsRepo := ioc.GetDependency[repositories.VirtualServerRepository](scope)
    userRepo := ioc.GetDependency[repositories.UserRepository](scope)
    m := ioc.GetDependency[mediatr.Mediator](scope)
    
    // 1. Check if registration is enabled
    vs, err := vsRepo.First(ctx, repositories.NewVirtualServerFilter().Name(cmd.VirtualServerName))
    if err != nil {
        return nil, fmt.Errorf("getting virtual server: %w", err)
    }
    if !vs.EnableRegistration() {
        return nil, errors.New("registration disabled")
    }
    
    // 2. Validate username is available
    existing, _ := userRepo.First(ctx, repositories.NewUserFilter().
        VirtualServerId(vs.Id()).
        Username(cmd.Username))
    if existing != nil {
        return nil, errors.New("username taken")
    }
    
    // 3. Create user
    user := repositories.NewUser(cmd.Username, cmd.Username, cmd.Email, vs.Id())
    // Set password hash...
    
    err = userRepo.Insert(ctx, user)
    if err != nil {
        return nil, fmt.Errorf("inserting user: %w", err)
    }
    
    // 4. Emit events
    _ = mediatr.SendEvent(ctx, m, events.UserCreatedEvent{
        UserID: user.Id(),
    })
    
    return &commands.RegisterUserResponse{UserID: user.Id()}, nil
}

// Event handler function 1: Send verification email
func QueueEmailVerificationJobOnUserCreatedEvent(
    ctx context.Context,
    evt events.UserCreatedEvent,
) error {
    scope := middlewares.GetScope(ctx)
    
    userRepo := ioc.GetDependency[repositories.UserRepository](scope)
    tokenService := ioc.GetDependency[services.TokenService](scope)
    
    // Get user
    user, err := userRepo.First(ctx, repositories.NewUserFilter().Id(evt.UserID))
    if err != nil {
        return fmt.Errorf("getting user: %w", err)
    }
    
    // Generate verification token
    token, err := tokenService.GenerateAndStoreToken(ctx, 
        services.EmailVerificationTokenType, 
        user.Id().String(), 
        time.Minute*15)
    if err != nil {
        return fmt.Errorf("generating token: %w", err)
    }
    
    // Queue email sending job...
    return nil
}

// Event handler function 2: Create audit log
func CreateAuditLogOnUserCreatedEvent(
    ctx context.Context,
    evt events.UserCreatedEvent,
) error {
    scope := middlewares.GetScope(ctx)
    auditRepo := ioc.GetDependency[repositories.AuditLogRepository](scope)
    
    return auditRepo.Insert(ctx, repositories.NewAuditLog("user_created", evt.UserID))
}
```

### Example 2: Query with Authorization

```go
// Query request
type GetApplication struct {
    ApplicationID uuid.UUID
}

// Query response
type GetApplicationResponse struct {
    ApplicationID uuid.UUID
    Name          string
    ClientID      string
    // ... other fields
}

// Handler function
func HandleGetApplication(
    ctx context.Context,
    query GetApplication,
) (*GetApplicationResponse, error) {
    scope := middlewares.GetScope(ctx)
    
    appRepo := ioc.GetDependency[repositories.ApplicationRepository](scope)
    
    // Get application
    filter := repositories.NewApplicationFilter().Id(query.ApplicationID)
    app, err := appRepo.First(ctx, filter)
    if err != nil {
        return nil, fmt.Errorf("getting application: %w", err)
    }
    
    // Authorization is handled by PolicyBehaviour automatically
    // based on the query's IsAllowed method
    
    // Map to response
    return &GetApplicationResponse{
        ApplicationID: app.Id(),
        Name:          app.Name(),
        ClientID:      app.ClientId(),
        // ... other fields
    }, nil
}
```

### Example 3: Command with Transaction

```go
func HandleAssignRole(
    ctx context.Context,
    cmd AssignRole,
) (*AssignRoleResponse, error) {
    scope := middlewares.GetScope(ctx)
    
    db := ioc.GetDependency[*sql.DB](scope)
    roleRepo := ioc.GetDependency[repositories.RoleRepository](scope)
    assignmentRepo := ioc.GetDependency[repositories.RoleAssignmentRepository](scope)
    m := ioc.GetDependency[mediatr.Mediator](scope)
    
    // Start transaction
    tx, err := db.BeginTx(ctx, nil)
    if err != nil {
        return nil, fmt.Errorf("starting transaction: %w", err)
    }
    defer tx.Rollback() // Rollback if not committed
    
    // Create context with transaction
    txCtx := context.WithValue(ctx, "tx", tx)
    
    // 1. Verify role exists
    role, err := roleRepo.First(txCtx, repositories.NewRoleFilter().Id(cmd.RoleID))
    if err != nil {
        return nil, fmt.Errorf("getting role: %w", err)
    }
    
    // 2. Create assignment
    assignment := repositories.NewRoleAssignment(cmd.UserID, cmd.RoleID)
    err = assignmentRepo.Insert(txCtx, assignment)
    if err != nil {
        return nil, fmt.Errorf("creating assignment: %w", err)
    }
    
    // 3. Commit transaction
    if err := tx.Commit(); err != nil {
        return nil, fmt.Errorf("committing transaction: %w", err)
    }
    
    // 4. Emit event (after commit)
    _ = mediatr.SendEvent(ctx, m, events.RoleAssignedEvent{
        UserID: cmd.UserID,
        RoleID: cmd.RoleID,
    })
    
    return &AssignRoleResponse{Success: true}, nil
}
```

## Testing Commands and Queries

Commands and queries are easy to test in isolation:

```go
func TestCreateUser(t *testing.T) {
    // Note: Handler functions get dependencies from context/scope
    // In tests, you can mock the entire mediator or test the handler
    // function directly with a test scope containing mock repositories
    
    // Create gomock controller
    ctrl := gomock.NewController(t)
    defer ctrl.Finish()
    
    // Create test dependencies
    mockUserRepo := mocks.NewMockUserRepository(ctrl)
    mockUserRepo.EXPECT().Insert( gomock.Any())
    
    // For unit testing handler functions, you would:
    // 1. Create a test IoC scope with mocked dependencies
    // 2. Pass context with that scope
    // 3. Test the handler function directly
    
    // Example:
    dc := ioc.NewDependencyCollection()
    ioc.RegisterTransient(dc, func(_ *ioc.DependencyProvider) repositories.UserRepository {
        return mockUserRepo
    })
    scope := dc.BuildProvider()
    defer scope.Close()
    
    ctx := middlewares.ContextWithScope(context.Background(), scope)
    
    result, err := HandleCreateUser(ctx, CreateUser{
        Username: "testuser",
        Email:    "test@example.com",
    })
    
    assert.NoError(t, err)
    assert.NotNil(t, result)
}
```

## Next Steps

Now that you understand CQRS and the Mediator pattern:

1. **Learn about dependency injection** → [Dependency Injection with IoC](03-dependency-injection.md)
2. **See more examples** → [Common Patterns and Examples](05-common-patterns.md)
3. **Start building** → [Development Workflow](04-development-workflow.md)

## Additional Resources

- [CQRS by Martin Fowler](https://martinfowler.com/bliki/CQRS.html)
- [Mediator Pattern](https://refactoring.guru/design-patterns/mediator)
