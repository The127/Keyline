# Configuration Package

The config package provides flexible configuration management for Keyline using YAML files and environment variables. It supports multiple configuration sources with environment-specific defaults and validation.

## Features

- **Multiple Configuration Sources** - Load from YAML files and environment variables
- **Environment-Specific Defaults** - Different defaults for PRODUCTION and DEVELOPMENT
- **Validation** - Automatic validation of required fields with panic on missing values
- **Type-Safe** - Strongly-typed configuration structure
- **Flexible Cache Modes** - Support for in-memory and Redis caching
- **Pluggable Key Storage** - Directory-based or OpenBao key storage options

## Configuration Sources

Configuration is loaded in the following order (later sources override earlier ones):

1. **YAML Configuration File** - Specified via `--config` flag
2. **Environment Variables** - Prefixed with `KEYLINE_` and using underscores for nesting

### Environment Variables

Environment variables are automatically mapped to configuration keys:

```bash
# Server configuration
KEYLINE_SERVER_HOST=0.0.0.0
KEYLINE_SERVER_PORT=8080
KEYLINE_SERVER_EXTERNALURL=https://auth.example.com

# Database configuration
KEYLINE_DATABASE_HOST=db.example.com
KEYLINE_DATABASE_PASSWORD=secret

# Cache configuration
KEYLINE_CACHE_MODE=redis
KEYLINE_CACHE_REDIS_HOST=redis.example.com

# Arrays are supported using space-separated values
KEYLINE_SERVER_ALLOWEDORIGINS="https://app1.com https://app2.com"
```

**Note:** Nested keys use underscores (`_`) in environment variables which are converted to dots (`.`) internally.

## Configuration Structure

### Server Configuration

Controls the HTTP server settings and external URLs.

```yaml
server:
  host: "0.0.0.0"                    # Server bind address
  port: 8080                          # Server port
  externalUrl: "https://auth.example.com"  # Public-facing URL
  allowedOrigins:                     # CORS allowed origins
    - "https://app.example.com"
    - "https://admin.example.com"
```

**Defaults:**
- **host**: `localhost` (DEVELOPMENT), *required* (PRODUCTION)
- **port**: `8080`
- **externalUrl**: `{host}:{port}` (DEVELOPMENT), *required* (PRODUCTION)
- **allowedOrigins**: `["*", "http://localhost:5173"]` (DEVELOPMENT), *required* (PRODUCTION)

### Frontend Configuration

Specifies the frontend application URL for redirects and CORS.

```yaml
frontend:
  externalUrl: "https://ui.example.com"  # Frontend public URL
```

**Defaults:**
- **externalUrl**: `http://localhost:5173` (DEVELOPMENT), *required* (PRODUCTION)

### Database Configuration

PostgreSQL database connection settings.

```yaml
database:
  host: "localhost"        # Database host
  port: 5432              # Database port
  database: "keyline"     # Database name
  username: "keyline"     # Database username
  password: "secret"      # Database password
  sslMode: "require"      # SSL mode: disable, require, verify-ca, verify-full
```

**Defaults:**
- **database**: `keyline`
- **port**: `5432`
- **sslMode**: `enable`
- **host**, **username**, **password**: *required*

### Cache Configuration

Configure caching strategy: in-memory or Redis-based.

```yaml
cache:
  mode: "redis"  # Cache mode: "memory" or "redis"
  redis:
    host: "localhost"     # Redis host
    port: 6379           # Redis port
    username: ""         # Redis username (optional)
    password: ""         # Redis password (optional)
    database: 0          # Redis database number
```

**Cache Modes:**

#### Memory Mode
In-memory caching suitable for single-instance deployments or development.

```yaml
cache:
  mode: "memory"
```

No additional configuration needed. Cache data is stored in application memory and will be lost on restart.

#### Redis Mode
Distributed caching using Redis, suitable for multi-instance production deployments.

```yaml
cache:
  mode: "redis"
  redis:
    host: "redis.example.com"
    port: 6379
    password: "redis-secret"
    database: 0
```

**Defaults (Redis mode):**
- **host**: `localhost` (DEVELOPMENT), *required* (PRODUCTION)
- **port**: `6379`
- **username**, **password**: optional
- **database**: `0`

### Key Store Configuration

