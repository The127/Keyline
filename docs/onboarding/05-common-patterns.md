# Common Patterns and Examples

This guide provides real-world examples and common patterns you'll encounter when working with Keyline.

## Table of Contents
- [Creating a New Command](#creating-a-new-command)
- [Creating a New Query](#creating-a-new-query)
- [Adding a New API Endpoint](#adding-a-new-api-endpoint)
- [Working with Repositories](#working-with-repositories)
- [Handling Events](#handling-events)
- [Authorization and Permissions](#authorization-and-permissions)
- [Error Handling](#error-handling)
- [Pagination](#pagination)
- [Caching Patterns](#caching-patterns)
- [Transaction Management](#transaction-management)

## Creating a New Command

Commands modify state. Here's the complete pattern:

### Basic Command Structure

```go
// internal/commands/UpdateUserProfile.go
package commands

import (
    "context"
    "errors"
    "Keyline/internal/repositories"
    "Keyline/internal/models"
    "Keyline/mediator"
    "github.com/google/uuid"
)

// 1. Define the command
type UpdateUserProfileCommand struct {
    UserID      uuid.UUID
    DisplayName string
    Bio         string
}

// 2. Define the result
type UpdateUserProfileResult struct {
    Success bool
}

// 3. Define the handler
type UpdateUserProfileHandler struct {
    userRepo repositories.UserRepository
    mediator mediator.Mediator
}

// 4. Implement Handle method
func (h *UpdateUserProfileHandler) Handle(
    ctx context.Context,
    cmd UpdateUserProfileCommand,
) (UpdateUserProfileResult, error) {
    // Validate input
    if cmd.UserID == uuid.Nil {
        return UpdateUserProfileResult{}, errors.New("user ID required")
    }
    
    // Get existing user
    user, err := h.userRepo.GetByID(ctx, cmd.UserID)
    if err != nil {
        return UpdateUserProfileResult{}, fmt.Errorf("user not found: %w", err)
    }
    
    // Apply changes
    user.DisplayName = cmd.DisplayName
    user.Bio = cmd.Bio
    user.UpdatedAt = time.Now()
    
    // Save changes
    if err := h.userRepo.Update(ctx, user); err != nil {
        return UpdateUserProfileResult{}, fmt.Errorf("failed to update: %w", err)
    }
    
    // Emit event (optional)
    _ = mediator.SendEvent(ctx, h.mediator, UserProfileUpdatedEvent{
        UserID:      user.ID,
        DisplayName: user.DisplayName,
    })
    
    return UpdateUserProfileResult{Success: true}, nil
}
```

### Register the Command

```go
// internal/setup/setup.go
func Commands(dc *ioc.DependencyCollection) {
    // ... other commands ...
    
    ioc.RegisterSingleton(dc, func(dp *ioc.DependencyProvider) any {
        m := ioc.GetDependency[mediator.Mediator](dp)
        userRepo := ioc.GetDependency[repositories.UserRepository](dp)
        
        handler := &commands.UpdateUserProfileHandler{
            userRepo: userRepo,
            mediator: m,
        }
        
        mediator.RegisterHandler(m, handler.Handle)
        return handler
    })
}
```

## Creating a New Query

Queries read data without side effects:

### Basic Query Structure

```go
// internal/queries/ListUsers.go
package queries

import (
    "context"
    "Keyline/internal/repositories"
    "github.com/google/uuid"
)

// 1. Define the query
type ListUsersQuery struct {
    VirtualServerID uuid.UUID
    Page            int
    PageSize        int
    SearchTerm      string
}

// 2. Define the result
type ListUsersResult struct {
    Users      []UserDTO
    TotalCount int
    Page       int
    PageSize   int
}

type UserDTO struct {
    UserID        uuid.UUID `json:"userId"`
    Username      string    `json:"username"`
    Email         string    `json:"email"`
    DisplayName   string    `json:"displayName"`
    EmailVerified bool      `json:"emailVerified"`
    IsActive      bool      `json:"isActive"`
}

// 3. Define the handler
type ListUsersHandler struct {
    userRepo repositories.UserRepository
}

// 4. Implement Handle method
func (h *ListUsersHandler) Handle(
    ctx context.Context,
    query ListUsersQuery,
) (ListUsersResult, error) {
    // Validate
    if query.Page < 1 {
        query.Page = 1
    }
    if query.PageSize < 1 || query.PageSize > 100 {
        query.PageSize = 20
    }
    
    // Get users with pagination
    users, total, err := h.userRepo.List(ctx, repositories.ListUsersParams{
        VirtualServerID: query.VirtualServerID,
        Offset:          (query.Page - 1) * query.PageSize,
        Limit:           query.PageSize,
        SearchTerm:      query.SearchTerm,
    })
    if err != nil {
        return ListUsersResult{}, err
    }
    
    // Map to DTOs
    dtos := make([]UserDTO, len(users))
    for i, user := range users {
        dtos[i] = UserDTO{
            UserID:        user.ID,
            Username:      user.Username,
            Email:         user.Email,
            DisplayName:   user.DisplayName,
            EmailVerified: user.EmailVerified,
            IsActive:      user.IsActive,
        }
    }
    
    return ListUsersResult{
        Users:      dtos,
        TotalCount: total,
        Page:       query.Page,
        PageSize:   query.PageSize,
    }, nil
}
```

### Register the Query

```go
// internal/setup/setup.go
func Queries(dc *ioc.DependencyCollection) {
    // ... other queries ...
    
    ioc.RegisterSingleton(dc, func(dp *ioc.DependencyProvider) any {
        m := ioc.GetDependency[mediator.Mediator](dp)
        userRepo := ioc.GetDependency[repositories.UserRepository](dp)
        
        handler := &queries.ListUsersHandler{
            userRepo: userRepo,
        }
        
        mediator.RegisterHandler(m, handler.Handle)
        return handler
    })
}
```

## Adding a New API Endpoint

Complete example from HTTP request to response:

### 1. Create the Handler Method

```go
// internal/handlers/users.go

// @Summary     List users
// @Description Get a paginated list of users
// @Tags        users
// @Accept      json
// @Produce     json
// @Param       page query int false "Page number" default(1)
// @Param       pageSize query int false "Page size" default(20)
// @Param       search query string false "Search term"
// @Success     200 {object} queries.ListUsersResult
// @Failure     400 {object} ErrorResponse
// @Failure     401 {object} ErrorResponse
// @Security    BearerAuth
// @Router      /api/v1/users [get]
func (h *UserHandlers) ListUsers(w http.ResponseWriter, r *http.Request) {
    // Parse query parameters
    page, _ := strconv.Atoi(r.URL.Query().Get("page"))
    if page < 1 {
        page = 1
    }
    
    pageSize, _ := strconv.Atoi(r.URL.Query().Get("pageSize"))
    if pageSize < 1 {
        pageSize = 20
    }
    
    searchTerm := r.URL.Query().Get("search")
    
    // Get virtual server ID from context (set by middleware)
    vsID := authentication.GetVirtualServerID(r.Context())
    
    // Send query via mediator
    result, err := mediator.Send[queries.ListUsersResult](
        r.Context(),
        h.mediator,
        queries.ListUsersQuery{
            VirtualServerID: vsID,
            Page:            page,
            PageSize:        pageSize,
            SearchTerm:      searchTerm,
        },
    )
    
    if err != nil {
        h.handleError(w, err)
        return
    }
    
    // Return response
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(http.StatusOK)
    json.NewEncoder(w).Encode(result)
}
```

### 2. Register the Route

```go
// internal/handlers/users.go
func (h *UserHandlers) RegisterRoutes(router *mux.Router) {
    api := router.PathPrefix("/api/v1").Subrouter()
    
    // Apply authentication middleware
    api.Use(h.authMiddleware)
    
    // User routes
    api.HandleFunc("/users", h.ListUsers).Methods("GET")
    api.HandleFunc("/users", h.CreateUser).Methods("POST")
    api.HandleFunc("/users/{id}", h.GetUser).Methods("GET")
    api.HandleFunc("/users/{id}", h.UpdateUser).Methods("PUT")
    api.HandleFunc("/users/{id}", h.DeleteUser).Methods("DELETE")
}
```

### 3. Error Handling in Handlers

```go
type ErrorResponse struct {
    Error   string `json:"error"`
    Code    string `json:"code,omitempty"`
    Details any    `json:"details,omitempty"`
}

func (h *UserHandlers) handleError(w http.ResponseWriter, err error) {
    var status int
    var response ErrorResponse
    
    switch {
    case errors.Is(err, repositories.ErrNotFound):
        status = http.StatusNotFound
        response = ErrorResponse{Error: "Resource not found"}
        
    case errors.Is(err, repositories.ErrDuplicate):
        status = http.StatusConflict
        response = ErrorResponse{Error: "Resource already exists"}
        
    case errors.Is(err, ErrUnauthorized):
        status = http.StatusUnauthorized
        response = ErrorResponse{Error: "Unauthorized"}
        
    case errors.Is(err, ErrForbidden):
        status = http.StatusForbidden
        response = ErrorResponse{Error: "Forbidden"}
        
    default:
        status = http.StatusInternalServerError
        response = ErrorResponse{Error: "Internal server error"}
        
        // Log unexpected errors
        logging.Logger.Error("unexpected error",
            zap.Error(err),
        )
    }
    
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(status)
    json.NewEncoder(w).Encode(response)
}
```

## Working with Repositories

### Basic CRUD Operations

```go
// internal/repositories/users.go
package repositories

type UserRepository interface {
    GetByID(ctx context.Context, id uuid.UUID) (*models.User, error)
    GetByUsername(ctx context.Context, vsID uuid.UUID, username string) (*models.User, error)
    GetByEmail(ctx context.Context, vsID uuid.UUID, email string) (*models.User, error)
    List(ctx context.Context, params ListUsersParams) ([]*models.User, int, error)
    Create(ctx context.Context, user *models.User) error
    Update(ctx context.Context, user *models.User) error
    Delete(ctx context.Context, id uuid.UUID) error
}
```

### PostgreSQL Implementation

```go
// internal/repositories/postgres/users.go
package postgres

import (
    "context"
    "database/sql"
    "Keyline/internal/models"
    "Keyline/internal/repositories"
    "github.com/huandu/go-sqlbuilder"
    "github.com/google/uuid"
)

type userRepository struct {
    db *sql.DB
}

func NewUserRepository(db *sql.DB) repositories.UserRepository {
    return &userRepository{db: db}
}

func (r *userRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.User, error) {
    sb := sqlbuilder.PostgreSQL.NewSelectBuilder()
    sb.Select("id", "virtual_server_id", "username", "email", "display_name", 
              "password_hash", "email_verified", "is_active", "created_at", "updated_at")
    sb.From("users")
    sb.Where(sb.Equal("id", id))
    
    query, args := sb.Build()
    
    var user models.User
    err := r.db.QueryRowContext(ctx, query, args...).Scan(
        &user.ID,
        &user.VirtualServerID,
        &user.Username,
        &user.Email,
        &user.DisplayName,
        &user.PasswordHash,
        &user.EmailVerified,
        &user.IsActive,
        &user.CreatedAt,
        &user.UpdatedAt,
    )
    
    if err == sql.ErrNoRows {
        return nil, repositories.ErrNotFound
    }
    if err != nil {
        return nil, err
    }
    
    return &user, nil
}

func (r *userRepository) Create(ctx context.Context, user *models.User) error {
    if user.ID == uuid.Nil {
        user.ID = uuid.New()
    }
    
    sb := sqlbuilder.PostgreSQL.NewInsertBuilder()
    sb.InsertInto("users")
    sb.Cols("id", "virtual_server_id", "username", "email", "display_name", 
            "password_hash", "email_verified", "is_active", "created_at", "updated_at")
    sb.Values(user.ID, user.VirtualServerID, user.Username, user.Email, 
              user.DisplayName, user.PasswordHash, user.EmailVerified, 
              user.IsActive, user.CreatedAt, user.UpdatedAt)
    
    query, args := sb.Build()
    
    _, err := r.db.ExecContext(ctx, query, args...)
    return err
}

func (r *userRepository) List(
    ctx context.Context,
    params repositories.ListUsersParams,
) ([]*models.User, int, error) {
    // Count query
    countSb := sqlbuilder.PostgreSQL.NewSelectBuilder()
    countSb.Select("COUNT(*)")
    countSb.From("users")
    countSb.Where(countSb.Equal("virtual_server_id", params.VirtualServerID))
    
    if params.SearchTerm != "" {
        countSb.Where(countSb.Or(
            countSb.Like("username", "%"+params.SearchTerm+"%"),
            countSb.Like("email", "%"+params.SearchTerm+"%"),
        ))
    }
    
    countQuery, countArgs := countSb.Build()
    
    var total int
    err := r.db.QueryRowContext(ctx, countQuery, countArgs...).Scan(&total)
    if err != nil {
        return nil, 0, err
    }
    
    // List query
    sb := sqlbuilder.PostgreSQL.NewSelectBuilder()
    sb.Select("id", "virtual_server_id", "username", "email", "display_name",
              "password_hash", "email_verified", "is_active", "created_at", "updated_at")
    sb.From("users")
    sb.Where(sb.Equal("virtual_server_id", params.VirtualServerID))
    
    if params.SearchTerm != "" {
        sb.Where(sb.Or(
            sb.Like("username", "%"+params.SearchTerm+"%"),
            sb.Like("email", "%"+params.SearchTerm+"%"),
        ))
    }
    
    sb.OrderBy("created_at DESC")
    sb.Limit(params.Limit)
    sb.Offset(params.Offset)
    
    query, args := sb.Build()
    
    rows, err := r.db.QueryContext(ctx, query, args...)
    if err != nil {
        return nil, 0, err
    }
    defer rows.Close()
    
    var users []*models.User
    for rows.Next() {
        var user models.User
        err := rows.Scan(
            &user.ID,
            &user.VirtualServerID,
            &user.Username,
            &user.Email,
            &user.DisplayName,
            &user.PasswordHash,
            &user.EmailVerified,
            &user.IsActive,
            &user.CreatedAt,
            &user.UpdatedAt,
        )
        if err != nil {
            return nil, 0, err
        }
        users = append(users, &user)
    }
    
    return users, total, nil
}
```

## Handling Events

### Define and Emit Events

```go
// internal/events/UserRegistered.go
package events

type UserRegisteredEvent struct {
    UserID          uuid.UUID
    Username        string
    Email           string
    VirtualServerID uuid.UUID
}

// In command handler
func (h *RegisterUserHandler) Handle(
    ctx context.Context,
    cmd RegisterUserCommand,
) (RegisterUserResult, error) {
    // Create user...
    user, err := h.userRepo.Create(ctx, ...)
    if err != nil {
        return RegisterUserResult{}, err
    }
    
    // Emit event
    _ = mediator.SendEvent(ctx, h.mediator, events.UserRegisteredEvent{
        UserID:          user.ID,
        Username:        user.Username,
        Email:           user.Email,
        VirtualServerID: user.VirtualServerID,
    })
    
    return RegisterUserResult{UserID: user.ID}, nil
}
```

### Handle Events

```go
// internal/events/SendWelcomeEmail.go
package events

type SendWelcomeEmailHandler struct {
    emailService services.EmailService
    tokenService services.TokenService
}

func (h *SendWelcomeEmailHandler) Handle(
    ctx context.Context,
    evt UserRegisteredEvent,
) error {
    // Generate verification token
    token, err := h.tokenService.GenerateEmailVerificationToken(evt.UserID)
    if err != nil {
        return err
    }
    
    // Send welcome email
    return h.emailService.SendWelcomeEmail(
        evt.Email,
        evt.Username,
        token,
    )
}

// Register event handler
func Events(dc *ioc.DependencyCollection) {
    ioc.RegisterSingleton(dc, func(dp *ioc.DependencyProvider) any {
        m := ioc.GetDependency[mediator.Mediator](dp)
        emailService := ioc.GetDependency[services.EmailService](dp)
        tokenService := ioc.GetDependency[services.TokenService](dp)
        
        handler := &events.SendWelcomeEmailHandler{
            emailService: emailService,
            tokenService: tokenService,
        }
        
        mediator.RegisterEventHandler(m, handler.Handle)
        return handler
    })
}
```

## Authorization and Permissions

### Check Permissions in Behavior

```go
// internal/behaviours/PolicyMiddleware.go
package behaviours

type PolicyMiddleware struct {
    userRepo repositories.UserRepository
    roleRepo repositories.RoleRepository
}

func (p *PolicyMiddleware) Handle(
    ctx context.Context,
    request any,
    next mediator.Next,
) (any, error) {
    // Get user from context
    userID := authentication.GetUserID(ctx)
    if userID == uuid.Nil {
        return nil, errors.New("unauthorized")
    }
    
    // Check permission based on request type
    switch req := request.(type) {
    case commands.DeleteUserCommand:
        hasPermission := p.checkPermission(ctx, userID, "users:delete")
        if !hasPermission {
            return nil, errors.New("forbidden: missing users:delete permission")
        }
        
    case commands.CreateApplicationCommand:
        hasPermission := p.checkPermission(ctx, userID, "applications:create")
        if !hasPermission {
            return nil, errors.New("forbidden: missing applications:create permission")
        }
    }
    
    // Call next behavior or handler
    return next()
}

func (p *PolicyMiddleware) checkPermission(
    ctx context.Context,
    userID uuid.UUID,
    permission string,
) bool {
    permissions, err := p.roleRepo.GetUserPermissions(ctx, userID)
    if err != nil {
        return false
    }
    
    for _, p := range permissions {
        if p == permission {
            return true
        }
    }
    
    return false
}
```

## Error Handling

### Define Custom Errors

```go
// internal/repositories/errors.go
package repositories

import "errors"

var (
    ErrNotFound  = errors.New("resource not found")
    ErrDuplicate = errors.New("resource already exists")
    ErrConflict  = errors.New("resource conflict")
)
```

### Wrap Errors with Context

```go
func (h *CreateUserHandler) Handle(
    ctx context.Context,
    cmd CreateUserCommand,
) (CreateUserResult, error) {
    // Check if username exists
    existing, err := h.userRepo.GetByUsername(ctx, cmd.VirtualServerID, cmd.Username)
    if err != nil && !errors.Is(err, repositories.ErrNotFound) {
        return CreateUserResult{}, fmt.Errorf("failed to check username: %w", err)
    }
    
    if existing != nil {
        return CreateUserResult{}, fmt.Errorf("username '%s' is already taken", cmd.Username)
    }
    
    // Create user
    user, err := h.userRepo.Create(ctx, ...)
    if err != nil {
        return CreateUserResult{}, fmt.Errorf("failed to create user: %w", err)
    }
    
    return CreateUserResult{UserID: user.ID}, nil
}
```

## Pagination

### Standard Pagination Pattern

```go
type PaginationParams struct {
    Page     int
    PageSize int
}

type PaginatedResult[T any] struct {
    Items      []T `json:"items"`
    TotalCount int `json:"totalCount"`
    Page       int `json:"page"`
    PageSize   int `json:"pageSize"`
    TotalPages int `json:"totalPages"`
}

func NewPaginatedResult[T any](items []T, total, page, pageSize int) PaginatedResult[T] {
    totalPages := (total + pageSize - 1) / pageSize
    
    return PaginatedResult[T]{
        Items:      items,
        TotalCount: total,
        Page:       page,
        PageSize:   pageSize,
        TotalPages: totalPages,
    }
}
```

## Caching Patterns

### Cache Query Results

```go
type GetUserHandler struct {
    userRepo     repositories.UserRepository
    cacheService services.CacheService
}

func (h *GetUserHandler) Handle(
    ctx context.Context,
    query GetUserQuery,
) (GetUserResult, error) {
    // Try cache first
    cacheKey := fmt.Sprintf("user:%s", query.UserID)
    
    var result GetUserResult
    if h.cacheService.Get(ctx, cacheKey, &result) == nil {
        return result, nil
    }
    
    // Cache miss - get from database
    user, err := h.userRepo.GetByID(ctx, query.UserID)
    if err != nil {
        return GetUserResult{}, err
    }
    
    // Map to result
    result = GetUserResult{
        UserID:   user.ID,
        Username: user.Username,
        // ... other fields
    }
    
    // Store in cache (5 minutes TTL)
    _ = h.cacheService.Set(ctx, cacheKey, result, 5*time.Minute)
    
    return result, nil
}
```

### Invalidate Cache on Updates

```go
func (h *UpdateUserHandler) Handle(
    ctx context.Context,
    cmd UpdateUserCommand,
) (UpdateUserResult, error) {
    // Update user
    err := h.userRepo.Update(ctx, ...)
    if err != nil {
        return UpdateUserResult{}, err
    }
    
    // Invalidate cache
    cacheKey := fmt.Sprintf("user:%s", cmd.UserID)
    _ = h.cacheService.Delete(ctx, cacheKey)
    
    return UpdateUserResult{Success: true}, nil
}
```

## Transaction Management

### Using Transactions

```go
func (h *AssignRoleHandler) Handle(
    ctx context.Context,
    cmd AssignRoleCommand,
) (AssignRoleResult, error) {
    // Begin transaction
    tx, err := h.db.BeginTx(ctx, nil)
    if err != nil {
        return AssignRoleResult{}, err
    }
    defer tx.Rollback() // Rollback if not committed
    
    // Create context with transaction
    txCtx := context.WithValue(ctx, "tx", tx)
    
    // Perform operations within transaction
    err = h.assignmentRepo.Create(txCtx, &models.RoleAssignment{
        UserID: cmd.UserID,
        RoleID: cmd.RoleID,
    })
    if err != nil {
        return AssignRoleResult{}, err
    }
    
    // Update user's role count
    err = h.userRepo.IncrementRoleCount(txCtx, cmd.UserID)
    if err != nil {
        return AssignRoleResult{}, err
    }
    
    // Commit transaction
    if err := tx.Commit(); err != nil {
        return AssignRoleResult{}, err
    }
    
    return AssignRoleResult{Success: true}, nil
}
```

## Next Steps

Now that you've seen common patterns:

1. **Learn about testing** → [Testing Guide](06-testing-guide.md)
2. **Review the architecture** → [Architecture Overview](01-architecture-overview.md)
3. **Start contributing** → Pick an issue and implement it!

## Additional Resources

- [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments)
- [Effective Go](https://go.dev/doc/effective_go)
- [Repository Pattern](https://martinfowler.com/eaaCatalog/repository.html)
