# GitHub Copilot Instructions for Keyline

## Project Overview

Keyline is an open-source OpenID Connect (OIDC) / Identity Provider (IDP) server built with Go. It's designed to be self-hostable, lightweight, fast, secure, and developer-friendly. The project is currently in late alpha and under active development.

## Tech Stack

- **Language**: Go 1.25+
- **Database**: PostgreSQL
- **Cache/Session**: Redis (Valkey)
- **Message Queue**: RabbitMQ
- **Web Framework**: Gorilla Mux
- **SQL Builder**: go-sqlbuilder
- **JWT**: golang-jwt/jwt
- **Logging**: Uber Zap (structured logging)
- **Configuration**: Koanf
- **API Documentation**: Swagger/OpenAPI

## Architecture Patterns

### Clean Architecture
Keyline follows clean architecture principles with clear separation of concerns:

- **Handlers** (`internal/handlers/`): HTTP request handlers and routing
- **Commands** (`internal/commands/`): Write operations in CQRS pattern
- **Queries** (`internal/queries/`): Read operations in CQRS pattern
- **Repositories** (`internal/repositories/`): Data access layer
- **Services** (`internal/services/`): Core business services
- **Mediator**: Request/event mediator pattern for decoupled communication (separate repo)
- **IoC Container**: Custom dependency injection container (separate repo)

### CQRS Pattern
Commands and queries are separated:
- **Commands**: Modify state and are handled through command handlers
- **Queries**: Read data without side effects and are handled through query handlers
- Both use the mediator pattern for decoupled execution

### Mediator Pattern
The mediator pattern is used to implement CQRS:
- All commands and queries go through the mediator
- Handlers are registered with the mediator
- Behaviors can be added for cross-cutting concerns (validation, logging, etc.)

### IoC Container
Custom dependency injection with three lifetime types:
- **Transient**: New instance per resolution
- **Scoped**: Single instance per scope (e.g., per HTTP request)
- **Singleton**: Single instance for application lifetime

## Code Style and Conventions

### General Guidelines
- Follow standard Go conventions (gofmt, golint)
- Use meaningful variable and function names
- Keep functions small and focused
- Prefer composition over inheritance
- Write self-documenting code; add comments only when necessary

### Error Handling
- Always handle errors explicitly
- Use wrapped errors with context: `fmt.Errorf("context: %w", err)`
- Return errors rather than panicking
- Use custom error types when appropriate

### Naming Conventions
- **Packages**: Lowercase, single word (avoid underscores)
- **Types**: PascalCase
- **Functions/Methods**: PascalCase for exported, camelCase for unexported
- **Variables**: camelCase
- **Constants**: PascalCase or ALL_CAPS for package-level constants
- **Interfaces**: Name with "-er" suffix when appropriate (e.g., `Handler`, `Repository`)

### File Organization
- One main type per file
- Group related functions together
- Place tests in `*_test.go` files
- Use internal packages to hide implementation details

## Testing Guidelines

### Test Structure
- Use `testing` package for unit tests
- Follow the pattern: `TestFunctionName` or `TestTypeName_MethodName`
- Use table-driven tests when testing multiple scenarios
- Use `testify` for assertions and mocks
- Place mocks in `*_test.go` files or use `go.uber.org/mock`

### Test Coverage
- Write tests for all business logic
- Focus on command and query handlers
- Test error cases and edge conditions
- Aim for meaningful coverage, not just high percentages

### Running Tests
```bash
# Run all tests
go test ./...

# Run tests with coverage
go test -cover ./...

# Run tests for a specific package
go test ./internal/commands/...
```

## Building and Linting

### Building
```bash
# Build the API server
go build -o keyline ./cmd/api

# Build the queue worker
go build -o queue-worker ./cmd/queueWorker
```

### Linting
The project uses golangci-lint with configuration in `.golangci.yml`:
```bash
# Run linter
golangci-lint run

# Run linter with auto-fix
golangci-lint run --fix
```

## Database and Migrations

### Migrations
- Migrations are in `internal/database/migrations/`
- Use `rubenv/sql-migrate` for migration management
- Migrations run automatically on startup
- Follow naming convention: `YYYYMMDDHHMMSS_description.sql`

### Repository Pattern
- All database access goes through repositories
- Use `go-sqlbuilder` for query construction
- Keep SQL logic in repositories, not in handlers or commands
- Always use parameterized queries to prevent SQL injection

## Security Considerations

### Authentication and Authorization
- Use Argon2id for password hashing
- Support JWT signing with RS256 and EdDSA
- Implement proper RBAC (Role-Based Access Control)
- Always validate permissions before operations
- Use middleware for authentication checks

### Input Validation
- Validate all user inputs
- Use `go-playground/validator` for struct validation
- Sanitize inputs to prevent injection attacks
- Return appropriate error messages without leaking sensitive info

