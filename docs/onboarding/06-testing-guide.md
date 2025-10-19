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

```go
func TestCreateUser_Success(t *testing.T) {
    // Arrange - Set up test data and mocks
    mockRepo := new(mocks.MockUserRepository)
    mockMediator := new(mocks.MockMediator)
    
    handler := &commands.CreateUserHandler{
        userRepo: mockRepo,
        mediator: mockMediator,
    }
    
    cmd := commands.CreateUserCommand{
        VirtualServerID: uuid.New(),
        Username:        "testuser",
        Email:           "test@example.com",
        Password:        "password123",
    }
    
    mockRepo.On("Create", mock.Anything, mock.Anything).Return(nil)
    mockMediator.On("SendEvent", mock.Anything, mock.Anything).Return(nil)
    
    // Act - Execute the code under test
    result, err := handler.Handle(context.Background(), cmd)
    
    // Assert - Verify the results
    assert.NoError(t, err)
    assert.NotEqual(t, uuid.Nil, result.UserID)
    mockRepo.AssertExpectations(t)
}
```

### Testing Commands

```go
// internal/commands/CreateUser_test.go
package commands

import (
    "context"
    "testing"
    "Keyline/internal/repositories/mocks"
    "Keyline/internal/models"
    "github.com/google/uuid"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/mock"
)

func TestCreateUser_Success(t *testing.T) {
    // Setup mocks
    mockRepo := new(mocks.MockUserRepository)
    mockMediator := new(mocks.MockMediator)
    
    handler := &CreateUserHandler{
        userRepo: mockRepo,
        mediator: mockMediator,
    }
    
    // Configure mock behavior
    mockRepo.On("Create", mock.Anything, mock.MatchedBy(func(user *models.User) bool {
        // Verify the user being created has correct fields
        return user.Username == "testuser" && user.Email == "test@example.com"
    })).Return(nil)
    
    mockMediator.On("SendEvent", mock.Anything, mock.Anything).Return(nil)
    
    // Execute
    result, err := handler.Handle(context.Background(), CreateUserCommand{
        VirtualServerID: uuid.New(),
        Username:        "testuser",
        Email:           "test@example.com",
        Password:        "password123",
    })
    
    // Verify
    assert.NoError(t, err)
    assert.NotEqual(t, uuid.Nil, result.UserID)
    mockRepo.AssertExpectations(t)
}

func TestCreateUser_DuplicateUsername(t *testing.T) {
    mockRepo := new(mocks.MockUserRepository)
    handler := &CreateUserHandler{userRepo: mockRepo}
    
    // Simulate duplicate username error
    mockRepo.On("Create", mock.Anything, mock.Anything).
        Return(repositories.ErrDuplicate)
    
    result, err := handler.Handle(context.Background(), CreateUserCommand{
        Username: "duplicate",
        Email:    "test@example.com",
    })
    
    assert.Error(t, err)
    assert.Equal(t, uuid.Nil, result.UserID)
}

func TestCreateUser_InvalidInput(t *testing.T) {
    handler := &CreateUserHandler{}
    
    // Test empty username
    result, err := handler.Handle(context.Background(), CreateUserCommand{
        Username: "",
        Email:    "test@example.com",
    })
    
    assert.Error(t, err)
    assert.Contains(t, err.Error(), "username")
}
```

### Testing Queries

```go
// internal/queries/GetUser_test.go
package queries

func TestGetUser_Success(t *testing.T) {
    // Setup
    mockRepo := new(mocks.MockUserRepository)
    handler := &GetUserHandler{userRepo: mockRepo}
    
    expectedUser := &models.User{
        ID:       uuid.New(),
        Username: "testuser",
        Email:    "test@example.com",
    }
    
    mockRepo.On("GetByID", mock.Anything, expectedUser.ID).
        Return(expectedUser, nil)
    
    // Execute
    result, err := handler.Handle(context.Background(), GetUserQuery{
        UserID: expectedUser.ID,
    })
    
    // Verify
    assert.NoError(t, err)
    assert.Equal(t, expectedUser.ID, result.UserID)
    assert.Equal(t, expectedUser.Username, result.Username)
}

func TestGetUser_NotFound(t *testing.T) {
    mockRepo := new(mocks.MockUserRepository)
    handler := &GetUserHandler{userRepo: mockRepo}
    
    userID := uuid.New()
    mockRepo.On("GetByID", mock.Anything, userID).
        Return(nil, repositories.ErrNotFound)
    
    result, err := handler.Handle(context.Background(), GetUserQuery{
        UserID: userID,
    })
    
    assert.Error(t, err)
    assert.True(t, errors.Is(err, repositories.ErrNotFound))
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

### Creating Mocks

Keyline uses `testify/mock` for mocking:

```go
// internal/repositories/mocks/user_repository.go
package mocks

