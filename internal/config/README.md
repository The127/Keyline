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
KEYLINE_DATABASE_MODE=postgres
KEYLINE_DATABASE_POSTGRES_HOST=db.example.com
KEYLINE_DATABASE_POSTGRES_PASSWORD=secret

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
  apiPort: 8081                       # API server port (for running the api on a different port)
  externalUrl: "https://auth.example.com"  # Public-facing URL
  allowedOrigins:                     # CORS allowed origins
    - "https://app.example.com"
    - "https://admin.example.com"
```

**Defaults:**
- **host**: `localhost` (DEVELOPMENT), *required* (PRODUCTION)
- **port**: `8080`
- **apiPort**: `0` (runs on the same port as the server/no separate API server)
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

Keyline supports multiple database backends with a mode-based configuration.

```yaml
database:
  mode: "postgres"  # Database mode: "postgres" or "sqlite"
  postgres:
    host: "localhost"        # Database host
    port: 5432              # Database port
    database: "keyline"     # Database name
    username: "keyline"     # Database username
    password: "secret"      # Database password
    sslMode: "require"      # SSL mode: disable, require, verify-ca, verify-full
  sqlite:
    database: "./keyline.db"  # SQLite database file path
```

**Database Modes:**

#### PostgreSQL Mode
Production-ready relational database suitable for multi-instance deployments.

```yaml
database:
  mode: "postgres"
  postgres:
    host: "postgres.example.com"
    port: 5432
    database: "keyline"
    username: "keyline_user"
    password: "secure_password"
    sslMode: "require"
```

**Defaults (PostgreSQL mode):**
- **database**: `keyline`
- **port**: `5432`
- **sslMode**: `enable`
- **host**, **username**, **password**: *required*

#### SQLite Mode (Work in Progress)
Lightweight file-based database suitable for development and single-server deployments.

```yaml
database:
  mode: "sqlite"
  sqlite:
    database: "./data/keyline.db"
```

**Defaults (SQLite mode):**
- **database**: *required* (path to SQLite database file)

**Note:** SQLite support is currently work in progress and not yet fully implemented.

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
  mode: "directory"  # Key store mode: "memory", "directory", or "openbao"
  directory:
    path: "./keys"   # Directory path for key storage
  # openbao:
  #   # OpenBao configuration (work in progress)
```

**Key Store Modes:**

#### Memory Mode (Testing Only)
Store keys in memory. **Only for testing and development** - keys are lost on restart.

```yaml
keyStore:
  mode: "memory"
```

**Warning:** This mode stores keys only in application memory. All keys will be lost when the application restarts. This mode is intended **only for testing and development purposes** and should **never be used in production**.

No additional configuration needed.

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

### Leader Election Configuration

Configure leader election for multi-instance deployments. Leader election ensures only one instance executes background jobs like key rotation and outbox message processing.

```yaml
leaderElection:
  mode: "none"  # Leader election mode: "none" or "raft"
  raft:
    host: "0.0.0.0"           # Raft bind address
    port: 7000                # Raft communication port
    id: "node1"               # Unique node identifier
    initiatorId: "node1"      # ID of the node that bootstraps the cluster
    nodes:                    # All nodes in the Raft cluster
      - id: "node1"
        address: "node1:7000"
      - id: "node2"
        address: "node2:7000"
      - id: "node3"
        address: "node3:7000"
```

**Leader Election Modes:**

#### None Mode (Default)
Single-instance deployment where the current instance is always the leader.

```yaml
leaderElection:
  mode: "none"
```

Use this mode when:
- Running a single instance of Keyline
- Testing or development environments
- You don't need high availability

No additional configuration needed. The instance will always execute background jobs.

#### Raft Mode
Distributed leader election using HashiCorp Raft consensus algorithm for multi-instance deployments.

```yaml
leaderElection:
  mode: "raft"
  raft:
    host: "0.0.0.0"           # Address to bind Raft server
    port: 7000                # Port for Raft communication
    id: "keyline-node-1"      # Unique identifier for this node
    initiatorId: "keyline-node-1"  # Node that bootstraps the cluster
    nodes:
      - id: "keyline-node-1"
        address: "10.0.1.10:7000"
      - id: "keyline-node-2"
        address: "10.0.1.11:7000"
      - id: "keyline-node-3"
        address: "10.0.1.12:7000"
```

**Raft Configuration Options:**

- **host**: IP address or hostname where this node's Raft server binds (required)
- **port**: TCP port for Raft inter-node communication (required)
- **id**: Unique identifier for this Keyline instance in the cluster (required)
- **initiatorId**: The node ID that will bootstrap the Raft cluster on first startup (required)
  - Should be set to the same value on all nodes
  - Only the node with this ID will initialize the cluster
