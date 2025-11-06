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
    "github.com/The127/mediatr"
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
    m mediatr.Mediator,
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
        ioc.RegisterTransient(dc, func(_ *ioc.DependencyProvider) mediatr.Mediator {
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
    mockmediatr.EXPECT().SendEvent(gomock.Any(), gomock.AssignableToTypeOf(events.UserCreatedEvent{}), gomock.Any())
    
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
    ctrl := gomock.NewController(t)
    defer ctrl.Finish()
    
    mockRepo := mocks.NewMockUserRepository(ctrl)
    
    // Create context with timeout
    ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
    defer cancel()
    
    // Simulate slow operation
    mockRepo.EXPECT().
        Insert(gomock.Any(), gomock.Any()).
        DoAndReturn(func(ctx context.Context, user *repositories.User) error {
            time.Sleep(200 * time.Millisecond)
            return nil
        })
    
    // Create context with mocked dependency
    testCtx := createTestContext(mockRepo)
    
    _, err := HandleCreateUser(testCtx, CreateUser{
        Username: "test",
        Email:    "test@example.com",
    })
    
    assert.Error(t, err)
    assert.True(t, errors.Is(err, context.DeadlineExceeded))
}
```

## Integration Testing

Integration tests test multiple components working together using real dependencies (database, mediator) but without the HTTP layer. Keyline uses **Ginkgo** (BDD testing framework) and **Gomega** (matcher library) for integration tests.

### Test Framework

Integration tests use:
- **[Ginkgo](https://onsi.github.io/ginkgo/)**: BDD-style testing framework
- **[Gomega](https://onsi.github.io/gomega/)**: Matcher/assertion library
- **Test Harness**: Custom test infrastructure (`harness.go`) for isolated test environments
- **Mediator Pattern**: Commands and queries are tested through the mediator

### Test Structure

```go
// tests/integration/suite_test.go
//go:build integration
// +build integration

package integration

import (
    "Keyline/internal/logging"
    "testing"

    . "github.com/onsi/ginkgo/v2"
    . "github.com/onsi/gomega"
)

func TestIntegration(t *testing.T) {
    RegisterFailHandler(Fail)
    logging.Init()
    RunSpecs(t, "Integration Suite")
}
```

### Test Harness

The test harness (`harness.go`) provides:
- **Unique database**: Randomly named PostgreSQL database for complete isolation
- **Mediator**: Configured with all dependencies
- **Context**: Pre-configured with system user authentication
- **Time mocking**: Ability to set/control time for time-dependent tests
- **Automatic cleanup**: Database is dropped after tests complete

Key harness methods:
```go
h.Mediator()       // Get the mediator instance
h.Ctx()            // Get the context with authentication
h.VirtualServer()  // Get the test virtual server name ("test-vs")
h.SetTime(t)       // Set the current time for testing
h.Close()          // Clean up (called in AfterAll)
```

### Integration Test Example

```go
// tests/integration/application_flow_test.go
//go:build integration

package integration

import (
    "Keyline/internal/commands"
    "Keyline/internal/queries"
    "Keyline/internal/repositories"
    "github.com/The127/mediatr"
    "Keyline/utils"

    "github.com/google/uuid"
    . "github.com/onsi/ginkgo/v2"
    . "github.com/onsi/gomega"
    "github.com/onsi/gomega/gstruct"
)

var _ = Describe("Application flow", Ordered, func() {
    var h *harness
    var applicationId uuid.UUID

    BeforeAll(func() {
        // Create test harness once for all tests in this suite
        h = newIntegrationTestHarness()
    })

    AfterAll(func() {
        // Clean up after all tests
        h.Close()
    })

    It("should persist public application successfully", func() {
        // Arrange
        req := commands.CreateApplication{
            VirtualServerName:      h.VirtualServer(),
            Name:                   "test-app",
            DisplayName:            "Test App",
            Type:                   repositories.ApplicationTypePublic,
            RedirectUris:           []string{"http://localhost:8080/callback"},
            PostLogoutRedirectUris: []string{"http://localhost:8080/logout"},
        }
        
        // Act
        response, err := mediatr.Send[*commands.CreateApplicationResponse](
            h.Ctx(), h.Mediator(), req)
        
        // Assert
        Expect(err).ToNot(HaveOccurred())
        applicationId = response.Id
    })

    It("should list applications successfully", func() {
        req := queries.ListApplications{
            VirtualServerName: h.VirtualServer(),
            SearchText:        "test-app",
        }
        
        response, err := mediatr.Send[*queries.ListApplicationsResponse](
            h.Ctx(), h.Mediator(), req)
        
        Expect(err).ToNot(HaveOccurred())
        Expect(response.Items).To(ContainElement(gstruct.MatchFields(gstruct.IgnoreExtras, gstruct.Fields{
            "Id":   Equal(applicationId),
            "Name": Equal("test-app"),
        })))
    })

    It("should edit application successfully", func() {
        cmd := commands.PatchApplication{
            VirtualServerName: h.VirtualServer(),
            ApplicationId:     applicationId,
            DisplayName:       utils.Ptr("Updated Test App"),
        }
        
        _, err := mediatr.Send[*commands.PatchApplicationResponse](
            h.Ctx(), h.Mediator(), cmd)
        
        Expect(err).ToNot(HaveOccurred())
    })

    It("should reflect updated values", func() {
        req := queries.GetApplication{
            VirtualServerName: h.VirtualServer(),
            ApplicationId:     applicationId,
        }
        
        app, err := mediatr.Send[*queries.GetApplicationResult](
            h.Ctx(), h.Mediator(), req)
        
        Expect(err).ToNot(HaveOccurred())
        Expect(app.DisplayName).To(Equal("Updated Test App"))
    })
})
```

### Test Isolation

Each test suite (each `Describe` block with a harness) gets:
- **Unique database**: Complete data isolation between test suites
- **Fresh dependencies**: Clean mediator and repository instances
- **No data pollution**: Tests can be run in any order

The `Ordered` flag ensures tests within a suite run sequentially, which is useful when later tests depend on earlier ones (e.g., creating then updating an entity).

### Running Integration Tests

```bash
# Start test database
docker compose up -d postgres

# Run integration tests
just integration

# Or manually with Go
go test -race -count=1 -tags=integration ./tests/integration/...

# Or with Ginkgo CLI
ginkgo -tags=integration ./tests/integration/
```

## End-to-End Testing

End-to-end tests validate the complete Keyline system by running the actual API server and making real HTTP requests. These tests ensure that all components work together correctly in a production-like environment.

### Test Framework

E2E tests use:
- **[Ginkgo](https://onsi.github.io/ginkgo/)**: BDD-style testing framework
- **[Gomega](https://onsi.github.io/gomega/)**: Matcher/assertion library
- **Test Harness**: Custom test infrastructure (`harness.go`) for isolated test environments
- **Keyline API Client**: Type-safe Go client for API interactions

### Test Structure

```go
// tests/e2e/suite_test.go
//go:build e2e
// +build e2e

package e2e

import (
    "Keyline/internal/logging"
    "testing"

    . "github.com/onsi/ginkgo/v2"
    . "github.com/onsi/gomega"
)

func TestE2e(t *testing.T) {
    RegisterFailHandler(Fail)
    logging.Init()
    RunSpecs(t, "e2e Suite")
}
```

### Test Harness

The E2E test harness provides:
- **Unique database**: Randomly named PostgreSQL database for complete isolation
- **Unique server port**: Avoids port conflicts when running tests in parallel
- **API client**: Configured to communicate with the test server
- **Context**: Pre-configured with authentication and scope
- **Time mocking**: Ability to set/control time for time-dependent tests
- **Automatic cleanup**: Database and server are cleaned up after tests complete

Key harness methods:
```go
h.Client()         // Get the API client
h.Ctx()            // Get the context with authentication
h.VirtualServer()  // Get the test virtual server name ("test-vs")
h.SetTime(t)       // Set the current time for testing
h.Close()          // Clean up (called in AfterAll)
```

### E2E Test Example

```go
// tests/e2e/application_flow_test.go
//go:build e2e

package e2e

import (
    "Keyline/internal/handlers"

    . "github.com/onsi/ginkgo/v2"
    . "github.com/onsi/gomega"
)

var _ = Describe("Application flow", Ordered, func() {
    var h *harness

    BeforeAll(func() {
        // Create test harness once for all tests in this suite
        h = newE2eTestHarness()
    })

    AfterAll(func() {
        // Clean up after all tests
        h.Close()
    })

    It("rejects unauthorized requests", func() {
        // Attempt to create application without authentication
        _, err := h.Client().Application().Create(h.Ctx(), handlers.CreateApplicationRequestDto{
            Name:           "test-app",
            DisplayName:    "Test App",
            RedirectUris:   []string{"http://localhost:8080/callback"},
            PostLogoutUris: []string{"http://localhost:8080/logout"},
            Type:           "public",
        })
        
        // Should receive 401 Unauthorized
        Expect(err).To(HaveOccurred())
        Expect(err).To(MatchError(ContainSubstring("401 Unauthorized")))
    })
})
```

### More E2E Examples

For comprehensive examples including:
- Authentication flows
- Time-dependent testing
- Database management
- Server configuration
- Best practices

See the [E2E Test README](../../tests/e2e/README.md) which provides detailed documentation on:
- Test architecture and components
- Test isolation strategies
- Writing effective E2E tests
- Using the test harness
- Testing with authentication
- CI/CD integration

### Running E2E Tests

```bash
# Start test dependencies
docker compose up -d postgres

# Run E2E tests
just e2e

# Or manually with Go
go test -tags=e2e ./tests/e2e/...

# Or with Ginkgo CLI
ginkgo -tags=e2e ./tests/e2e/

# Run specific test
ginkgo -tags=e2e --focus "Application flow" ./tests/e2e/
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
mockmediatr.EXPECT().
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
    ctrl := gomock.NewController(t)
    defer ctrl.Finish()
    
    mockHasher := mocks.NewMockPasswordHasher(ctrl)
    mockHasher.EXPECT().Hash("password123").Return("hashed", nil)
    // This test breaks if we change hash implementation
}

// Good - Tests behavior
func TestCreateUser_StoresHashedPassword(t *testing.T) {
    ctrl := gomock.NewController(t)
    defer ctrl.Finish()
    
    mockRepo := mocks.NewMockUserRepository(ctrl)
    mockRepo.EXPECT().Insert(gomock.Any(), gomock.Any()).Return(nil)
    
    ctx := createTestContext(mockRepo)
    result, err := HandleCreateUser(ctx, CreateUser{
        Password: "password123",
    })
    assert.NoError(t, err)
    // Verify user was created successfully
    assert.NotNil(t, result)
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