import (
    "context"
    "Keyline/internal/models"
    "Keyline/internal/repositories"
    "github.com/google/uuid"
    "github.com/stretchr/testify/mock"
)

type MockUserRepository struct {
    mock.Mock
}

func (m *MockUserRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.User, error) {
    args := m.Called(ctx, id)
    if args.Get(0) == nil {
        return nil, args.Error(1)
    }
    return args.Get(0).(*models.User), args.Error(1)
}

func (m *MockUserRepository) Create(ctx context.Context, user *models.User) error {
    args := m.Called(ctx, user)
    return args.Error(0)
}

// ... other methods
```

### Using Mocks

```go
func TestCreateUser(t *testing.T) {
    // Create mock
    mockRepo := new(mocks.MockUserRepository)
    
    // Set expectations
    mockRepo.On("Create", 
        mock.Anything,  // any context
        mock.MatchedBy(func(user *models.User) bool {
            return user.Username == "testuser"
        })).Return(nil)
    
    // Use mock
    handler := &CreateUserHandler{userRepo: mockRepo}
    result, err := handler.Handle(context.Background(), CreateUserCommand{
        Username: "testuser",
    })
    
    // Verify expectations were met
    assert.NoError(t, err)
    mockRepo.AssertExpectations(t)
}
```

### Mock Patterns

```go
// Return different values on consecutive calls
mockRepo.On("GetByID", mock.Anything, userID).
    Return(user1, nil).Once().
    On("GetByID", mock.Anything, userID).
    Return(user2, nil).Once()

// Simulate error
mockRepo.On("Create", mock.Anything, mock.Anything).
    Return(errors.New("database error"))

// Run function when called
mockRepo.On("Create", mock.Anything, mock.Anything).
    Run(func(args mock.Arguments) {
        user := args.Get(1).(*models.User)
        user.ID = uuid.New()  // Simulate ID generation
    }).Return(nil)

// Match specific argument
mockRepo.On("GetByUsername", 
    mock.Anything, 
    mock.Anything,
    "specificUsername").Return(user, nil)
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
// Good
func TestCreateUser_WithDuplicateUsername_ReturnsError(t *testing.T)
func TestCreateUser_WithValidData_CreatesUserSuccessfully(t *testing.T)
func TestCreateUser_WithEmptyUsername_ReturnsValidationError(t *testing.T)
```

### 3. One Assertion Per Test (When Possible)

```go
// Prefer
func TestCreateUser_ReturnsUserID(t *testing.T) {
    result, _ := handler.Handle(ctx, cmd)
    assert.NotEqual(t, uuid.Nil, result.UserID)
}

func TestCreateUser_EmitsEvent(t *testing.T) {
    handler.Handle(ctx, cmd)
    mockMediator.AssertCalled(t, "SendEvent", mock.Anything, mock.Anything)
}

// Over
func TestCreateUser_Success(t *testing.T) {
    result, err := handler.Handle(ctx, cmd)
    assert.NoError(t, err)
    assert.NotEqual(t, uuid.Nil, result.UserID)
    assert.Equal(t, "testuser", result.Username)
    mockRepo.AssertExpectations(t)
    mockMediator.AssertCalled(t, "SendEvent")
}
```

### 4. Use Test Fixtures Wisely

```go
// Create reusable test data
func newTestUser() *models.User {
    return &models.User{
        ID:              uuid.New(),
        VirtualServerID: uuid.New(),
        Username:        "testuser",
        Email:           "test@example.com",
        IsActive:        true,
    }
}

func TestCreateUser(t *testing.T) {
    user := newTestUser()
    // Use in test...
}
```

### 5. Clean Up Resources

```go
func TestWithDatabase(t *testing.T) {
    db := setupTestDB(t)
    
    // Cleanup when test completes
    t.Cleanup(func() {
        cleanupDB(t, db)
        db.Close()
    })
    
    // Test code...
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