- **nodes**: List of all nodes in the Raft cluster (required)
  - **id**: Unique identifier for the node (must match the node's configured `id`)
  - **address**: Network address where the node can be reached (host:port format)

**Raft Mode Requirements:**

- Minimum of 3 nodes recommended for fault tolerance (can tolerate 1 failure)
- 5 nodes recommended for high availability (can tolerate 2 failures)
- All nodes must be able to communicate with each other on the Raft port
- The `nodes` list should be identical across all instances
- Only the node with ID matching `initiatorId` will bootstrap the cluster

**How It Works:**

1. When started, Keyline nodes form a Raft cluster
2. The cluster automatically elects one node as the leader
3. Only the leader instance executes background jobs:
   - Outbox message processing
   - Signing key rotation
4. If the leader fails, the cluster automatically elects a new leader
5. Non-leader instances continue serving HTTP requests normally

**Use Cases:**

- **High Availability**: Multiple Keyline instances with automatic failover
- **Load Balancing**: Distribute HTTP traffic across multiple instances
- **Zero Downtime**: Rolling updates without interrupting background jobs

**Example Multi-Instance Setup:**

For a 3-node cluster, configure each node differently:

**Node 1 Configuration:**
```yaml
leaderElection:
  mode: "raft"
  raft:
    host: "0.0.0.0"
    port: 7000
    id: "keyline-node-1"
    initiatorId: "keyline-node-1"
    nodes:
      - id: "keyline-node-1"
        address: "keyline-1.internal:7000"
      - id: "keyline-node-2"
        address: "keyline-2.internal:7000"
      - id: "keyline-node-3"
        address: "keyline-3.internal:7000"
```

**Node 2 Configuration:**
```yaml
leaderElection:
  mode: "raft"
  raft:
    host: "0.0.0.0"
    port: 7000
    id: "keyline-node-2"
    initiatorId: "keyline-node-1"  # Same initiator
    nodes:
      - id: "keyline-node-1"
        address: "keyline-1.internal:7000"
      - id: "keyline-node-2"
        address: "keyline-2.internal:7000"
      - id: "keyline-node-3"
        address: "keyline-3.internal:7000"
```

**Node 3 Configuration:**
```yaml
leaderElection:
  mode: "raft"
  raft:
    host: "0.0.0.0"
    port: 7000
    id: "keyline-node-3"
    initiatorId: "keyline-node-1"  # Same initiator
    nodes:
      - id: "keyline-node-1"
        address: "keyline-1.internal:7000"
      - id: "keyline-node-2"
        address: "keyline-2.internal:7000"
      - id: "keyline-node-3"
        address: "keyline-3.internal:7000"
```

**Defaults:**
- **mode**: *required* (`none` or `raft`)

**Raft Mode Defaults:**
- All Raft configuration fields are required when `mode` is set to `raft`

### Initial Virtual Server Configuration

Configure the initial virtual server (tenant) created on first startup.

```yaml
initialVirtualServer:
  name: "default"                      # Internal name (URL-safe)
  displayName: "Default Server"        # Display name
  enableRegistration: true             # Allow user self-registration
  signingAlgorithm: "EdDSA"           # JWT signing: "EdDSA" or "RS256"
  createSystemAdminRole: true         # Create system admin role (can create and manage virtual servers)
  createAdmin: true                   # Create initial admin user
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
      roles:                           # Application-specific roles (optional)
        - name: "user"
          description: "Regular user role"
        - name: "admin"
          description: "Administrator role"
  initialRoles:                        # Global roles (optional)
    - name: "viewer"
      description: "Can view resources"
    - name: "editor"
      description: "Can edit resources"
  initialServiceUsers:                 # Service accounts (optional)
    - username: "api-service"
      publicKey: "-----BEGIN PUBLIC KEY-----\n..."  # PEM-encoded public key
      roles:
        - "viewer"                     # Global role assignment
        - "my-app admin"               # Application-specific role (format: "app-name role-name")
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
- Application role names are required if roles are specified

**Initial Roles:**

Initial roles can be defined at two levels:

1. **Global Roles** (`initialRoles`): Virtual server-wide roles that can be assigned to any user
2. **Application-Specific Roles** (`initialApplications[].roles`): Roles scoped to a specific application

Both types support:
- **name**: Role identifier (required)
- **description**: Human-readable description (optional)

**Initial Service Users:**

Service users are non-interactive accounts for machine-to-machine authentication using public key cryptography.

Configuration:
- **username**: Service user identifier (required)
- **publicKey**: PEM-encoded public key for authentication (required)
- **roles**: Array of role assignments (optional)

**Role Assignment Format:**
- **Global role**: Just the role name (e.g., `"viewer"`)
- **Application-specific role**: Format `"application-name role-name"` (e.g., `"my-app admin"`)

Service users authenticate by signing JWT tokens with their private key, verified against the configured public key.

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
  mode: "postgres"
  postgres:
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
      roles:
        - name: "user"
          description: "Regular application user"
        - name: "admin"
          description: "Application administrator"
    - name: "mobile-app"
      displayName: "Mobile Application"
      type: "public"
      redirectUris:
        - "com.example.app://callback"
      postLogoutRedirectUris:
        - "com.example.app://logout"
      roles:
        - name: "user"
          description: "Mobile app user"
  initialRoles:
    - name: "viewer"
      description: "Can view all resources"
    - name: "editor"
      description: "Can edit resources"
  initialServiceUsers:
    - username: "api-gateway"
      publicKey: |
        -----BEGIN PUBLIC KEY-----
        MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEA...
        -----END PUBLIC KEY-----
      roles:
        - "viewer"              # Global role
        - "web-app user"        # Application-specific role
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
  mode: "postgres"
  postgres:
    host: "localhost"
    port: 5732
    username: "dev"
    password: "dev"
    sslMode: "disable"

cache:
  mode: "memory"  # Use in-memory cache for development

keyStore:
  mode: "memory"  # Use in-memory key store for testing (keys lost on restart)
  # Or use directory mode for persistent keys:
  # mode: "directory"
  # directory:
  #   path: "./keys"

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

## Advanced Configuration Examples

### Example 1: Multi-Application Setup with Roles

This example shows how to set up multiple applications with their own roles, plus global roles and service users:

```yaml
initialVirtualServer:
  name: "company"
  displayName: "Company SSO"
  enableRegistration: false
  signingAlgorithm: "EdDSA"
  createInitialAdmin: true
  initialAdmin:
    username: "admin"
    displayName: "System Administrator"
    primaryEmail: "admin@company.com"
    passwordHash: "$argon2id$v=19$m=65536,t=3,p=4$..."
  
  # Define multiple applications with their specific roles
  initialApplications:
    - name: "crm"
      displayName: "Customer Relationship Management"
      type: "confidential"
      hashedSecret: "$argon2id$v=19$m=65536,t=3,p=4$..."
      redirectUris:
        - "https://crm.company.com/auth/callback"
      postLogoutRedirectUris:
        - "https://crm.company.com/logout"
      roles:
        - name: "sales-rep"
          description: "Sales representative with customer access"
        - name: "sales-manager"
          description: "Sales manager with team oversight"
        - name: "admin"
          description: "CRM administrator"
    
    - name: "analytics"
      displayName: "Analytics Dashboard"
      type: "public"
      redirectUris:
        - "https://analytics.company.com/callback"
      postLogoutRedirectUris:
        - "https://analytics.company.com"
      roles:
        - name: "viewer"
          description: "Can view reports and dashboards"
        - name: "analyst"
          description: "Can create and edit reports"
    
    - name: "api"
      displayName: "Public API"
      type: "confidential"
      hashedSecret: "$argon2id$v=19$m=65536,t=3,p=4$..."
      redirectUris:
        - "https://api.company.com/oauth/callback"
      postLogoutRedirectUris:
        - "https://api.company.com"
      roles:
        - name: "read"
          description: "Read-only API access"
        - name: "write"
          description: "Read and write API access"
  
  # Define global roles that apply across all applications
  initialRoles:
    - name: "employee"
      description: "All company employees"
    - name: "contractor"
      description: "External contractors"
    - name: "auditor"
      description: "Security and compliance auditor"
  
  # Set up service users for automated systems
  initialServiceUsers:
    - username: "monitoring-service"
      publicKey: |
        -----BEGIN PUBLIC KEY-----
        MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEA...
        -----END PUBLIC KEY-----
      roles:
        - "auditor"              # Global role for audit access
        - "analytics viewer"     # Can view analytics
    
    - username: "data-pipeline"
      publicKey: |
        -----BEGIN PUBLIC KEY-----
        MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEA...
        -----END PUBLIC KEY-----
      roles:
        - "api write"            # Can write to API
        - "crm sales-rep"        # Can access CRM as sales rep
  
  mail:
    host: "smtp.company.com"
    port: 587
    username: "noreply@company.com"
    password: "smtp-password"
```

### Example 2: Microservices Architecture

Setting up service users for a microservices architecture where each service needs specific permissions:

```yaml
initialVirtualServer:
  name: "microservices"
  displayName: "Microservices Platform"
  enableRegistration: false
  signingAlgorithm: "EdDSA"
  createInitialAdmin: true
  initialAdmin:
    username: "admin"
    primaryEmail: "admin@platform.com"
    passwordHash: "$argon2id$v=19$m=65536,t=3,p=4$..."
  
  initialApplications:
    - name: "user-service"
      displayName: "User Management Service"
      type: "confidential"
      hashedSecret: "$argon2id$v=19$m=65536,t=3,p=4$..."
      redirectUris:
        - "http://user-service:8080/callback"
      postLogoutRedirectUris:
        - "http://user-service:8080"
      roles:
        - name: "admin"
          description: "User service administrator"
    
    - name: "payment-service"
      displayName: "Payment Processing Service"
      type: "confidential"
      hashedSecret: "$argon2id$v=19$m=65536,t=3,p=4$..."
      redirectUris:
        - "http://payment-service:8080/callback"
      postLogoutRedirectUris:
        - "http://payment-service:8080"
      roles:
        - name: "processor"
          description: "Can process payments"
        - name: "refunder"
          description: "Can issue refunds"
  
  initialRoles:
    - name: "service-account"
      description: "Internal service account"
  
  initialServiceUsers:
    - username: "api-gateway"
      publicKey: |
        -----BEGIN PUBLIC KEY-----
        MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEA...
        -----END PUBLIC KEY-----
      roles:
        - "service-account"
        - "user-service admin"
        - "payment-service processor"
    
    - username: "billing-service"
      publicKey: |
        -----BEGIN PUBLIC KEY-----
        MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEA...
        -----END PUBLIC KEY-----
      roles:
        - "service-account"
        - "payment-service processor"
        - "payment-service refunder"
```

### Example 3: Development Environment with Minimal Security

For local development with simplified configuration:

```yaml
initialVirtualServer:
  name: "dev"
  displayName: "Development"
  enableRegistration: true
  signingAlgorithm: "EdDSA"
  createInitialAdmin: true
  initialAdmin:
    username: "admin"
    primaryEmail: "admin@localhost"
    passwordHash: "$argon2id$v=19$m=16,t=2,p=1$MTIzNDU2Nzg$QWu7e+sjG5knAdNLoKdLDg"
  
  initialApplications:
    - name: "dev-app"
      displayName: "Development Application"
      type: "public"
      redirectUris:
        - "http://localhost:3000/callback"
      postLogoutRedirectUris:
        - "http://localhost:3000"
      roles:
        - name: "developer"
          description: "Developer access"
  
  initialRoles:
    - name: "tester"
      description: "QA tester"
  
  initialServiceUsers:
    - username: "test-service"
      publicKey: 
        kdi: "35cf4fea-62c6-4023-898a-ffa6d9c0a8f3"
        pem: |
            -----BEGIN PUBLIC KEY-----
            MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEA...
            -----END PUBLIC KEY-----
      roles:
        - "tester"
        - "dev-app developer"
```

### Understanding Role Assignment

**Global Roles vs Application Roles:**

- **Global Roles** (defined in `initialRoles`): Can be assigned to any user and provide permissions across the entire virtual server
- **Application Roles** (defined in `initialApplications[].roles`): Scoped to a specific application and only provide permissions within that application context

**Service User Role Assignment:**

When assigning roles to service users in the `roles` array:

Use the format `"project-name:role-name"`
   ```yaml
   roles:
     - "my-app:admin"
     - "api-service:write"
   ```

The system automatically detects the format based on whether the role string contains a space. If it contains a space, it's treated as an application-specific role where the first part is the application name and the second part is the role name.

### Public Key Format

Service user public keys must be in PEM format. You can generate a key pair using OpenSSL:

```bash
# Generate private key
openssl genrsa -out private_key.pem 2048

# Extract public key
openssl rsa -in private_key.pem -pubout -out public_key.pem

# View the public key (copy this to your config)
cat public_key.pem
```

The public key in the configuration should look like:
```yaml
publicKey: 
  pem: |
      -----BEGIN PUBLIC KEY-----
      MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEAw8vLPQXHEYmVHN...
      YourActualKeyDataHere...
      -----END PUBLIC KEY-----
  kid: <some-uuid>
```

### Password Hash Generation

Initial admin passwords must be Argon2id hashes. For production use strong parameters:

```bash
# Production settings (m=65536, t=3, p=4)
# This is more secure but slower

# Development settings (m=16, t=2, p=1) - faster but less secure
# Example hash: $argon2id$v=19$m=16,t=2,p=1$MTIzNDU2Nzg$QWu7e+sjG5knAdNLoKdLDg
```

Use Keyline's built-in utilities or a compatible Argon2id tool to generate these hashes.

## Dependencies

The config package uses [Koanf](https://github.com/knadh/koanf) for configuration management:

- YAML parsing
- Environment variable mapping
- Nested configuration support
- Type-safe unmarshaling