Manage cryptographic keys for token signing.

```yaml
keyStore:
  mode: "directory"  # Key store mode: "directory" or "openbao"
  directory:
    path: "./keys"   # Directory path for key storage
  # openbao:
  #   # OpenBao configuration (work in progress)
```

**Key Store Modes:**

#### Directory Mode
Store keys as files in a directory. Suitable for single-instance or development setups.

```yaml
keyStore:
  mode: "directory"
  directory:
    path: "/var/lib/keyline/keys"
```

**Defaults:**
- **directory.path**: *required*

#### OpenBao Mode
Store keys in OpenBao for enhanced security in production environments.

```yaml
keyStore:
  mode: "openbao"
  openbao:
    # Configuration work in progress
```

**Note:** OpenBao mode is not yet implemented.

### Initial Virtual Server Configuration

Configure the initial virtual server (tenant) created on first startup.

```yaml
initialVirtualServer:
  name: "default"                      # Internal name (URL-safe)
  displayName: "Default Server"        # Display name
  enableRegistration: true             # Allow user self-registration
  signingAlgorithm: "EdDSA"           # JWT signing: "EdDSA" or "RS256"
  createInitialAdmin: true             # Create initial admin user
  initialAdmin:
    username: "admin"                  # Admin username
    displayName: "Administrator"       # Admin display name
    primaryEmail: "admin@example.com"  # Admin email (required)
    passwordHash: "$argon2id$v=19..."  # Argon2id password hash (required)
  initialApplications:                 # Pre-configured applications
    - name: "my-app"
      displayName: "My Application"
      type: "public"                   # "public" or "confidential"
      redirectUris:
        - "https://app.example.com/callback"
      postLogoutRedirectUris:
        - "https://app.example.com"
  mail:
    host: "smtp.example.com"           # SMTP host
    port: 587                          # SMTP port
    username: "noreply@example.com"    # SMTP username
    password: "smtp-secret"            # SMTP password
```

**Signing Algorithms:**
- **EdDSA**: Ed25519 elliptic curve (recommended, faster, smaller keys)
- **RS256**: RSA 2048-bit (traditional, widely compatible)

**Application Types:**
- **public**: Client-side applications (SPAs, mobile apps) - no secret required
- **confidential**: Server-side applications - requires `hashedSecret`

**Defaults:**
- **name**: `keyline`
- **displayName**: `Keyline`
- **signingAlgorithm**: `EdDSA`
- **initialAdmin.username**: `admin`
- **initialAdmin.displayName**: `Administrator`
- **initialAdmin.primaryEmail**: *required if createInitialAdmin is true*
- **initialAdmin.passwordHash**: *required if createInitialAdmin is true*

**Application Validation:**
- Name is required
- Type must be "public" or "confidential"
- Confidential apps must have a non-empty `hashedSecret`

## Usage

### Initialization

The config package is initialized at application startup:

```go
import "Keyline/internal/config"

func main() {
    // Initialize configuration
    config.Init()
    
    // Access configuration
    dbHost := config.C.Database.Host
    serverPort := config.C.Server.Port
    
    // Check environment
    if config.IsProduction() {
        // Production-specific logic
    }
}
```

### Command-Line Flags

```bash
# Specify custom config file
./keyline --config /etc/keyline/config.yaml

# Set environment (default: PRODUCTION)
./keyline --environment DEVELOPMENT --config config.yaml
```

**Flags:**
- `--config`: Path to YAML configuration file (optional)
- `--environment`: Environment mode - `PRODUCTION` or `DEVELOPMENT` (default: `PRODUCTION`)

### Environment Detection

```go
// Check if running in production
if config.IsProduction() {
    // Use strict validation
    // Require all production settings
}
```

## Complete Configuration Example

### YAML File Example

