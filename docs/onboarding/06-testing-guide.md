# Testing Guide

This guide explains how to write effective tests for Keyline, covering unit tests, integration tests, and end-to-end tests.

## Table of Contents
- [Testing Philosophy](#testing-philosophy)
- [Test Types](#test-types)
- [Unit Testing](#unit-testing)
- [Integration Testing](#integration-testing)
- [End-to-End Testing](#end-to-end-testing)
- [Mocking](#mocking)
- [Test Coverage](#test-coverage)
- [Best Practices](#best-practices)

## Testing Philosophy

Keyline follows these testing principles:

1. **Tests should be fast**: Unit tests run in milliseconds
2. **Tests should be reliable**: No flaky tests
3. **Tests should be isolated**: One test doesn't affect another
4. **Tests should be readable**: Clear intent and structure
5. **Tests should be maintainable**: Easy to update when code changes

### The Testing Pyramid

```
        ┌─────────────┐
        │     E2E     │  Few, slow, expensive
        └─────────────┘
       ┌───────────────┐
       │  Integration  │  Some, moderate speed
       └───────────────┘
     ┌───────────────────┐
     │   Unit Tests      │  Many, fast, cheap
     └───────────────────┘
```

## Test Types

### Unit Tests

- **What**: Test individual components in isolation
- **When**: For every command, query, service, repository
- **Speed**: Very fast (milliseconds)
- **Dependencies**: Mocked

### Integration Tests

- **What**: Test multiple components working together
- **When**: Database interactions, external service integration
- **Speed**: Moderate (seconds)
- **Dependencies**: Real (database, Redis)

### End-to-End Tests

- **What**: Test complete user workflows
- **When**: Critical user journeys
- **Speed**: Slow (seconds to minutes)
- **Dependencies**: Full system running

## Unit Testing

### Structure: Arrange-Act-Assert (AAA)

Keyline uses **gomock** for mocking and **testify/suite** for test organization.

```go
package commands

import (
    "context"
    "testing"
    "Keyline/internal/middlewares"
    "Keyline/internal/repositories"
    "Keyline/internal/repositories/mocks"
    "Keyline/ioc"
    "github.com/stretchr/testify/suite"
    "go.uber.org/mock/gomock"
)

type CreateUserCommandSuite struct {
    suite.Suite
}

func TestCreateUserCommandSuite(t *testing.T) {
    t.Parallel()
    suite.Run(t, new(CreateUserCommandSuite))
}

// Helper to create context with mocked dependencies
func (s *CreateUserCommandSuite) createContext(
    userRepo repositories.UserRepository,
) context.Context {
    dc := ioc.NewDependencyCollection()
    
    if userRepo != nil {
        ioc.RegisterTransient(dc, func(_ *ioc.DependencyProvider) repositories.UserRepository {
            return userRepo
        })
    }
    
    scope := dc.BuildProvider()
    s.T().Cleanup(func() {
        _ = scope.Close()
    })
    
    return middlewares.ContextWithScope(s.T().Context(), scope)
}

func (s *CreateUserCommandSuite) TestCreateUser_Success() {
    // Arrange - Set up test data and mocks
    ctrl := gomock.NewController(s.T())
    defer ctrl.Finish()
    
    mockUserRepo := mocks.NewMockUserRepository(ctrl)
    mockUserRepo.EXPECT().Insert(gomock.Any(), gomock.Any()).Return(nil)
    
    ctx := s.createContext(mockUserRepo)
    cmd := CreateUser{
        Username: "testuser",
        Email:    "test@example.com",
    }
    
    // Act - Execute the code under test
    result, err := HandleCreateUser(ctx, cmd)
    
    // Assert - Verify the results
    s.NoError(err)
    s.NotNil(result)
}
```

### Testing Commands

```go
// internal/commands/CreateUser_test.go
package commands

import (
    "context"
    "errors"
    "testing"
    "Keyline/internal/events"
    "Keyline/internal/middlewares"
    "Keyline/internal/repositories"
    "Keyline/internal/repositories/mocks"
    "Keyline/ioc"
    "Keyline/mediator"
    mediatormocks "Keyline/mediator/mocks"
    "github.com/stretchr/testify/suite"
    "go.uber.org/mock/gomock"
)

type CreateUserCommandSuite struct {
    suite.Suite
}

func TestCreateUserCommandSuite(t *testing.T) {
    t.Parallel()
    suite.Run(t, new(CreateUserCommandSuite))
}

func (s *CreateUserCommandSuite) createContext(
    vsRepo repositories.VirtualServerRepository,
    userRepo repositories.UserRepository,
    m mediator.Mediator,
) context.Context {
    dc := ioc.NewDependencyCollection()
    
    if vsRepo != nil {
        ioc.RegisterTransient(dc, func(_ *ioc.DependencyProvider) repositories.VirtualServerRepository {
            return vsRepo
        })
    }
    
    if userRepo != nil {
        ioc.RegisterTransient(dc, func(_ *ioc.DependencyProvider) repositories.UserRepository {
            return userRepo
        })
    }
    
    if m != nil {
        ioc.RegisterTransient(dc, func(_ *ioc.DependencyProvider) mediator.Mediator {
            return m
        })
    }
    
    scope := dc.BuildProvider()
    s.T().Cleanup(func() {
        _ = scope.Close()
    })
    
    return middlewares.ContextWithScope(s.T().Context(), scope)
}

func (s *CreateUserCommandSuite) TestCreateUser_Success() {
    // Arrange
    ctrl := gomock.NewController(s.T())
    defer ctrl.Finish()
    
    virtualServer := repositories.NewVirtualServer("vs-name", "VS Display")
    
    mockVsRepo := mocks.NewMockVirtualServerRepository(ctrl)
    mockVsRepo.EXPECT().Single(gomock.Any(), gomock.Any()).Return(virtualServer, nil)
    
    mockUserRepo := mocks.NewMockUserRepository(ctrl)
    mockUserRepo.EXPECT().Insert(gomock.Any(), gomock.Any()).Return(nil)
    
    mockMediator := mediatormocks.NewMockMediator(ctrl)
    mockMediator.EXPECT().SendEvent(gomock.Any(), gomock.AssignableToTypeOf(events.UserCreatedEvent{}), gomock.Any())
    
    ctx := s.createContext(mockVsRepo, mockUserRepo, mockMediator)
    cmd := CreateUser{
        VirtualServerName: "vs-name",
        Username:          "testuser",
        Email:             "test@example.com",
    }
    
    // Act
    result, err := HandleCreateUser(ctx, cmd)
    
    // Assert
    s.Require().NoError(err)
    s.NotNil(result)
}

func (s *CreateUserCommandSuite) TestCreateUser_VirtualServerError() {
    // Arrange
    ctrl := gomock.NewController(s.T())
    defer ctrl.Finish()
    
    mockVsRepo := mocks.NewMockVirtualServerRepository(ctrl)
    mockVsRepo.EXPECT().Single(gomock.Any(), gomock.Any()).Return(nil, errors.New("error"))
    
    ctx := s.createContext(mockVsRepo, nil, nil)
    cmd := CreateUser{}
    
    // Act
    _, err := HandleCreateUser(ctx, cmd)
    
    // Assert
    s.Error(err)
}
```

### Testing Queries

Queries follow the same pattern as commands:

```go
// internal/queries/GetUser_test.go
package queries

import (
    "context"
    "errors"
    "testing"
    "Keyline/internal/middlewares"
    "Keyline/internal/repositories"
    "Keyline/internal/repositories/mocks"
    "Keyline/ioc"
    "github.com/stretchr/testify/suite"
    "go.uber.org/mock/gomock"
)

type GetUserQuerySuite struct {
    suite.Suite
}

func TestGetUserQuerySuite(t *testing.T) {
    t.Parallel()
    suite.Run(t, new(GetUserQuerySuite))
}

func (s *GetUserQuerySuite) createContext(
    userRepo repositories.UserRepository,
) context.Context {
    dc := ioc.NewDependencyCollection()
    
    if userRepo != nil {
        ioc.RegisterTransient(dc, func(_ *ioc.DependencyProvider) repositories.UserRepository {
            return userRepo
        })
    }
    
    scope := dc.BuildProvider()
    s.T().Cleanup(func() {
        _ = scope.Close()
    })
    
    return middlewares.ContextWithScope(s.T().Context(), scope)
}

func (s *GetUserQuerySuite) TestGetUser_Success() {
    // Arrange
    ctrl := gomock.NewController(s.T())
    defer ctrl.Finish()
    
    user := repositories.NewUser("testuser", "Test User", "test@example.com", uuid.New())
    
    mockUserRepo := mocks.NewMockUserRepository(ctrl)
    mockUserRepo.EXPECT().First(gomock.Any(), gomock.Any()).Return(user, nil)
    
    ctx := s.createContext(mockUserRepo)
    query := GetUser{UserID: user.Id()}
    
    // Act
    result, err := HandleGetUser(ctx, query)
    
    // Assert
    s.Require().NoError(err)
    s.Equal(user.Id(), result.UserID)
    s.Equal(user.Username(), result.Username)
}

func (s *GetUserQuerySuite) TestGetUser_NotFound() {
    // Arrange
    ctrl := gomock.NewController(s.T())
    defer ctrl.Finish()
    
    mockUserRepo := mocks.NewMockUserRepository(ctrl)
    mockUserRepo.EXPECT().First(gomock.Any(), gomock.Any()).Return(nil, errors.New("not found"))
    
    ctx := s.createContext(mockUserRepo)
    query := GetUser{UserID: uuid.New()}
    
    // Act
    _, err := HandleGetUser(ctx, query)
    
    // Assert
    s.Error(err)
}
```

### Table-Driven Tests

Use when testing multiple scenarios:

```go
func TestValidateEmail(t *testing.T) {
    tests := []struct {
        name    string
        email   string
        wantErr bool
    }{
        {
            name:    "valid email",
            email:   "user@example.com",
            wantErr: false,
        },
        {
            name:    "invalid - no @",
            email:   "userexample.com",
            wantErr: true,
        },
        {
            name:    "invalid - no domain",
            email:   "user@",
            wantErr: true,
        },
        {
            name:    "invalid - empty",
            email:   "",
            wantErr: true,
        },
        {
            name:    "valid - with subdomain",
            email:   "user@mail.example.com",
            wantErr: false,
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := validateEmail(tt.email)
            if tt.wantErr {
                assert.Error(t, err)
            } else {
                assert.NoError(t, err)
            }
        })
    }
}
```

### Testing with Context

```go
func TestCommandWithTimeout(t *testing.T) {
    mockRepo := new(mocks.MockUserRepository)
    handler := &CreateUserHandler{userRepo: mockRepo}
    
    // Create context with timeout
    ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
    defer cancel()
    
    // Simulate slow operation
    mockRepo.On("Create", mock.Anything, mock.Anything).
        Return(nil).
        After(200 * time.Millisecond)
    
    _, err := handler.Handle(ctx, CreateUserCommand{
        Username: "test",
        Email:    "test@example.com",
    })
    
    assert.Error(t, err)
    assert.True(t, errors.Is(err, context.DeadlineExceeded))
}
```

## Integration Testing

Integration tests use real dependencies (database, Redis):

### Setup

```go
// tests/integration/setup_test.go
package integration

import (
    "database/sql"
    "testing"
    "Keyline/internal/database"
    "Keyline/internal/repositories/postgres"
)

func setupTestDB(t *testing.T) *sql.DB {
    // Connect to test database
    db, err := sql.Open("postgres", "postgres://user:pass@localhost:5732/keyline_test?sslmode=disable")
    if err != nil {
        t.Fatalf("failed to connect to test db: %v", err)
    }
    
    // Run migrations
    err = database.RunMigrations(db)
    if err != nil {
        t.Fatalf("failed to run migrations: %v", err)
    }
    
    // Clean up on test completion
    t.Cleanup(func() {
        db.Close()
    })
    
    return db
}

func cleanupDB(t *testing.T, db *sql.DB) {
    // Truncate all tables
    tables := []string{"users", "applications", "roles", "role_assignments"}
    for _, table := range tables {
        _, err := db.Exec("TRUNCATE TABLE " + table + " CASCADE")
        if err != nil {
            t.Logf("failed to truncate %s: %v", table, err)
        }
    }
}
```

### Integration Test Example

```go
// tests/integration/user_repository_test.go
//go:build integration

package integration

import (
    "context"
    "testing"
    "Keyline/internal/models"
    "Keyline/internal/repositories/postgres"
    "github.com/google/uuid"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

func TestUserRepository_CreateAndGet(t *testing.T) {
    // Setup
    db := setupTestDB(t)
    defer cleanupDB(t, db)
    
    repo := postgres.NewUserRepository(db)
    ctx := context.Background()
    
    // Create user
    user := &models.User{
        ID:              uuid.New(),
        VirtualServerID: uuid.New(),
        Username:        "testuser",
        Email:           "test@example.com",
        PasswordHash:    "hashed_password",
        EmailVerified:   false,
        IsActive:        true,
    }
    
    err := repo.Create(ctx, user)
    require.NoError(t, err)
    
    // Get user
    retrieved, err := repo.GetByID(ctx, user.ID)
    require.NoError(t, err)
    
    // Verify
    assert.Equal(t, user.ID, retrieved.ID)
    assert.Equal(t, user.Username, retrieved.Username)
    assert.Equal(t, user.Email, retrieved.Email)
}

func TestUserRepository_List(t *testing.T) {
    db := setupTestDB(t)
    defer cleanupDB(t, db)
    
    repo := postgres.NewUserRepository(db)
    ctx := context.Background()
    vsID := uuid.New()
    
    // Create multiple users
    for i := 0; i < 5; i++ {
        user := &models.User{
            ID:              uuid.New(),
            VirtualServerID: vsID,
            Username:        fmt.Sprintf("user%d", i),
            Email:           fmt.Sprintf("user%d@example.com", i),
            IsActive:        true,
        }
        err := repo.Create(ctx, user)
        require.NoError(t, err)
    }
    
    // List users
    users, total, err := repo.List(ctx, repositories.ListUsersParams{
        VirtualServerID: vsID,
        Offset:          0,
        Limit:           10,
    })
    
    require.NoError(t, err)
    assert.Equal(t, 5, total)
    assert.Len(t, users, 5)
}
```

### Running Integration Tests

```bash
# Start test database
docker compose up -d postgres

# Run integration tests
just integration

# Or manually
go test -tags=integration ./tests/integration/...
```

## End-to-End Testing

E2E tests use the full system including HTTP API:

### E2E Test Example

```go
// tests/e2e/user_registration_test.go
//go:build e2e

package e2e

import (
    "testing"
    "Keyline/client"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

func TestUserRegistration_FullFlow(t *testing.T) {
    // Setup client
    c := client.NewClient("http://localhost:8081")
    
    // 1. Register new user
    registerResp, err := c.RegisterUser(client.RegisterUserRequest{
        Username: "newuser",
        Email:    "newuser@example.com",
        Password: "SecurePass123!",
    })
    require.NoError(t, err)
    assert.NotEmpty(t, registerResp.UserID)
    
    // 2. User should not be able to login (email not verified)
    _, err = c.Login(client.LoginRequest{
        Username: "newuser",
        Password: "SecurePass123!",
    })
    assert.Error(t, err)
    assert.Contains(t, err.Error(), "email not verified")
    
    // 3. Verify email (in real test, get token from email)
    verifyToken := getVerificationTokenFromEmail(t, "newuser@example.com")
    err = c.VerifyEmail(verifyToken)
    require.NoError(t, err)
    
    // 4. Now login should work
    loginResp, err := c.Login(client.LoginRequest{
        Username: "newuser",
        Password: "SecurePass123!",
    })
    require.NoError(t, err)
    assert.NotEmpty(t, loginResp.AccessToken)
    
    // 5. Access protected resource
    c.SetToken(loginResp.AccessToken)
    profile, err := c.GetProfile()
    require.NoError(t, err)
    assert.Equal(t, "newuser", profile.Username)
}

func TestUserRegistration_DuplicateUsername(t *testing.T) {
    c := client.NewClient("http://localhost:8081")
    
    // Create first user
    _, err := c.RegisterUser(client.RegisterUserRequest{
        Username: "duplicate",
        Email:    "user1@example.com",
        Password: "password123",
    })
    require.NoError(t, err)
    
    // Try to create user with same username
    _, err = c.RegisterUser(client.RegisterUserRequest{
        Username: "duplicate",
        Email:    "user2@example.com",
        Password: "password123",
    })
    
    assert.Error(t, err)
    assert.Contains(t, err.Error(), "username already exists")
}
```

### Running E2E Tests

```bash
# Start full system
docker compose up -d
just run &

# Run E2E tests
just e2e

# Or manually
go test -tags=e2e ./tests/e2e/...
```

## Mocking

### Creating Mocks with gomock

Keyline uses **gomock** (go.uber.org/mock/gomock) for generating mocks. Mocks are generated from interfaces using the `mockgen` tool.

#### Generating Mocks

Mocks are generated using `go:generate` directives in the interface files:

```go
// internal/repositories/users.go
//go:generate mockgen -destination=mocks/mock_user_repository.go -package=mocks . UserRepository

package repositories

type UserRepository interface {
    Insert(ctx context.Context, user *User) error
    First(ctx context.Context, filter UserFilter) (*User, error)
    Single(ctx context.Context, filter UserFilter) (*User, error)
    List(ctx context.Context, filter UserFilter) ([]*User, int, error)
    Update(ctx context.Context, user *User) error
    Delete(ctx context.Context, id uuid.UUID) error
}
```

Generate mocks by running:
```bash
go generate ./...
```

### Using Mocks

```go
func (s *CreateUserCommandSuite) TestCreateUser_Success() {
    // Create gomock controller
    ctrl := gomock.NewController(s.T())
    defer ctrl.Finish()
    
    // Create mock repository
    mockRepo := mocks.NewMockUserRepository(ctrl)
    
    // Set expectations with EXPECT()
    mockRepo.EXPECT().
        Insert(gomock.Any(), gomock.Any()).
        Return(nil)
    
    // Create context with mocked dependency
    ctx := s.createContext(mockRepo)
    
    // Execute command
    result, err := HandleCreateUser(ctx, CreateUser{
        Username: "testuser",
    })
    
    // Assertions
    s.NoError(err)
    s.NotNil(result)
    
    // gomock automatically verifies expectations when ctrl.Finish() is called
}
```

### Mock Patterns with gomock

```go
// Match specific parameter values
mockRepo.EXPECT().
    Insert(gomock.Any(), gomock.Cond(func(u *repositories.User) bool {
        return u.Username() == "testuser"
    })).
    Return(nil)

// Simulate error
mockRepo.EXPECT().
    Insert(gomock.Any(), gomock.Any()).
    Return(errors.New("database error"))

// Match specific types
mockMediator.EXPECT().
    SendEvent(gomock.Any(), gomock.AssignableToTypeOf(events.UserCreatedEvent{}), gomock.Any())

// Return different values on consecutive calls
mockRepo.EXPECT().Insert(gomock.Any(), gomock.Any()).Return(nil).Times(1)
mockRepo.EXPECT().Insert(gomock.Any(), gomock.Any()).Return(errors.New("error")).Times(1)

// Use DoAndReturn to execute custom logic
mockRepo.EXPECT().
    Insert(gomock.Any(), gomock.Any()).
    DoAndReturn(func(ctx context.Context, user *repositories.User) error {
        // Custom logic, like setting an ID
        return nil
    })
```

## Test Coverage

### Generate Coverage Report

```bash
# Run tests with coverage
go test -coverprofile=coverage.out ./...

# View coverage in terminal
go tool cover -func=coverage.out

# Generate HTML report
go tool cover -html=coverage.out -o coverage.html

# Open in browser
open coverage.html
```

### Coverage Goals

- **Overall**: Aim for >80% coverage
- **Business Logic**: >90% coverage for commands and queries
- **Repositories**: 100% coverage for critical paths
- **Handlers**: >80% coverage

### What to Cover

**High Priority**:
- Commands (write operations)
- Queries (read operations)
- Critical business logic
- Security-sensitive code
- Error handling paths

**Medium Priority**:
- Repository implementations
- Services
- Middleware
- Validators

**Lower Priority**:
- DTOs and models
- Configuration
- Main functions

## Best Practices

### 1. Test Behavior, Not Implementation

```go
// Bad - Tests implementation details
func TestCreateUser_CallsHashPassword(t *testing.T) {
    mockHasher := new(MockPasswordHasher)
    mockHasher.On("Hash", "password123").Return("hashed", nil)
    // This test breaks if we change hash implementation
}

// Good - Tests behavior
func TestCreateUser_StoresHashedPassword(t *testing.T) {
    handler := &CreateUserHandler{...}
    result, err := handler.Handle(ctx, CreateUserCommand{
        Password: "password123",
    })
    assert.NoError(t, err)
    // Verify password was hashed (without caring how)
    storedUser, _ := repo.GetByID(ctx, result.UserID)
    assert.NotEqual(t, "password123", storedUser.PasswordHash)
}
```

### 2. Use Descriptive Test Names

```go
// Good - Using testify suite
func (s *CreateUserCommandSuite) TestCreateUser_WithValidData_Success()
func (s *CreateUserCommandSuite) TestCreateUser_WithDuplicateUsername_ReturnsError()
func (s *CreateUserCommandSuite) TestCreateUser_VirtualServerNotFound_ReturnsError()
```

### 3. Focus Tests on Specific Scenarios

```go
// Prefer focused tests
func (s *CreateUserCommandSuite) TestCreateUser_Success() {
    // Arrange
    ctrl := gomock.NewController(s.T())
    defer ctrl.Finish()
    
    mockRepo := mocks.NewMockUserRepository(ctrl)
    mockRepo.EXPECT().Insert(gomock.Any(), gomock.Any()).Return(nil)
    
    ctx := s.createContext(mockRepo)
    
    // Act
    result, err := HandleCreateUser(ctx, CreateUser{Username: "testuser"})
    
    // Assert
    s.NoError(err)
    s.NotNil(result)
}

func (s *CreateUserCommandSuite) TestCreateUser_RepositoryError() {
    // Arrange
    ctrl := gomock.NewController(s.T())
    defer ctrl.Finish()
    
    mockRepo := mocks.NewMockUserRepository(ctrl)
    mockRepo.EXPECT().Insert(gomock.Any(), gomock.Any()).Return(errors.New("db error"))
    
    ctx := s.createContext(mockRepo)
    
    // Act
    _, err := HandleCreateUser(ctx, CreateUser{Username: "testuser"})
    
    // Assert
    s.Error(err)
}
```

### 4. Use Test Fixtures Wisely

```go
// Create reusable test data
func newTestUser() *repositories.User {
    return repositories.NewUser(
        "testuser",
        "Test User",
        "test@example.com",
        uuid.New(),
    )
}

func (s *UserTestSuite) TestSomething() {
    user := newTestUser()
    // Use in test...
}
```

### 5. Clean Up Resources

```go
func (s *TestSuite) createContext(...) context.Context {
    dc := ioc.NewDependencyCollection()
    // ... register dependencies
    
    scope := dc.BuildProvider()
    
    // Automatic cleanup
    s.T().Cleanup(func() {
        _ = scope.Close()
    })
    
    return middlewares.ContextWithScope(s.T().Context(), scope)
}
```

### 6. Test Edge Cases

```go
func TestDivide(t *testing.T) {
    tests := []struct {
        name      string
        a, b      float64
        want      float64
        wantError bool
    }{
        {"normal", 10, 2, 5, false},
        {"divide by zero", 10, 0, 0, true},
        {"negative result", 10, -2, -5, false},
        {"very small divisor", 10, 0.0001, 100000, false},
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result, err := divide(tt.a, tt.b)
            if tt.wantError {
                assert.Error(t, err)
            } else {
                assert.NoError(t, err)
                assert.InDelta(t, tt.want, result, 0.0001)
            }
        })
    }
}
```

## Next Steps

You now have a comprehensive understanding of testing in Keyline!

- **Review the architecture** → [Architecture Overview](01-architecture-overview.md)
- **Learn patterns** → [Common Patterns](05-common-patterns.md)
- **Start contributing** → Write tests for existing code or new features

## Additional Resources

- [Go Testing Documentation](https://go.dev/doc/tutorial/add-a-test)
- [Testify Documentation](https://github.com/stretchr/testify)
- [Table-Driven Tests](https://go.dev/wiki/TableDrivenTests)
- [E2E Testing Documentation](../../tests/e2e/README.md)
