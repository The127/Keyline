# Keyline

[![License: AGPL v3](https://img.shields.io/badge/License-AGPL%20v3-blue.svg)](https://www.gnu.org/licenses/agpl-3.0)
[![Go Version](https://img.shields.io/badge/Go-1.24-00ADD8?logo=go)](https://go.dev/)
[![Go CI](https://github.com/The127/Keyline/actions/workflows/go.yml/badge.svg)](https://github.com/The127/Keyline/actions/workflows/go.yml)

**Keyline** is an open-source OpenID Connect (OIDC) / Identity Provider (IDP) server built with Go. It provides a robust, scalable authentication and authorization solution for modern applications.

The goal is to create an open-source, self-hostable, lightweight, fast, secure, feature rich (but in a good opinionated way), easily configurable and developer friendly OIDC server.

Keyline is still under active development and not ready for production use. Consider it a very feature-rich late alpha release.
Keyline does pass the basic OIDC basic server conformance tests (oidcc-basic-certification-test-plan) for features we support (no address or phone scope and no requested claims) on a local dev machine, but it is not yet production ready.
For future major releases, we will be including a link to the conformance test results in this README.

## Features

- 🔐 **OpenID Connect (OIDC) Provider** - Full OIDC implementation for authentication
- 👥 **User Management** - Complete user lifecycle management with registration, verification, and password reset
- 🎭 **Role-Based Access Control (RBAC)** - Fine-grained permissions with roles and groups
- 🔑 **Multiple Application Support** - Manage multiple client applications (public and confidential)
- 🎨 **Custom Claims Mapping** - Transform roles into custom JWT claims using JavaScript
- 📧 **Email Integration** - Built-in email verification and notification system (work-in-progress)
- 🔒 **Multi-Factor Authentication (MFA)** - TOTP-based 2FA support
- 🏢 **Virtual Servers** - Multi-tenancy support via virtual servers
- 📝 **Template System** - Customizable email templates
- 📊 **Audit Logging** - Comprehensive audit trail for security and compliance
- 🔄 **Session Management** - Secure session handling with Redis support
- 🪪 **Flexible Key Storage** - In-memory (testing), directory-based, or OpenBao (work-in-progress)
- 💾 **Flexible Cache Layer** - in-memory for dev, Redis for production
- 🗄️ **Configurable Database** - PostgreSQL for production, SQLite for development/single-server (work-in-progress)
- 🎯 **Service Users** - Support for service accounts with public key authentication
- 📦 **User Metadata** - Store custom user and application-specific metadata
- 📈 **Metrics & Monitoring** - Prometheus metrics integration
- 🐳 **Container Ready** - Docker/Podman support with provided Containerfile
- ⚖️ **Leader Election** - Raft-based leader election for high-availability multi-instance deployments

## Architecture

Keyline follows a clean architecture pattern with clear separation of concerns:

- **Handlers** - HTTP request handlers and routing
- **Commands/Queries** - CQRS pattern for business logic
- **Repositories** - Data access layer
- **Services** - Core business services
- **Mediator** - Request/event mediator pattern for decoupled communication
- **IoC Container** - Custom dependency injection container

### 📚 New to Keyline?

Check out our comprehensive **[Onboarding Guide](docs/onboarding/README.md)** to understand the architecture, design patterns, and development workflows. The guide covers:

- **Architecture Overview** - Understand clean architecture and folder structure
- **CQRS & Mediator Pattern** - Learn the core communication patterns
- **Dependency Injection** - Master the IoC container
- **Development Workflow** - Get started with development
- **Testing Guide** - Write effective tests

Perfect for new contributors and developers wanting to understand Keyline's architecture!

## Prerequisites

- **Go 1.24** or higher
- **Database** - PostgreSQL (recommended for production) or SQLite (work-in-progress, for development/single-server)
- **Cache Storage** - Redis (Valkey) recommended for production, or in-memory for development/single-instance
- **Mail server** (for email notifications)

## Quick Start

Keyline uses [just](https://github.com/casey/just) as a command runner for development tasks. Install it first:

```bash
# macOS
brew install just

# Linux
# On Ubuntu/Debian
apt install just

# Other methods: https://github.com/casey/just#installation
```

### 1. Clone the Repository

```bash
git clone https://github.com/The127/Keyline.git
cd Keyline
```

### 2. Start Dependencies with Docker Compose

```bash
docker compose up -d
```

This will start:
- PostgreSQL on port 5732
- Mailpit (mail server) on ports 1025 (SMTP) and 8025 (Web UI)
- Redis (Valkey) on port 6379
- RabbitMQ on ports 5672 and 15672 (management UI)

### 3. Configure the Application

Copy the configuration template and customize it:

```bash
cp config.yaml.template config.yaml
```

Edit `config.yaml` with your settings. Key configuration sections:

#### Server Configuration
```yaml
server:
  host: "127.0.0.1"
  port: 8081
  externalUrl: "http://127.0.0.1:8081"
```

#### Database Configuration
```yaml
database:
  mode: "postgres"  # "postgres" or "sqlite" (sqlite is work-in-progress)
  postgres:
    host: "localhost"
    port: 5732
    username: "user"
    password: "password"
    sslMode: "disable"
  # For SQLite (work-in-progress):
  # sqlite:
  #   database: "./keyline.db"
```

#### Initial Virtual Server
```yaml
initialVirtualServer:
  name: "default"
  displayName: "Default Server"
  enableRegistration: true
  signingAlgorithm: "RS256"  # or "EdDSA"
  createInitialAdmin: true
  initialAdmin:
    username: admin
    displayName: Admin
    primaryEmail: admin@example.com
    passwordHash: "$argon2id$v=19$m=16,t=2,p=1$..."
  # Optional: Pre-configure applications with roles
  initialApplications:
    - name: "my-app"
      type: "public"
      redirectUris: ["http://localhost:3000/callback"]
      roles:
        - name: "user"
          description: "Regular user"
  # Optional: Define global roles
  initialRoles:
    - name: "viewer"
      description: "Can view resources"
  # Optional: Create service users for machine-to-machine auth
  initialServiceUsers:
    - username: "api-service"
      publicKey: "-----BEGIN PUBLIC KEY-----\n..."
      roles: ["viewer", "my-app user"]
```

For detailed configuration options including service users, application roles, and global roles, see the [Configuration Package Documentation](internal/config/README.md).

#### Cache Configuration
```yaml
cache:
  mode: "redis"  # or "memory" for single-instance deployments
  redis:
    host: "localhost"
    port: 6379
```

#### Key Store Configuration
```yaml
keyStore:
  mode: "directory"  # "memory" (testing only), "directory", or "openbao"
  directory:
    path: "./keys"
```

**Note:** Use `mode: "memory"` only for testing/development - keys are lost on restart.

#### Leader Election Configuration
```yaml
leaderElection:
  mode: "none"  # "none" for single instance, "raft" for multi-instance with leader election
  # Raft configuration (for high-availability deployments):
  # raft:
  #   host: "0.0.0.0"
  #   port: 7000
  #   id: "keyline-node-1"
  #   initiatorId: "keyline-node-1"
  #   nodes:
  #     - id: "keyline-node-1"
  #       address: "node1.internal:7000"
  #     - id: "keyline-node-2"
  #       address: "node2.internal:7000"
  #     - id: "keyline-node-3"
  #       address: "node3.internal:7000"
```

**Leader Election Modes:**
- **none** (default) - Single instance deployment, always acts as leader
- **raft** - Distributed leader election for multi-instance deployments using HashiCorp Raft

When using Raft mode, only the elected leader executes background jobs (key rotation, outbox processing), while all instances serve HTTP requests. This enables high-availability deployments with automatic failover.

For detailed leader election configuration, see the [Configuration Package Documentation](internal/config/README.md#leader-election-configuration).

### 4. Run Database Migrations

Migrations are automatically run on startup. The application will create all necessary tables and initial data.

### 5. Build and Run

Using just (recommended):

```bash
# Build and run the application
just run
```

Or manually:

```bash
# Build the application
go build -o keyline ./cmd/api

# Run the application
./keyline
```

### 6. Access the Application

- **API**: http://localhost:8081
- **API Documentation**: http://localhost:8081/swagger/index.html
- **Mailpit UI**: http://localhost:8025

## Admin UI

Keyline has a separate web UI for administration available at [KeylineUI](https://github.com/The127/KeylineUi).

### Running KeylineUI with Docker

The UI is available as a container image: `ghcr.io/the127/keyline-ui:v0.1.2`

Example docker-compose configuration:

```yaml
keyline-ui:
  image: ghcr.io/the127/keyline-ui:v0.1.2
  container_name: keyline-ui
  restart: unless-stopped
  environment:
    KEYLINE_API_URL: "https://api.sso.example.com"  # URL to your Keyline API
    KEYLINE_HOST: "https://sso.example.com"          # Public URL for the UI
  ports:
    - "3000:80"  # Map to your desired port
```

**Environment Variables:**
- `KEYLINE_API_URL` - The URL where your Keyline API is accessible
- `KEYLINE_HOST` - The public URL where the UI will be accessed

## Integration Examples

Keyline includes example applications demonstrating how to integrate with various frameworks:

- **[Java Spring Example](docs/examples/java-spring/)** - A Spring Boot application demonstrating OAuth2 resource server integration with Keyline using JWT authentication. This example shows how to configure Spring Security to validate access tokens issued by Keyline.

## Configuration

Keyline provides flexible configuration management through YAML files and environment variables. Configuration supports multiple database backends (PostgreSQL, SQLite work-in-progress), cache backends (in-memory or Redis) and key storage options (directory or OpenBao).

**Quick Start:**
1. Copy `config.yaml.template` to `config.yaml` and customize it
2. Alternatively, use environment variables with `KEYLINE_` prefix to override settings

Example environment variables:
```bash
KEYLINE_SERVER_HOST=0.0.0.0
KEYLINE_SERVER_PORT=8080
KEYLINE_DATABASE_MODE=postgres
KEYLINE_DATABASE_POSTGRES_HOST=localhost
KEYLINE_DATABASE_POSTGRES_PASSWORD=secret
KEYLINE_CACHE_MODE=redis
KEYLINE_CACHE_REDIS_HOST=localhost
```

For comprehensive configuration documentation, including all available options, defaults, and validation rules, see the [Configuration Package Documentation](internal/config/README.md).

## API Documentation

Keyline provides comprehensive API documentation using Swagger/OpenAPI:

1. Start the application
2. Navigate to http://localhost:8081/swagger/index.html
3. Use the "Authorize" button to authenticate with Bearer tokens or Basic Auth

## Development

### Project Structure

```
.
├── cmd/api/              # Application entry point
├── internal/
│   ├── authentication/   # Authentication middleware and utilities
│   ├── commands/         # CQRS command handlers
│   ├── queries/          # CQRS query handlers
│   ├── handlers/         # HTTP handlers
│   ├── repositories/     # Data access layer
│   ├── services/         # Business services
│   ├── database/         # Database connection and migrations
│   ├── config/           # Configuration management (see internal/config/README.md)
│   ├── middlewares/      # HTTP middlewares
│   └── ...
├── ioc/                  # IoC container implementation (see ioc/Readme.md)
├── mediator/             # Mediator pattern implementation (see mediator/README.md)
├── client/               # API client library (see client/README.md)
├── tests/
│   ├── e2e/              # End-to-end tests (see tests/e2e/README.md)
│   └── integration/      # Integration tests
├── utils/                # Utility functions
├── docs/                 # Swagger documentation
├── templates/            # Email templates
├── justfile              # Development task runner
└── config.yaml.template  # Configuration file template
```

### Using Just for Development

Keyline uses [just](https://github.com/casey/just) as a command runner. Available commands:

```bash
# List all available commands
just --list

# Build the application
just build

# Build and run the application
just run

# Run unit tests
just test

# Run integration tests
just integration

# Format code
just fmt

# Run linter (check only)
just lint

# Run linter with auto-fix
just lint fix

# Run all CI checks (format, lint, test, integration)
just ci

# Run all CI checks with auto-fix
just ci fix

# Clean build artifacts
just clean
```

### Running Tests

Using just (recommended):

```bash
# Run all unit tests
just test

# Run integration tests
just integration

# Run end-to-end tests
just e2e

# Run all tests with CI checks
just ci
```

Or manually:

```bash
# Run all tests
go test ./...

# Run tests with coverage
go test -cover ./...

# Run tests for a specific package
go test ./ioc/...

# Run end-to-end tests
go test -tags=e2e ./tests/e2e/...
```

For detailed information about testing:
- [E2E Tests Documentation](tests/e2e/README.md) - Complete guide to end-to-end testing
- [API Client Documentation](client/README.md) - Learn how to use the Keyline API client

### Linting

Using just (recommended):

```bash
# Check for linting issues
just lint

# Auto-fix linting issues
just lint fix
```

Or manually:

```bash
# Run golangci-lint
golangci-lint run
```

### Building with Docker

```bash
# Build the container image
docker build -f Containerfile -t keyline:latest .

# Run the container
docker run -p 8080:8080 \
  -v $(pwd)/config.yaml:/app/config.yaml \
  -v $(pwd)/keys:/app/keys \
  keyline:latest
```

### Using Pre-built Container Images

Keyline provides pre-built container images on GitHub Container Registry (GHCR):

**Dev Builds** (from main branch):
- `ghcr.io/the127/keyline:latest` - Always points to the most recent dev build from the main branch
- `ghcr.io/the127/keyline:dev-<build-number>` - Specific dev build by CI run number (e.g., `dev-123`)
- `ghcr.io/the127/keyline:dev-<commit-sha>` - Specific dev build by commit SHA (e.g., `dev-abc123def`)

Dev builds are automatically created on every push to the main branch. Each push creates three tags: `latest` (updated to point to the new build), `dev-<build-number>`, and `dev-<commit-sha>`.

**Image Retention Policy:**
- **Dev builds**: Cleaned up automatically after 10 builds or 7 days (whichever comes first)
- **`latest` tag**: Never cleaned up - always points to the most recent dev build
- **Release builds** (`v*` tags): Permanently preserved, never cleaned up

**Release Builds** (from version tags):
- `ghcr.io/the127/keyline:v1.0.0` - Specific release version (e.g., v1.0.0, v1.2.3)
- Created automatically when a version tag (e.g., `v1.0.0`) is pushed
- Permanently preserved for production use

Example usage:
```bash
# Use the latest dev build (for testing/development)
docker pull ghcr.io/the127/keyline:latest

# Use a specific dev build by run number
docker pull ghcr.io/the127/keyline:dev-123

# Use a specific release version (recommended for production)
docker pull ghcr.io/the127/keyline:v1.0.0

# Run with a specific version
docker run -p 8080:8080 \
  -v $(pwd)/config.yaml:/app/config.yaml \
  -v $(pwd)/keys:/app/keys \
  ghcr.io/the127/keyline:v1.0.0
```

## Key Concepts

### Virtual Servers

Virtual servers enable multi-tenancy, allowing you to host multiple isolated identity providers within a single Keyline instance. Each virtual server has its own:
- Users
- Applications
- Roles and permissions
- Configuration settings

### Applications

Applications represent OAuth2/OIDC clients that integrate with Keyline. Supported application types:
- **Public** - For client-side applications (SPAs, mobile apps)
- **Confidential** - For server-side applications with client secrets
- **System** - For internal system operations

### Roles and Permissions

Keyline implements a comprehensive RBAC system:
- **Roles** - Named collections of permissions
- **Groups** - User collections for bulk role assignment
- **Permissions** - Fine-grained access control for specific operations
- **Role Assignments** - Link users to roles (optionally scoped to applications)

### User Types

- **Regular Users** - Standard user accounts
- **Service Users** - Non-interactive accounts for machine-to-machine authentication
- **System Users** - Built-in accounts for internal operations

### Custom Claims Mapping

Keyline allows you to transform user roles and metadata into custom JWT claims using JavaScript. Each application can define its own claims mapping script that runs during token generation.

**Available Variables in Mapping Scripts:**

- `roles` - Array of global role names assigned to the user
- `applicationRoles` - Array of application-specific role names assigned to the user
- `globalMetadata` - Object containing user's global metadata (shared across all applications)
- `appMetadata` - Object containing user's application-specific metadata

**Example Mapping Script:**

```javascript
// Transform roles and metadata into custom claims
({
  "custom_roles": roles.concat(applicationRoles),
  "department": globalMetadata.department || "unknown",
  "app_settings": appMetadata.settings || {},
  "is_admin": roles.includes("admin")
})
```

**Default Behavior:**

If no custom mapping script is defined, the default mapping returns:
```javascript
{
  "roles": roles,
  "application_roles": applicationRoles
}
```

The mapping script is set per application and can be updated via the API.

### User Metadata

Keyline supports storing custom metadata for users at two levels:

1. **Global Metadata** - User-level metadata shared across all applications (stored in `users.metadata`)
2. **Application-Specific Metadata** - Per-user, per-application metadata (stored in `application_user_metadata`)

Both metadata fields are JSON objects that can store arbitrary key-value pairs. They are accessible in claims mapping scripts as `globalMetadata` and `appMetadata` respectively.

**Note:** Keyline does not provide a way to filter or query users by metadata values. Metadata is designed for storing user attributes that are included in JWT claims, not for user search or filtering purposes. If you need to filter users by specific attributes, consider using roles or dedicated user fields instead.

## IoC Container

Keyline includes a custom IoC (Inversion of Control) container with support for:
- **Transient** - New instance per resolution
- **Scoped** - Single instance per scope
- **Singleton** - Single instance for application lifetime

See [ioc/Readme.md](ioc/Readme.md) for detailed documentation.

## Mediator Pattern

Keyline uses the mediator pattern to decouple components and implement CQRS (Command Query Responsibility Segregation):
- **Handlers** - Process requests and return responses
- **Behaviors** - Cross-cutting concerns like validation and logging
- **Events** - Publish/subscribe pattern for notifications

See [mediator/README.md](mediator/README.md) for detailed documentation.

## Security

### Password Policies

Keyline enforces comprehensive password validation policies to ensure user passwords meet security requirements:

- **Configurable Policies** - Minimum/maximum length, character type requirements (digits, uppercase, lowercase, special characters)
- **Common Password Protection** - Built-in check against ~100,000 most commonly used passwords
- **Per-Tenant Configuration** - Different password requirements per virtual server
- **Clear Error Messages** - Helpful feedback to guide users in creating secure passwords

For detailed information about password policies, configuration options, and best practices, see the [Password Policies Documentation](docs/password-policies.md).

### Password Hashing

Keyline uses Argon2id for secure password hashing, which is resistant to:
- GPU cracking attacks
- Side-channel attacks
- Time-memory trade-off attacks

### Token Signing

JWT tokens are signed using configurable algorithms:
- RS256 (RSA 2048-bit keys)
- EdDSA (Ed25519 keys)

Keys are automatically generated and rotated as needed.

### Multi-Factor Authentication

TOTP-based 2FA using standard authenticator apps (Google Authenticator, Authy, etc.).

## Deployment

### Production Considerations

1. **Use HTTPS** - Always use TLS in production
2. **Database Choice** - Use PostgreSQL for production deployments (SQLite is for development only)
3. **Secure Key Storage** - Consider using OpenBao or similar for key management
4. **Database Backups** - Regular backups of PostgreSQL database
5. **Distributed Caching** - Use Redis cache mode for multi-instance deployments (in-memory cache is not shared)
6. **Redis Persistence** - Configure Redis persistence for cache data if using Redis mode
7. **Log Aggregation** - Centralize logs for monitoring and debugging
8. **Metrics** - Monitor Prometheus metrics for performance insights
9. **Rate Limiting** - Implement rate limiting at the reverse proxy level
10. **High Availability** - Use Raft leader election mode for multi-instance deployments with automatic failover

### Deployment Architectures

#### Single Instance Deployment

For development, testing, or small-scale production:

```yaml
leaderElection:
  mode: "none"  # Single instance, always acts as leader
cache:
  mode: "memory"  # Or "redis" for persistence
database:
  mode: "postgres"  # Or "sqlite" for single-node only
```

**Characteristics:**
- Simple setup and maintenance
- All background jobs run on the single instance
- No leader election overhead
- Suitable for low-traffic applications

#### High-Availability Multi-Instance Deployment

For production environments requiring fault tolerance and load distribution:

```yaml
leaderElection:
  mode: "raft"
  raft:
    host: "0.0.0.0"
    port: 7000
    id: "keyline-node-1"  # Unique per instance
    initiatorId: "keyline-node-1"  # Same on all instances
    nodes:
      - id: "keyline-node-1"
        address: "node1.internal:7000"
      - id: "keyline-node-2"
        address: "node2.internal:7000"
      - id: "keyline-node-3"
        address: "node3.internal:7000"
cache:
  mode: "redis"  # Required for multi-instance
  redis:
    host: "redis.internal"
    port: 6379
database:
  mode: "postgres"  # Required for multi-instance
  postgres:
    host: "postgres.internal"
```

**Characteristics:**
- Automatic leader election and failover
- Only the leader executes background jobs (key rotation, outbox processing)
- All instances serve HTTP requests
- Minimum 3 nodes recommended (tolerates 1 failure)
- 5 nodes recommended for high availability (tolerates 2 failures)
- Requires Redis for shared cache and PostgreSQL for shared database

**Benefits:**
- **Zero Downtime**: Rolling updates without service interruption
- **Automatic Failover**: If leader fails, cluster elects new leader within seconds
- **Load Distribution**: HTTP traffic distributed across all instances
- **Job Safety**: Background jobs execute on exactly one instance

**Network Requirements:**
- All instances must communicate on the Raft port (default 7000)
- Load balancer for HTTP traffic distribution
- Shared PostgreSQL database accessible from all instances
- Shared Redis cache accessible from all instances

### Environment Variables for Production

```bash
KEYLINE_SERVER_EXTERNALURL=https://auth.example.com
KEYLINE_DATABASE_MODE=postgres
KEYLINE_DATABASE_POSTGRES_HOST=prod-db.example.com
KEYLINE_DATABASE_POSTGRES_PASSWORD=secure-password
KEYLINE_CACHE_MODE=redis
KEYLINE_CACHE_REDIS_HOST=prod-redis.example.com
KEYLINE_KEYSTORE_MODE=openbao

# For multi-instance with leader election:
KEYLINE_LEADERELECTION_MODE=raft
KEYLINE_LEADERELECTION_RAFT_HOST=0.0.0.0
KEYLINE_LEADERELECTION_RAFT_PORT=7000
KEYLINE_LEADERELECTION_RAFT_ID=keyline-node-1  # Unique per instance
KEYLINE_LEADERELECTION_RAFT_INITIATORID=keyline-node-1  # Same on all instances
```

## Contributing

Contributions are welcome! Please follow these guidelines:

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Write tests for your changes
4. Run tests and linting (`go test ./...` and `golangci-lint run`)
5. Commit your changes (`git commit -m 'Add amazing feature'`)
6. Push to the branch (`git push origin feature/amazing-feature`)
7. Open a Pull Request

### Developer Certificate of Origin

All commits to this repository must be signed off using the `-s` flag:

```bash
git commit -s -m "Your commit message"
```

Alternatively, your IDE may support signing off commits.
Sign-off is verified in GitHub Actions.

By signing off, you agree to the [Developer Certificate of Origin](DCO.md).

## License

This project is licensed under the GNU Affero General Public License v3.0 (AGPL-3.0). See the [LICENSE](LICENSE) file for details.

## Support

- **Issues**: [GitHub Issues](https://github.com/The127/Keyline/issues)
- **Discussions**: [GitHub Discussions](https://github.com/The127/Keyline/discussions)

## Acknowledgments

Built with:
- [Gorilla Mux](https://github.com/gorilla/mux) - HTTP router
- [sqlbuilder](https://github.com/huandu/go-sqlbuilder) - SQL query builder
- [Koanf](https://github.com/knadh/koanf) - Configuration management
- [Zap](https://github.com/uber-go/zap) - Structured logging
- [jwt-go](https://github.com/golang-jwt/jwt) - JWT implementation
- [Swagger](https://github.com/swaggo/swag) - API documentation
- [SecLists](https://github.com/danielmiessler/SecLists) - Common password list for password validation

---

Made with ❤️ by the Keyline contributors
