# Keyline

[![License: AGPL v3](https://img.shields.io/badge/License-AGPL%20v3-blue.svg)](https://www.gnu.org/licenses/agpl-3.0)
[![Go Version](https://img.shields.io/badge/Go-1.24-00ADD8?logo=go)](https://go.dev/)

**Keyline** is an open-source OpenID Connect (OIDC) / Identity Provider (IDP) server built with Go. It provides a robust, scalable authentication and authorization solution for modern applications.

Keyline is still under active development and not ready for production use. Consider it a very feature-rich late alpha release.

## Features

- 🔐 **OpenID Connect (OIDC) Provider** - Full OIDC implementation for authentication
- 👥 **User Management** - Complete user lifecycle management with registration, verification, and password reset
- 🎭 **Role-Based Access Control (RBAC)** - Fine-grained permissions with roles and groups
- 🔑 **Multiple Application Support** - Manage multiple client applications (public and confidential)
- 📧 **Email Integration** - Built-in email verification and notification system (work-in-progress)
- 🔒 **Multi-Factor Authentication (MFA)** - TOTP-based 2FA support
- 🏢 **Virtual Servers** - Multi-tenancy support via virtual servers
- 📝 **Template System** - Customizable email templates
- 📊 **Audit Logging** - Comprehensive audit trail for security and compliance
- 🔄 **Session Management** - Secure session handling with Redis support
- 🔐 **Flexible Key Storage** - Support for directory-based key stores (OpenBao support work-in-progress)
- 🎯 **Service Users** - Support for service accounts with public key authentication
- 📦 **User Metadata** - Store custom user and application-specific metadata
- 🔧 **Built-in IoC Container** - Lightweight dependency injection with transient, scoped, and singleton lifetimes
- 📈 **Metrics & Monitoring** - Prometheus metrics integration
- 🐳 **Container Ready** - Docker/Podman support with provided Containerfile

## Architecture

Keyline follows a clean architecture pattern with clear separation of concerns:

- **Handlers** - HTTP request handlers and routing
- **Commands/Queries** - CQRS pattern for business logic
- **Repositories** - Data access layer
- **Services** - Core business services
- **Mediator** - Request/event mediator pattern for decoupled communication
- **IoC Container** - Custom dependency injection container

## Prerequisites

- **Go 1.24** or higher
- **PostgreSQL** database
- **Redis** (Valkey) for session storage
- **Mail server** (for email notifications)

## Quick Start

### 1. Clone the Repository

```bash
git clone https://github.com/The127/Keyline.git
cd Keyline
```

### 2. Start Dependencies with Docker Compose

```bash
docker-compose up -d
```

This will start:
- PostgreSQL on port 5732
- Mailpit (mail server) on ports 1025 (SMTP) and 8025 (Web UI)
- Redis (Valkey) on port 6379
- RabbitMQ on ports 5672 and 15672 (management UI)

### 3. Configure the Application

Copy and customize the configuration file:

```bash
cp config.yaml config.local.yaml
```

Edit `config.local.yaml` with your settings. Key configuration sections:

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
  host: "localhost"
  port: 5732
  username: "user"
  password: "password"
  sslMode: "disable"
```

#### Initial Virtual Server
```yaml
initialVirtualServer:
  name: "default"
  displayName: "Default Server"
  enableRegistration: true
  signingAlgorithm: "RS256"
  createInitialAdmin: true
  initialAdmin:
    username: admin
    displayName: Admin
    primaryEmail: admin@example.com
    passwordHash: "$argon2id$v=19$m=16,t=2,p=1$..."
```

#### Key Store Configuration
```yaml
keyStore:
  mode: "directory"  # or "openbao"
  directory:
    path: "./keys"
```

### 4. Run Database Migrations

Migrations are automatically run on startup. The application will create all necessary tables and initial data.

### 5. Build and Run

```bash
# Build the application
go build -o keyline ./cmd/api

# Run the application
./keyline --config config.local.yaml
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

## Configuration

Configuration can be provided via:
1. YAML configuration file (default: `config.yaml`)
2. Environment variables with `KEYLINE_` prefix

Example environment variables:
```bash
KEYLINE_SERVER_HOST=0.0.0.0
KEYLINE_SERVER_PORT=8080
KEYLINE_DATABASE_HOST=localhost
KEYLINE_DATABASE_PASSWORD=secret
```

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
│   ├── config/           # Configuration management
│   ├── middlewares/      # HTTP middlewares
│   └── ...
├── ioc/                  # IoC container implementation
├── mediator/             # Mediator pattern implementation
├── utils/                # Utility functions
├── docs/                 # Swagger documentation
├── templates/            # Email templates
└── config.yaml           # Configuration file
```

### Running Tests

```bash
# Run all tests
go test ./...

# Run tests with coverage
go test -cover ./...

# Run tests for a specific package
go test ./ioc/...
```

### Linting

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

## IoC Container

Keyline includes a custom IoC (Inversion of Control) container with support for:
- **Transient** - New instance per resolution
- **Scoped** - Single instance per scope
- **Singleton** - Single instance for application lifetime

See [ioc/Readme.md](ioc/Readme.md) for detailed documentation.

## Security

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
2. **Secure Key Storage** - Consider using OpenBao or similar for key management
3. **Database Backups** - Regular backups of PostgreSQL database
4. **Redis Persistence** - Configure Redis persistence for session data
5. **Log Aggregation** - Centralize logs for monitoring and debugging
6. **Metrics** - Monitor Prometheus metrics for performance insights
7. **Rate Limiting** - Implement rate limiting at the reverse proxy level

### Environment Variables for Production

```bash
KEYLINE_SERVER_EXTERNALURL=https://auth.example.com
KEYLINE_DATABASE_HOST=prod-db.example.com
KEYLINE_DATABASE_PASSWORD=secure-password
KEYLINE_REDIS_HOST=prod-redis.example.com
KEYLINE_KEYSTORE_MODE=openbao
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

## License

This project is licensed under the GNU Affero General Public License v3.0 (AGPL-3.0). See the [LICENSE](LICENSE) file for details.

## Support

- **Issues**: [GitHub Issues](https://github.com/The127/Keyline/issues)
- **Discussions**: [GitHub Discussions](https://github.com/The127/Keyline/discussions)

## Acknowledgments

Built with:
- [Gorilla Mux](https://github.com/gorilla/mux) - HTTP router
- [sqlbuilder](https://github.com/huandu/go-sqlbuilder) - SQL query builder
- [Viper](https://github.com/spf13/viper) - Configuration management
- [Zap](https://github.com/uber-go/zap) - Structured logging
- [jwt-go](https://github.com/golang-jwt/jwt) - JWT implementation
- [Swagger](https://github.com/swaggo/swag) - API documentation

---

Made with ❤️ by the Keyline contributors