```yaml
server:
  host: "0.0.0.0"
  port: 8080
  externalUrl: "https://auth.example.com"
  allowedOrigins:
    - "https://app.example.com"
    - "https://admin.example.com"

frontend:
  externalUrl: "https://ui.example.com"

database:
  host: "postgres.example.com"
  port: 5432
  database: "keyline"
  username: "keyline_user"
  password: "secure_db_password"
  sslMode: "require"

cache:
  mode: "redis"
  redis:
    host: "redis.example.com"
    port: 6379
    password: "redis_password"
    database: 0

keyStore:
  mode: "directory"
  directory:
    path: "/var/lib/keyline/keys"

initialVirtualServer:
  name: "production"
  displayName: "Production Server"
  enableRegistration: false
  signingAlgorithm: "EdDSA"
  createInitialAdmin: true
  initialAdmin:
    username: "admin"
    displayName: "Administrator"
    primaryEmail: "admin@example.com"
    passwordHash: "$argon2id$v=19$m=65536,t=3,p=4$..."
  initialApplications:
    - name: "web-app"
      displayName: "Web Application"
      type: "confidential"
      hashedSecret: "$argon2id$v=19$m=65536,t=3,p=4$..."
      redirectUris:
        - "https://app.example.com/auth/callback"
      postLogoutRedirectUris:
        - "https://app.example.com"
    - name: "mobile-app"
      displayName: "Mobile Application"
      type: "public"
      redirectUris:
        - "com.example.app://callback"
      postLogoutRedirectUris:
        - "com.example.app://logout"
  mail:
    host: "smtp.example.com"
    port: 587
    username: "noreply@example.com"
    password: "smtp_password"
```

### Development Configuration Example

```yaml
server:
  host: "127.0.0.1"
  port: 8081

database:
  host: "localhost"
  port: 5732
  username: "dev"
  password: "dev"
  sslMode: "disable"

cache:
  mode: "memory"  # Use in-memory cache for development

keyStore:
  mode: "directory"
  directory:
    path: "./keys"

initialVirtualServer:
  enableRegistration: true
  createInitialAdmin: true
  initialAdmin:
    primaryEmail: "admin@localhost"
    passwordHash: "$argon2id$v=19$m=16,t=2,p=1$..."
  mail:
    host: "localhost"
    port: 1025  # Mailpit or similar
```

## Validation and Defaults

The config package automatically:

1. **Validates required fields** based on environment
2. **Sets sensible defaults** for optional fields
3. **Panics on invalid configuration** to fail fast during startup
4. **Provides environment-specific defaults**:
   - Development: Relaxed requirements, localhost defaults
   - Production: Strict validation, no insecure defaults

### Production Requirements

When running with `--environment PRODUCTION`, the following are required:

- Server external URL
- Server allowed origins (no wildcards recommended)
- Frontend external URL
- Database credentials (host, username, password)
- Redis host (if using Redis cache mode)
- Initial admin email and password hash (if creating initial admin)
- Key store configuration

### Error Handling

Invalid configuration results in a panic with descriptive error messages:

```
panic: missing database host
panic: cache mode missing or not supported
panic: missing key store directory path
panic: missing initial admin primary email
```

This fail-fast approach ensures configuration errors are caught immediately at startup rather than causing runtime failures.

## Security Considerations

1. **Never commit secrets** - Use environment variables for sensitive values
2. **Use Argon2id hashes** - For passwords and secrets (use Keyline's hashing utilities)
3. **Enable SSL in production** - Set `database.sslMode` to `require` or higher
4. **Restrict CORS origins** - Avoid wildcards in production `allowedOrigins`
5. **Use Redis for distributed deployments** - Memory cache is not shared across instances
6. **Secure key storage** - Ensure `keyStore.directory.path` has appropriate permissions

## Migration Notes

### Redis to Cache Structure

The configuration structure has been updated to support multiple cache backends:

**Old structure:**
```yaml
redis:
  host: "localhost"
  port: 6379
```

**New structure:**
```yaml
cache:
  mode: "redis"
  redis:
    host: "localhost"
    port: 6379
```

**To migrate:**
1. Add `cache.mode: "redis"` to your configuration
2. Move Redis settings under `cache.redis`
3. Or switch to `cache.mode: "memory"` for single-instance deployments

## Related Documentation

- [Main Project README](../../README.md) - Overall project documentation
- [IoC Container](../../ioc/Readme.md) - Dependency injection system
- [Mediator Pattern](../../mediator/README.md) - CQRS implementation

## Dependencies

The config package uses [Koanf](https://github.com/knadh/koanf) for configuration management:

- YAML parsing
- Environment variable mapping
- Nested configuration support
- Type-safe unmarshaling
