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
Mediator.Send(Command/Query)
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
type CreateUserCommand struct {
    VirtualServerID uuid.UUID
    Username        string
    Email           string
    Password        string
}

// 2. Define the result (the response)
type CreateUserResult struct {
    UserID   uuid.UUID
    Username string
}

// 3. Define the handler
type CreateUserHandler struct {
    userRepo    repositories.UserRepository
    emailService services.EmailService
    mediator    mediator.Mediator
}

// 4. Implement the handler
func (h *CreateUserHandler) Handle(
    ctx context.Context, 
    cmd CreateUserCommand,
) (CreateUserResult, error) {
    // Validate business rules
    if cmd.Username == "" {
        return CreateUserResult{}, errors.New("username required")
    }
    
    // Execute business logic
    user, err := h.userRepo.Create(ctx, &models.User{
        VirtualServerID: cmd.VirtualServerID,
        Username:        cmd.Username,
        Email:           cmd.Email,
        // Hash password...
    })
    if err != nil {
        return CreateUserResult{}, err
    }
    
    // Emit domain event (fire and forget)
    _ = mediator.SendEvent(ctx, h.mediator, UserCreatedEvent{
        UserID:   user.ID,
        Username: user.Username,
        Email:    user.Email,
    })
    
    // Return result
    return CreateUserResult{
        UserID:   user.ID,
        Username: user.Username,
    }, nil
}
```

### Command Registration

Commands are registered during application startup in `internal/setup/setup.go`:

```go
func Commands(dc *ioc.DependencyCollection) {
    // Register command handler with mediator
    ioc.RegisterSingleton(dc, func(dp *ioc.DependencyProvider) any {
        m := ioc.GetDependency[mediator.Mediator](dp)
        userRepo := ioc.GetDependency[repositories.UserRepository](dp)
        emailService := ioc.GetDependency[services.EmailService](dp)
        
        handler := &commands.CreateUserHandler{
            userRepo:     userRepo,
            emailService: emailService,
            mediator:     m,
        }
        
        mediator.RegisterHandler(m, handler.Handle)
        return handler
    })
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
type GetUserQuery struct {
    UserID uuid.UUID
}

// 2. Define the result (the response)
type GetUserResult struct {
    UserID          uuid.UUID
    Username        string
    Email           string
    DisplayName     string
    EmailVerified   bool
    CreatedAt       time.Time
    // ... more fields as needed
}

// 3. Define the handler
type GetUserHandler struct {
    userRepo repositories.UserRepository
}

// 4. Implement the handler
func (h *GetUserHandler) Handle(
    ctx context.Context,
    query GetUserQuery,
) (GetUserResult, error) {
    // Retrieve data
    user, err := h.userRepo.GetByID(ctx, query.UserID)
    if err != nil {
        return GetUserResult{}, err
    }
    
    // Map to result DTO
    return GetUserResult{
        UserID:        user.ID,
        Username:      user.Username,
        Email:         user.Email,
        DisplayName:   user.DisplayName,
        EmailVerified: user.EmailVerified,
        CreatedAt:     user.CreatedAt,
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
    next mediator.Next,
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
        m := ioc.GetDependency[mediator.Mediator](dp)
        userRepo := ioc.GetDependency[repositories.UserRepository](dp)
        
        behavior := &behaviours.PolicyMiddleware{
            userRepo: userRepo,
        }
        
        // Register behavior - it applies to all requests
        mediator.RegisterBehaviour(m, behavior.Handle)
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
    UserID   uuid.UUID
    Username string
    Email    string
}

// 2. Define event handlers
type SendWelcomeEmailHandler struct {
    emailService services.EmailService
}

func (h *SendWelcomeEmailHandler) Handle(
    ctx context.Context,
    evt UserCreatedEvent,
) error {
    // Send welcome email
    return h.emailService.SendWelcomeEmail(evt.Email, evt.Username)
}

type CreateUserProfileHandler struct {
    profileRepo repositories.ProfileRepository
}

func (h *CreateUserProfileHandler) Handle(
    ctx context.Context,
    evt UserCreatedEvent,
) error {
    // Create default user profile
    return h.profileRepo.CreateDefault(ctx, evt.UserID)
}
```

### Emitting Events

```go
func (h *CreateUserHandler) Handle(
    ctx context.Context,
    cmd CreateUserCommand,
) (CreateUserResult, error) {
    // Create user...
    user, err := h.userRepo.Create(ctx, ...)
    if err != nil {
        return CreateUserResult{}, err
    }
    
    // Emit event (fire and forget - errors are logged but don't fail the command)
    _ = mediator.SendEvent(ctx, h.mediator, UserCreatedEvent{
        UserID:   user.ID,
        Username: user.Username,
        Email:    user.Email,
    })
    
    return CreateUserResult{UserID: user.ID}, nil
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
// In Handler
func (h *UserHandlers) RegisterUser(w http.ResponseWriter, r *http.Request) {
    var dto RegisterUserDTO
    json.NewDecoder(r.Body).Decode(&dto)
    
    // Send command via mediator
    result, err := mediator.Send[commands.RegisterUserResult](
        r.Context(),
        h.mediator,
        commands.RegisterUserCommand{
            VirtualServerID: h.getVirtualServerID(r),
            Username:        dto.Username,
            Email:           dto.Email,
            Password:        dto.Password,
        },
    )
    if err != nil {
        // Handle error...
        return
    }
    
    w.WriteHeader(http.StatusCreated)
    json.NewEncoder(w).Encode(result)
}

// In Command Handler
func (h *RegisterUserHandler) Handle(
    ctx context.Context,
    cmd commands.RegisterUserCommand,
) (commands.RegisterUserResult, error) {
    // 1. Check if registration is enabled
    vs, _ := h.vsRepo.GetByID(ctx, cmd.VirtualServerID)
    if !vs.EnableRegistration {
        return commands.RegisterUserResult{}, errors.New("registration disabled")
    }
    
    // 2. Validate username is available
    existing, _ := h.userRepo.GetByUsername(ctx, cmd.VirtualServerID, cmd.Username)
    if existing != nil {
        return commands.RegisterUserResult{}, errors.New("username taken")
    }
    
    // 3. Hash password
    hashedPassword, _ := h.passwordService.Hash(cmd.Password)
    
    // 4. Create user
    user, err := h.userRepo.Create(ctx, &models.User{
        VirtualServerID: cmd.VirtualServerID,
        Username:        cmd.Username,
        Email:           cmd.Email,
        PasswordHash:    hashedPassword,
        EmailVerified:   false,
    })
    if err != nil {
        return commands.RegisterUserResult{}, err
    }
    
    // 5. Emit events
    _ = mediator.SendEvent(ctx, h.mediator, UserCreatedEvent{
        UserID:   user.ID,
        Username: user.Username,
        Email:    user.Email,
    })
    
    return commands.RegisterUserResult{UserID: user.ID}, nil
}

// Event Handler 1: Send verification email
func (h *SendVerificationEmailHandler) Handle(
    ctx context.Context,
    evt UserCreatedEvent,
) error {
    // Generate verification token
    token := h.tokenService.GenerateVerificationToken(evt.UserID)
    
    // Send email
    return h.emailService.SendVerificationEmail(evt.Email, token)
}

// Event Handler 2: Create audit log
func (h *AuditUserCreatedHandler) Handle(
    ctx context.Context,
    evt UserCreatedEvent,
) error {
    return h.auditService.Log(ctx, audit.UserCreated, evt.UserID)
}
```

### Example 2: Query with Authorization

```go
// Query
type GetApplicationQuery struct {
    ApplicationID uuid.UUID
}

// Handler
func (h *GetApplicationHandler) Handle(
    ctx context.Context,
    query GetApplicationQuery,
) (GetApplicationResult, error) {
    // Get application
    app, err := h.appRepo.GetByID(ctx, query.ApplicationID)
    if err != nil {
        return GetApplicationResult{}, err
    }
    
    // Check authorization (behavior already checked, but double-check if needed)
    userID := authentication.GetUserID(ctx)
    hasAccess := h.checkAccess(ctx, userID, app)
    if !hasAccess {
        return GetApplicationResult{}, errors.New("forbidden")
    }
    
    // Map to result
    return GetApplicationResult{
        ApplicationID: app.ID,
        Name:         app.Name,
        ClientID:     app.ClientID,
        // ... other fields
    }, nil
}
```

### Example 3: Command with Transaction

```go
func (h *AssignRoleHandler) Handle(
    ctx context.Context,
    cmd AssignRoleCommand,
) (AssignRoleResult, error) {
    // Start transaction
    tx, _ := h.db.BeginTx(ctx, nil)
    defer tx.Rollback()
    
    // 1. Verify role exists
    role, err := h.roleRepo.GetByID(ctx, cmd.RoleID)
    if err != nil {
        return AssignRoleResult{}, err
    }
    
    // 2. Create assignment
    err = h.assignmentRepo.Create(ctx, &models.RoleAssignment{
        UserID: cmd.UserID,
        RoleID: cmd.RoleID,
    })
    if err != nil {
        return AssignRoleResult{}, err
    }
    
    // 3. Invalidate user permissions cache
    h.cacheService.InvalidateUserPermissions(cmd.UserID)
    
    // Commit transaction
    tx.Commit()
    
    // Emit event (after commit)
    _ = mediator.SendEvent(ctx, h.mediator, RoleAssignedEvent{
        UserID: cmd.UserID,
        RoleID: cmd.RoleID,
    })
    
    return AssignRoleResult{Success: true}, nil
}
```

## Testing Commands and Queries

Commands and queries are easy to test in isolation:

```go
func TestCreateUser(t *testing.T) {
    // Setup
    mockRepo := &MockUserRepository{}
    handler := &CreateUserHandler{
        userRepo: mockRepo,
    }
    
    // Execute
    result, err := handler.Handle(context.Background(), CreateUserCommand{
        Username: "testuser",
        Email:    "test@example.com",
        Password: "password123",
    })
    
    // Assert
    assert.NoError(t, err)
    assert.NotEqual(t, uuid.Nil, result.UserID)
    assert.Equal(t, 1, mockRepo.CreateCallCount)
}
```

## Next Steps

Now that you understand CQRS and the Mediator pattern:

1. **Learn about dependency injection** → [Dependency Injection with IoC](03-dependency-injection.md)
2. **See more examples** → [Common Patterns and Examples](05-common-patterns.md)
3. **Start building** → [Development Workflow](04-development-workflow.md)

## Additional Resources

- [Mediator Package Documentation](../../mediator/README.md) - Deep dive into the mediator
- [CQRS by Martin Fowler](https://martinfowler.com/bliki/CQRS.html)
- [Mediator Pattern](https://refactoring.guru/design-patterns/mediator)
