# Architecture Overview

This document provides a comprehensive overview of Keyline's architecture, helping you understand how the system is organized and how its components interact.

## Table of Contents
- [Clean Architecture Principles](#clean-architecture-principles)
- [Folder Structure](#folder-structure)
- [Layer Responsibilities](#layer-responsibilities)
- [Request Flow](#request-flow)
- [Component Interaction Diagram](#component-interaction-diagram)
- [Key Design Decisions](#key-design-decisions)

## Clean Architecture Principles

Keyline follows Clean Architecture (also known as Hexagonal Architecture or Ports and Adapters):

### Core Principles

1. **Independence**: Business logic doesn't depend on frameworks, databases, or UI
2. **Testability**: Components can be tested in isolation
3. **Maintainability**: Clear boundaries make changes easier
4. **Flexibility**: Swap implementations without affecting business logic

### The Dependency Rule

> Dependencies point inward toward business logic

```
┌─────────────────────────────────────────────────────────┐
│              External Systems (DB, HTTP)                │
└──────────────────────┬──────────────────────────────────┘
                       │ Depends on
                       ▼
┌─────────────────────────────────────────────────────────┐
│         Interface Adapters (Handlers, Repos)            │
└──────────────────────┬──────────────────────────────────┘
                       │ Depends on
                       ▼
┌─────────────────────────────────────────────────────────┐
│         Business Logic (Commands, Queries)              │
└──────────────────────┬──────────────────────────────────┘
                       │ Depends on
                       ▼
┌─────────────────────────────────────────────────────────┐
│              Core Domain Models                         │
└─────────────────────────────────────────────────────────┘
```

## Folder Structure

```
Keyline/
├── cmd/                          # Application entry points
│   ├── api/                      # HTTP API server
│   │   └── main.go              # API startup and configuration
│   └── queueWorker/             # Background job processor
│       └── main.go              # Worker startup
│
├── internal/                     # Private application code
│   ├── handlers/                # HTTP request handlers
│   │   ├── users.go             # User management endpoints
│   │   ├── applications.go      # Application management
│   │   ├── oidc.go              # OIDC endpoints
│   │   └── ...                  # Other domain handlers
│   │
│   ├── commands/                # Write operations (CQRS)
│   │   ├── CreateUser.go        # Command for user creation
│   │   ├── UpdateUser.go        # Command for updates
│   │   └── ...                  # Other commands
│   │
│   ├── queries/                 # Read operations (CQRS)
│   │   ├── GetUser.go           # Query for user retrieval
│   │   ├── ListUsers.go         # Query for listing
│   │   └── ...                  # Other queries
│   │
│   ├── repositories/            # Data access layer
│   │   ├── users.go             # User repository interface
│   │   ├── applications.go      # Application repository
│   │   └── postgres/            # PostgreSQL implementations
│   │       ├── users.go         # Concrete user repo
│   │       └── ...              # Other implementations
│   │
│   ├── services/                # Core business services
│   │   ├── tokens.go            # JWT token service
│   │   ├── keys.go              # Key management
│   │   ├── audit/               # Audit logging service
│   │   ├── claimsMapping/       # JWT claims transformation
│   │   └── ...                  # Other services
│   │
│   ├── middlewares/             # HTTP middleware
│   │   ├── authentication.go    # Auth middleware
│   │   └── ...                  # Other middleware
│   │
│   ├── behaviours/              # Mediator behaviors (cross-cutting)
│   │   └── PolicyMiddleware.go  # Authorization behavior
│   │
│   ├── events/                  # Domain event handlers
│   │   └── ...                  # Event implementations
│   │
│   ├── database/                # Database management
│   │   ├── migrations/          # SQL migration files
│   │   └── connection.go        # DB connection setup
│   │
│   ├── config/                  # Configuration management
│   │   └── config.go            # Config structures
│   │
│   ├── setup/                   # Dependency setup
│   │   └── setup.go             # IoC container registration
│   │
│   └── ...                      # Other infrastructure
│
├── mediator/                    # Mediator pattern implementation
│   ├── mediatr.go              # Core mediator logic
│   └── README.md                # Mediator documentation
│
├── ioc/                         # IoC container implementation
│   ├── container.go             # Core IoC logic
│   └── Readme.md                # IoC documentation
│
├── client/                      # API client library
│   └── client.go                # HTTP client for Keyline API
│
├── tests/                       # Test suites
│   ├── e2e/                     # End-to-end tests
│   └── integration/             # Integration tests
│
├── docs/                        # Documentation
│   ├── onboarding/              # This onboarding guide
│   └── ApplicationArchitecture.drawio  # Architecture diagram
│
└── templates/                   # Email templates
    └── ...                      # Email template files
```

## Layer Responsibilities

### 1. HTTP Layer (Handlers)

**Location**: `internal/handlers/`

**Responsibilities**:
- Handle HTTP requests and responses
- Parse and validate input
- Delegate to commands/queries via mediator
- Map domain results to HTTP responses
- Handle HTTP-specific concerns (status codes, headers)

**Example** (`internal/handlers/users.go`):
```go
func (h *UserHandlers) CreateUser(w http.ResponseWriter, r *http.Request) {
    // 1. Parse request
    var dto CreateUserDTO
    json.NewDecoder(r.Body).Decode(&dto)
    
    // 2. Delegate to command via mediator
    result, err := mediatr.Send[CreateUserResult](r.Context(), h.mediator, 
        CreateUserCommand{...})
    
    // 3. Return HTTP response
    w.WriteHeader(http.StatusCreated)
    json.NewEncoder(w).Encode(result)
}
```

**Key Points**:
- Handlers are thin - they don't contain business logic
- All business operations go through the mediator
- Handlers should not access repositories directly

### 2. Mediator Layer

**Location**: `mediator/`

**Responsibilities**:
- Route requests to appropriate handlers
- Execute behavior pipeline (validation, logging, auth)
- Provide decoupled communication between components

**Key Points**:
- Central hub for all commands, queries, and events
- Enables cross-cutting concerns via behaviors
- See [CQRS and Mediator Pattern](02-cqrs-and-mediatr.md) for details

### 3. Command Layer (Write Operations)

**Location**: `internal/commands/`

**Responsibilities**:
- Modify application state (create, update, delete)
- Enforce business rules
- Emit domain events
- Return operation results

**Example** (`internal/commands/CreateUser.go`):
```go
type CreateUserCommand struct {
    VirtualServerID uuid.UUID
    Username        string
    Email           string
    Password        string
}

type CreateUserResult struct {
    UserID uuid.UUID
}

func (h *CreateUserHandler) Handle(ctx context.Context, cmd CreateUserCommand) (CreateUserResult, error) {
    // 1. Validate business rules
    if cmd.Username == "" {
        return CreateUserResult{}, errors.New("username required")
    }
    
    // 2. Perform operation via repository
    user := h.userRepo.Create(ctx, ...)
    
    // 3. Emit domain event
    mediatr.SendEvent(ctx, h.mediator, UserCreatedEvent{...})
    
    // 4. Return result
    return CreateUserResult{UserID: user.ID}, nil
}
```

**Key Points**:
- Commands have side effects
- Commands should be named with imperative verbs (Create, Update, Delete)
- Each command has exactly one handler

### 4. Query Layer (Read Operations)

**Location**: `internal/queries/`

**Responsibilities**:
- Retrieve data from the system
- No side effects or state changes
- Return read-optimized DTOs

**Example** (`internal/queries/GetUser.go`):
```go
type GetUserQuery struct {
    UserID uuid.UUID
}

type GetUserResult struct {
    UserID   uuid.UUID
    Username string
    Email    string
    // ... other fields
}

func (h *GetUserHandler) Handle(ctx context.Context, query GetUserQuery) (GetUserResult, error) {
    // 1. Retrieve from repository
    user := h.userRepo.GetByID(ctx, query.UserID)
    
    // 2. Map to result DTO
    return GetUserResult{
        UserID:   user.ID,
        Username: user.Username,
        Email:    user.Email,
    }, nil
}
```

**Key Points**:
- Queries are read-only
- Queries should be named with question words (Get, List, Find)
- Queries can be optimized differently than commands

### 5. Repository Layer

**Location**: `internal/repositories/`

**Responsibilities**:
- Abstract data access
- Provide interface for data operations
- Hide database implementation details
- Execute SQL queries

**Example** (`internal/repositories/users.go`):
```go
type UserRepository interface {
    GetByID(ctx context.Context, id uuid.UUID) (*User, error)
    Create(ctx context.Context, user *User) error
    Update(ctx context.Context, user *User) error
    Delete(ctx context.Context, id uuid.UUID) error
}
```

**Key Points**:
- Repositories define interfaces, implementations are in subpackages
- Use `go-sqlbuilder` for query construction
- All queries are parameterized to prevent SQL injection

### 6. Service Layer

**Location**: `internal/services/`

**Responsibilities**:
- Core business services used across the application
- Token generation and validation
- Key management
- Email sending
- Claims mapping

**Examples**:
- `tokens.go`: JWT token creation and validation
- `keys.go`: Cryptographic key management
- `audit/`: Audit logging service
- `claimsMapping/`: JavaScript-based claims transformation

**Key Points**:
- Services are reusable across commands and queries
- Services are registered in the IoC container
- Services should have clear interfaces

## Request Flow

### Typical API Request Flow

```
1. HTTP Request
   ↓
2. Router (Gorilla Mux) matches route
   ↓
3. Middleware Stack (auth, logging, etc.)
   ↓
4. Handler receives request
   ↓
5. Handler creates Command/Query
   ↓
6. Mediator receives request
   ↓
7. Mediator executes Behaviors (validation, auth)
   ↓
8. Mediator routes to Command/Query Handler
   ↓
9. Handler uses Repository for data access
   ↓
10. Handler uses Services for business logic
    ↓
11. Handler may emit Events
    ↓
12. Result flows back through layers
    ↓
13. Handler converts to HTTP response
    ↓
14. Response sent to client
```

### Example: Creating a User

```
POST /api/v1/users
↓
UsersHandler.CreateUser()
↓
mediatr.Send(CreateUserCommand{...})
↓
[PolicyMiddleware validates permissions]
↓
CreateUserHandler.Handle()
├─→ userRepo.Create() - Saves user to database
├─→ tokenService.Generate() - Creates verification token
├─→ mediatr.SendEvent(UserCreatedEvent{...})
│   └─→ EmailHandler sends welcome email (async)
└─→ Returns CreateUserResult{UserID: ...}
↓
Handler returns HTTP 201 with user details
```

## Component Interaction Diagram

```
┌─────────────────────────────────────────────────────────────────┐
│                         API Server (main.go)                     │
│  • Initializes IoC Container                                    │
│  • Registers all dependencies                                   │
│  • Sets up HTTP server with routes                             │
└────────────────────────────┬────────────────────────────────────┘
                             │
              ┌──────────────┴──────────────┐
              │    IoC Container (Singleton) │
              │  • Resolves dependencies     │
              │  • Manages lifetimes         │
              └──────────────┬───────────────┘
                             │
        ┌────────────────────┼────────────────────┐
        │                    │                    │
        ▼                    ▼                    ▼
┌──────────────┐    ┌──────────────┐    ┌──────────────┐
│  Handlers    │    │   Mediator   │    │ Repositories │
│  (Scoped)    │───>│ (Singleton)  │<───│ (Singleton)  │
└──────────────┘    └──────────────┘    └──────────────┘
                            │
        ┌───────────────────┼───────────────────┐
        │                   │                   │
        ▼                   ▼                   ▼
┌──────────────┐    ┌──────────────┐    ┌──────────────┐
│  Commands    │    │   Queries    │    │  Services    │
│  (Handlers)  │    │  (Handlers)  │    │ (Singleton)  │
└──────────────┘    └──────────────┘    └──────────────┘
        │                   │                   │
        └───────────────────┴───────────────────┘
                            │
                            ▼
                    ┌──────────────┐
                    │   Database   │
                    │ (PostgreSQL) │
                    └──────────────┘
```

## Key Design Decisions

### 1. Why CQRS?

**Benefits**:
- Clear separation between reads and writes
- Optimized read and write operations independently
- Easier to reason about side effects
- Better testability

**Trade-offs**:
- More boilerplate code
- Learning curve for new developers

### 2. Why Mediator Pattern?

**Benefits**:
- Decoupled communication between components
- Easy to add cross-cutting concerns (validation, logging, auth)
- Components don't need to know about each other
- Easier to test in isolation

**Trade-offs**:
- Indirection can make flow harder to trace initially
- Additional abstraction layer

### 3. Why Custom IoC Container?

**Benefits**:
- Full control over dependency resolution
- Type-safe with Go generics
- Lightweight and fast
- Scoped lifetimes for request-specific dependencies

**Trade-offs**:
- Custom code to maintain
- Less ecosystem tooling compared to popular frameworks

### 4. Why Go?

**Benefits**:
- Fast compilation and execution
- Built-in concurrency (goroutines)
- Strong standard library
- Simple, readable syntax
- Excellent for building web services

### 5. Why PostgreSQL?

**Benefits**:
- ACID compliance
- Strong data integrity
- Excellent performance
- Rich feature set
- Mature ecosystem

**SQLite support** (work-in-progress) for single-instance deployments.

## Next Steps

Now that you understand the overall architecture:

1. **Dive deeper into CQRS** → [CQRS and Mediator Pattern](02-cqrs-and-mediatr.md)
2. **Learn about dependency injection** → [Dependency Injection with IoC](03-dependency-injection.md)
3. **Start coding** → [Development Workflow](04-development-workflow.md)

## Additional Resources

- [Architecture Diagram](../ApplicationArchitecture.drawio) - Visual representation (open with draw.io)
- [Clean Architecture Blog](https://blog.cleancoder.com/uncle-bob/2012/08/13/the-clean-architecture.html)
- [CQRS Pattern](https://martinfowler.com/bliki/CQRS.html)