### Secrets Management
- Never commit secrets to the repository
- Use environment variables or configuration files (gitignored)
- Support key storage backends (directory-based, OpenBao)
- Rotate keys periodically

## API Development

### HTTP Handlers
- Keep handlers thin - delegate to commands/queries
- Use proper HTTP status codes
- Return consistent JSON responses
- Document APIs with Swagger annotations
- Handle errors gracefully

### Swagger Documentation
- Add Swagger comments to all API endpoints
- Use `@Summary`, `@Description`, `@Tags`, `@Accept`, `@Produce`, `@Param`, `@Success`, `@Failure`, `@Security`
- Generate docs with: `go generate ./...` (uses `//go:generate` directives in the codebase)
- Access docs at: `/swagger/index.html`

### Request/Response
- Use DTOs (Data Transfer Objects) for request/response
- Validate request bodies before processing
- Return meaningful error messages
- Use appropriate status codes (200, 201, 400, 401, 403, 404, 500)

## Multi-Tenancy (Virtual Servers)

- Each virtual server is isolated
- Users, applications, roles, and settings are per virtual server
- Always scope queries by virtual server ID
- Validate virtual server access in handlers

## Dependencies and Imports

### Adding Dependencies
```bash
# Add a new dependency
go get github.com/package/name

# Tidy up dependencies
go mod tidy
```

### Import Grouping
Group imports in this order:
1. Standard library
2. External packages
3. Internal packages

```go
import (
    "context"
    "fmt"
    
    "github.com/google/uuid"
    "github.com/gorilla/mux"
    
    "Keyline/internal/commands"
    "github.com/The127/mediatr"
)
```

## Configuration

- Configuration is managed by Koanf
- Support both YAML files and environment variables
- Environment variables use `KEYLINE_` prefix
- Keep sensitive values out of config files
- Document all configuration options

## Logging

- Use structured logging with Uber Zap
- Log at appropriate levels: Debug, Info, Warn, Error
- Include context in log messages (user ID, request ID, etc.)
- Don't log sensitive information (passwords, tokens)
- Use log fields for structured data

```go
logger.Info("user registered",
    zap.String("userId", userId.String()),
    zap.String("email", email),
)
```

## Common Patterns

### Creating a New Command
1. Define the command struct in `internal/commands/`
2. Implement the command handler
3. Register the handler with the mediator
4. Add validation if needed
5. Write tests for the command handler

### Creating a New Query
1. Define the query struct in `internal/queries/`
2. Implement the query handler
3. Register the handler with the mediator
4. Write tests for the query handler

### Adding a New API Endpoint
1. Add the handler function in `internal/handlers/`
2. Register the route in the router setup
3. Add Swagger documentation
4. Implement using commands/queries through the mediator
5. Add authentication/authorization middleware if needed
6. Write tests

## Performance Considerations

- Use connection pooling for database and Redis
- Cache frequently accessed data when appropriate
- Use pagination for large result sets
- Optimize SQL queries (avoid N+1 problems)
- Profile code to identify bottlenecks
- Use background jobs for long-running operations

## Development Workflow

1. Create a feature branch
2. Write failing tests first (TDD approach recommended)
3. Implement the feature
4. Run tests: `go test ./...`
5. Run linter: `golangci-lint run`
6. Build the application: `go build ./cmd/api`
7. Test manually if needed
8. Commit with descriptive messages
9. Open a Pull Request

## Docker and Deployment

- Use `docker-compose.yml` for local development
- Build with `Containerfile` (Podman/Docker compatible)
- Always use TLS/HTTPS in production
- Configure proper key storage (consider OpenBao)
- Set up database backups
- Monitor metrics via Prometheus endpoints

## Project-Specific Knowledge

### Key Concepts
- **Virtual Servers**: Multi-tenancy support
- **Applications**: OAuth2/OIDC clients (public, confidential, system)
- **Roles and Permissions**: RBAC system with groups
- **User Types**: Regular users, service users, system users
- **Custom Claims Mapping**: Transform roles to JWT claims using JavaScript

### Important Files
- `config.yaml`: Default configuration
- `.golangci.yml`: Linter configuration
- `go.mod`: Go module dependencies
- `cmd/api/main.go`: API server entry point
- `cmd/queueWorker/main.go`: Background job processor
- `internal/database/migrations/`: Database migrations

## Troubleshooting

### Common Issues
- **Build failures**: Run `go mod tidy` and ensure Go 1.25+ is installed
- **Test failures**: Check database connection and Redis availability
- **Migration issues**: Ensure PostgreSQL is running and accessible
- **Import errors**: Run `go mod download` to fetch dependencies

### Debug Mode
- Enable debug logging in configuration
- Use Go debugger (delve) for complex issues
- Check logs in structured format
- Use Swagger UI to test API endpoints
