# Development Workflow

This guide walks you through the day-to-day development workflow for contributing to Keyline.

## Table of Contents
- [Environment Setup](#environment-setup)
- [Development Tools](#development-tools)
- [Making Your First Change](#making-your-first-change)
- [Running Tests](#running-tests)
- [Linting and Formatting](#linting-and-formatting)
- [Building and Running](#building-and-running)
- [Debugging](#debugging)
- [Common Workflows](#common-workflows)
- [Troubleshooting](#troubleshooting)

## Environment Setup

### Prerequisites

1. **Go 1.24 or higher**
   ```bash
   go version  # Should be >= 1.24
   ```

2. **Just** (command runner)
   ```bash
   # macOS
   brew install just
   
   # Linux (Ubuntu/Debian)
   apt install just
   
   # Other: https://github.com/casey/just#installation
   ```

3. **Docker and Docker Compose** (for dependencies)
   ```bash
   docker --version
   docker compose version
   ```

4. **golangci-lint** (for linting)
   ```bash
   # Install
   go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
   
   # Verify
   golangci-lint --version
   ```

### Clone and Setup

```bash
# 1. Clone the repository
git clone https://github.com/The127/Keyline.git
cd Keyline

# 2. Start dependencies
docker compose up -d

# 3. Copy configuration
cp config.yaml config.local.yaml

# 4. Download Go dependencies
go mod download

# 5. Verify setup
just test
```

### Configure Your Environment

Edit `config.local.yaml` with your development settings:

```yaml
server:
  host: "127.0.0.1"
  port: 8081
  externalUrl: "http://127.0.0.1:8081"

database:
  mode: "postgres"
  postgres:
    host: "localhost"
    port: 5732
    username: "user"
    password: "password"
    database: "keyline"
    sslMode: "disable"

cache:
  mode: "memory"  # Use "redis" for production-like testing

keyStore:
  mode: "directory"
  directory:
    path: "./keys"

logging:
  level: "debug"  # More verbose for development
```

## Development Tools

### Just Commands

Keyline uses `just` as a command runner. See available commands:

```bash
# List all commands
just --list

# Common commands
just build          # Build the application
just run            # Build and run
just test           # Run unit tests
just integration    # Run integration tests
just e2e            # Run end-to-end tests
just lint           # Check for linting issues
just lint fix       # Auto-fix linting issues
just fmt            # Format code
just ci             # Run all CI checks
just ci fix         # Run all CI checks with auto-fix
just clean          # Clean build artifacts
```

### IDE Setup

#### VSCode

Recommended extensions:
- **Go** (`golang.go`) - Official Go extension
- **Go Test Explorer** (`premparihar.gotestexplorer`) - Run tests from UI
- **golangci-lint** (`golangci.golangci-lint`) - Inline linting

Settings (`.vscode/settings.json`):
```json
{
  "go.lintTool": "golangci-lint",
  "go.lintOnSave": "workspace",
  "go.formatTool": "gofmt",
  "editor.formatOnSave": true,
  "go.testFlags": ["-v", "-race"]
}
```

#### GoLand / IntelliJ IDEA

1. Enable Go Modules support
2. Configure golangci-lint as external tool
3. Set gofmt as formatter
4. Enable "Optimize imports on the fly"

## Making Your First Change

### Step-by-Step Example: Adding a New Command

Let's add a command to deactivate a user.

#### 1. Understand the Requirement

**Goal**: Create a `DeactivateUser` command that:
- Marks a user as inactive
- Prevents them from logging in
- Logs the action in audit logs

#### 2. Create the Command File

Create `internal/commands/DeactivateUser.go`:

```go
package commands

import (
    "context"
    "errors"
    "fmt"
    "Keyline/internal/middlewares"
    "Keyline/internal/repositories"
    "Keyline/internal/events"
    "Keyline/ioc"
    "Keyline/mediator"
    "github.com/google/uuid"
)

// Command request
type DeactivateUser struct {
    UserID uuid.UUID
    Reason string
}

// Command response
type DeactivateUserResponse struct {
    Success bool
}

// Handler function
func HandleDeactivateUser(
    ctx context.Context,
    cmd DeactivateUser,
) (*DeactivateUserResponse, error) {
    // Validate
    if cmd.UserID == uuid.Nil {
        return nil, errors.New("user ID required")
    }
    
    // Get scope and dependencies
    scope := middlewares.GetScope(ctx)
    userRepo := ioc.GetDependency[repositories.UserRepository](scope)
    m := ioc.GetDependency[mediator.Mediator](scope)
    
    // Get user
    filter := repositories.NewUserFilter().Id(cmd.UserID)
    user, err := userRepo.First(ctx, filter)
    if err != nil {
        return nil, fmt.Errorf("getting user: %w", err)
    }
    
    // Update status
    user.SetActive(false)
    err = userRepo.Update(ctx, user)
    if err != nil {
        return nil, fmt.Errorf("updating user: %w", err)
    }
    
    // Emit event for audit logging
    _ = mediator.SendEvent(ctx, m, events.UserDeactivatedEvent{
        UserID: cmd.UserID,
        Reason: cmd.Reason,
    })
    
    return &DeactivateUserResponse{Success: true}, nil
}
```

#### 3. Create the Event

Create `internal/events/UserDeactivated.go`:

```go
package events

import "github.com/google/uuid"

type UserDeactivatedEvent struct {
    UserID uuid.UUID
    Reason string
}
```

#### 4. Register the Command

Add to `internal/setup/setup.go`:

```go
func setupHandlers(m mediator.Mediator) {
    // ... existing commands ...
    
    // DeactivateUser command
    mediator.RegisterHandler(m, commands.HandleDeactivateUser)
}
```

#### 5. Create Tests

Create `internal/commands/DeactivateUser_test.go`:

```go
package commands

import (
    "context"
    "testing"
    "Keyline/internal/repositories/mocks"
    "github.com/google/uuid"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/mock"
)

func TestDeactivateUser_Success(t *testing.T) {
    // Setup
    mockRepo := new(mocks.MockUserRepository)
    mockMediator := new(mocks.MockMediator)
    
    userID := uuid.New()
    user := &models.User{
        ID:       userID,
        IsActive: true,
    }
    
    mockRepo.On("GetByID", mock.Anything, userID).Return(user, nil)
    mockRepo.On("Update", mock.Anything, mock.Anything).Return(nil)
    mockMediator.On("SendEvent", mock.Anything, mock.Anything).Return(nil)
    
    handler := &DeactivateUserHandler{
        userRepo: mockRepo,
        mediator: mockMediator,
    }
    
    // Execute
    result, err := handler.Handle(context.Background(), DeactivateUserCommand{
        UserID: userID,
        Reason: "Policy violation",
    })
    
    // Assert
    assert.NoError(t, err)
    assert.True(t, result.Success)
    assert.False(t, user.IsActive)
    mockRepo.AssertExpectations(t)
}

func TestDeactivateUser_InvalidUserID(t *testing.T) {
    handler := &DeactivateUserHandler{}
    
    result, err := handler.Handle(context.Background(), DeactivateUserCommand{
        UserID: uuid.Nil,
    })
    
    assert.Error(t, err)
    assert.False(t, result.Success)
}
```

#### 6. Run Tests

```bash
# Run just this test
go test -v ./internal/commands -run TestDeactivateUser

# Run all tests
just test
```

#### 7. Add HTTP Handler

Add to `internal/handlers/users.go`:

```go
// @Summary     Deactivate user
// @Description Deactivate a user account
// @Tags        users
// @Accept      json
// @Produce     json
// @Param       id path string true "User ID"
// @Success     200 {object} DeactivateUserResult
// @Failure     400 {object} ErrorResponse
// @Security    BearerAuth
// @Router      /api/v1/users/{id}/deactivate [post]
func (h *UserHandlers) DeactivateUser(w http.ResponseWriter, r *http.Request) {
    vars := mux.Vars(r)
    userID, _ := uuid.Parse(vars["id"])
    
    var dto struct {
        Reason string `json:"reason"`
    }
    json.NewDecoder(r.Body).Decode(&dto)
    
    result, err := mediator.Send[commands.DeactivateUserResult](
        r.Context(),
        h.mediator,
        commands.DeactivateUserCommand{
            UserID: userID,
            Reason: dto.Reason,
        },
    )
    
    if err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }
    
    json.NewEncoder(w).Encode(result)
}
```

#### 8. Register the Route

Add to router setup in `internal/handlers/users.go`:

```go
func (h *UserHandlers) RegisterRoutes(router *mux.Router) {
    // ... existing routes ...
    
    router.HandleFunc("/api/v1/users/{id}/deactivate",
        h.DeactivateUser).Methods("POST")
}
```

#### 9. Run Linting

```bash
just lint fix
```

#### 10. Manual Test

```bash
# Start the server
just run

# Test the endpoint
curl -X POST http://localhost:8081/api/v1/users/USER_ID/deactivate \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"reason": "Test deactivation"}'
```

## Running Tests

### Unit Tests

```bash
# Run all unit tests
just test

# Run tests for specific package
go test ./internal/commands/...

# Run specific test
go test -v ./internal/commands -run TestDeactivateUser

# Run with coverage
go test -cover ./...

# Generate coverage report
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

### Integration Tests

```bash
# Run integration tests
just integration

# Run specific integration test
go test -tags=integration ./tests/integration/... -run TestUserFlow
```

### End-to-End Tests

```bash
# Start dependencies first
docker compose up -d

# Run e2e tests
just e2e

# Run with verbose output
go test -v -tags=e2e ./tests/e2e/...
```

### Test Best Practices

1. **Follow the AAA pattern**:
   ```go
   func TestSomething(t *testing.T) {
       // Arrange
       setup := createTestSetup()
       
       // Act
       result := doSomething(setup)
       
       // Assert
       assert.Equal(t, expected, result)
   }
   ```

2. **Use table-driven tests** for multiple scenarios:
   ```go
   func TestValidation(t *testing.T) {
       tests := []struct {
           name    string
           input   string
           wantErr bool
       }{
           {"valid", "test@example.com", false},
           {"invalid", "not-an-email", true},
           {"empty", "", true},
       }
       
       for _, tt := range tests {
           t.Run(tt.name, func(t *testing.T) {
               err := validate(tt.input)
               if tt.wantErr {
                   assert.Error(t, err)
               } else {
                   assert.NoError(t, err)
               }
           })
       }
   }
   ```

3. **Use mocks for external dependencies**:
   ```go
   mockRepo := new(mocks.MockUserRepository)
   mockRepo.On("GetByID", mock.Anything, userID).Return(user, nil)
   ```

## Linting and Formatting

### Format Code

```bash
# Format all Go code
just fmt

# Or manually
go fmt ./...
```

### Run Linter

```bash
# Check for issues
just lint

# Auto-fix issues
just lint fix

# Or manually
golangci-lint run
golangci-lint run --fix
```

### Common Linter Issues

1. **Unused variables**:
   ```go
   // Bad
   result, err := doSomething()
   
   // Good (if result not used)
   _, err := doSomething()
   ```

2. **Error not checked**:
   ```go
   // Bad
   doSomething()
   
   // Good
   if err := doSomething(); err != nil {
       // Handle error
   }
   ```

3. **Ineffective assignment**:
   ```go
   // Bad
   result := something
   result = somethingElse
   
   // Good
   result := somethingElse
   ```

## Building and Running

### Build the Application

```bash
# Build using just
just build

# Or manually
go build -o bin/keyline-api ./cmd/api
```

### Run the Application

```bash
# Run using just (builds first)
just run

# Or manually
./bin/keyline-api --config config.local.yaml

# With environment override
KEYLINE_LOGGING_LEVEL=debug ./bin/keyline-api
```

### Access the Application

- **API**: http://localhost:8081
- **Swagger UI**: http://localhost:8081/swagger/index.html
- **Health Check**: http://localhost:8081/health
- **Metrics**: http://localhost:8081/metrics

### Rebuild Swagger Docs

After adding new API endpoints:

```bash
# Install swag
go install github.com/swaggo/swag/cmd/swag@latest

# Generate docs
swag init -g cmd/api/main.go
```

## Debugging

### Using Print Debugging

```go
import "Keyline/internal/logging"

func (h *Handler) Handle(ctx context.Context, cmd Command) (Result, error) {
    logging.Logger.Debug("handling command",
        zap.String("commandType", "CreateUser"),
        zap.Any("command", cmd),
    )
    
    // Your code...
    
    logging.Logger.Debug("command result",
        zap.Any("result", result),
    )
}
```

### Using Delve Debugger

```bash
# Install delve
go install github.com/go-delve/delve/cmd/dlv@latest

# Debug the application
dlv debug ./cmd/api -- --config config.local.yaml

# In delve
(dlv) break main.main
(dlv) continue
(dlv) step
(dlv) print variableName
```

### VSCode Debugging

Create `.vscode/launch.json`:

```json
{
  "version": "0.2.0",
  "configurations": [
    {
      "name": "Launch API",
      "type": "go",
      "request": "launch",
      "mode": "debug",
      "program": "${workspaceFolder}/cmd/api",
      "args": ["--config", "config.local.yaml"]
    }
  ]
}
```

## Common Workflows

### Adding a New API Endpoint

1. Create command/query in `internal/commands` or `internal/queries`
2. Write tests
3. Register with mediator in `internal/setup/setup.go`
4. Add handler in `internal/handlers`
5. Add Swagger documentation
6. Register route
7. Test manually

### Fixing a Bug

1. Write a failing test that reproduces the bug
2. Fix the code
3. Verify the test passes
4. Run all tests to ensure no regressions
5. Commit with descriptive message

### Refactoring

1. Ensure all existing tests pass
2. Make small, incremental changes
3. Run tests after each change
4. Keep commits small and focused
5. Update documentation if needed

### Adding a New Dependency

```bash
# Add dependency
go get github.com/some/package

# Tidy up
go mod tidy

# Verify tests still pass
just test
```

## Troubleshooting

### Tests Failing

```bash
# Check which tests are failing
go test -v ./...

# Run specific failing test with more output
go test -v ./internal/commands -run TestCreateUser

# Check for race conditions
go test -race ./...
```

### Build Errors

```bash
# Clean and rebuild
just clean
just build

# Update dependencies
go mod tidy
go mod download
```

### Linting Errors

```bash
# See all errors
just lint

# Try auto-fix
just lint fix

# Check specific files
golangci-lint run internal/commands/CreateUser.go
```

### Database Connection Issues

```bash
# Check if PostgreSQL is running
docker compose ps

# View logs
docker compose logs postgres

# Restart services
docker compose restart

# Reset database
docker compose down -v
docker compose up -d
```

### Application Won't Start

1. Check configuration in `config.local.yaml`
2. Verify dependencies are running: `docker compose ps`
3. Check logs for errors
4. Verify database migrations ran successfully
5. Check if port 8081 is already in use: `lsof -i :8081`

## Next Steps

Now that you know the development workflow:

1. **Learn common patterns** → [Common Patterns and Examples](05-common-patterns.md)
2. **Read testing guide** → [Testing Guide](06-testing-guide.md)
3. **Start contributing** → Make your first PR!

## Additional Resources

- [Just Documentation](https://github.com/casey/just)
- [Go Testing](https://go.dev/doc/tutorial/add-a-test)
- [Debugging with Delve](https://github.com/go-delve/delve/tree/master/Documentation)
- [golangci-lint](https://golangci-lint.run/)
