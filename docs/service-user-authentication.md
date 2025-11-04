# Service User Authentication

Service users (also known as service accounts) are a special type of user in Keyline designed for machine-to-machine authentication. Unlike regular users who authenticate with passwords or passkeys, service users authenticate using public/private key pairs and the OAuth 2.0 Token Exchange grant type.

## Overview

Service user authentication in Keyline uses the following flow:

1. **Create a service user** - An administrator creates a service user account
2. **Associate a public key** - Register the service user's public key with Keyline
3. **Create and sign a JWT** - The service application creates a JWT signed with its private key
4. **Exchange for access token** - The signed JWT is exchanged for an access token via the token endpoint

This approach is based on the [OAuth 2.0 Token Exchange (RFC 8693)](https://datatracker.ietf.org/doc/html/rfc8693) specification and is ideal for:

- Service-to-service authentication
- CI/CD pipelines
- Background jobs and workers
- API clients that don't involve user interaction

## Prerequisites

- A virtual server configured in Keyline
- Admin access to create service users
- A key pair (public/private) for the service user (EdDSA/Ed25519 recommended)
- An application (client) registered in Keyline

## Creating a Service User

### Step 1: Create the Service User

Service users are created using the `CreateServiceUser` command. This can be done through the Keyline API or directly if you have admin access.

**Command Structure:**
```go
CreateServiceUser{
    VirtualServerName: "your-virtual-server",
    Username:          "my-service-user",
}
```

**Required Permission:** `ServiceUserCreate`

### Step 2: Associate a Public Key

After creating a service user, you need to associate its public key with the account. The public key should be in PEM format (PKIX).

**Command Structure:**
```go
AssociateServiceUserPublicKey{
    VirtualServerName: "your-virtual-server",
    ServiceUserId:     serviceUserId, // UUID of the service user
    PublicKey:         publicKeyPEM,  // PEM-encoded public key
}
```

**Required Permission:** `ServiceUserAssociateKey`

**Example Public Key Format:**
```
-----BEGIN PUBLIC KEY-----
MCowBQYDK2VwAyEA3M7NYNpucIwsMNDHPswe1yvLtMzIau2ddMB2FX40few=
-----END PUBLIC KEY-----
```

The response will include a Key ID (`kid`) which you'll need when signing JWTs.

## Authentication Flow

### Step 1: Create a Self-Signed JWT

The service application must create a JWT with the following characteristics:

**JWT Header:**
```json
{
  "alg": "EdDSA",
  "typ": "JWT",
  "kid": "<key-id-from-associate-key-response>"
}
```

**JWT Claims:**
```json
{
  "iss": "<service-user-id>",
  "sub": "<service-user-id>",
  "aud": "<target-application-name>",
  "scopes": "openid profile email"
}
```

**Key Requirements:**
- `iss` (issuer) must equal `sub` (subject) - both must be the service user's UUID
- `aud` (audience) must be the name of the target application
- `scopes` must be a space-separated string and must include "openid"
- The JWT must be signed with the private key corresponding to the registered public key
- The `kid` header must match the Key ID returned when associating the public key

**Example in Go:**

```go
import (
    "crypto/x509"
    "encoding/pem"
    "github.com/golang-jwt/jwt/v5"
)

// Decode the private key
block, _ := pem.Decode([]byte(privateKeyPEM))
key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
if err != nil {
    panic(err)
}

// Create JWT claims
claims := jwt.MapClaims{
    "aud":    "my-application",
    "iss":    serviceUserId.String(),
    "sub":    serviceUserId.String(),
    "scopes": "openid profile email",
}

// Create and sign the token
jwtToken := jwt.NewWithClaims(jwt.SigningMethodEdDSA, claims)
jwtToken.Header["kid"] = keyId
signedJWT, err := jwtToken.SignedString(key)
if err != nil {
    panic(err)
}
```

### Step 2: Exchange the JWT for an Access Token

Once you have the signed JWT, exchange it for an access token using the token endpoint with the Token Exchange grant type.

**Endpoint:**
```
POST /oidc/{virtualServerName}/token
Content-Type: application/x-www-form-urlencoded
```

**Request Parameters:**
- `grant_type`: Must be `urn:ietf:params:oauth:grant-type:token-exchange`
- `subject_token`: The signed JWT from Step 1
- `subject_token_type`: Must be `urn:ietf:params:oauth:token-type:access_token`

**Example Request:**

```bash
curl -X POST "https://keyline.example.com/oidc/my-virtual-server/token" \
  -H "Content-Type: application/x-www-form-urlencoded" \
  -d "grant_type=urn:ietf:params:oauth:grant-type:token-exchange" \
  -d "subject_token=${SIGNED_JWT}" \
  -d "subject_token_type=urn:ietf:params:oauth:token-type:access_token"
```

**Example in Go:**

```go
import (
    "net/http"
    "net/url"
)

resp, err := http.PostForm(
    fmt.Sprintf("%s/oidc/%s/token", keylineURL, virtualServer),
    url.Values{
        "grant_type":         {"urn:ietf:params:oauth:grant-type:token-exchange"},
        "subject_token":      {signedJWT},
        "subject_token_type": {"urn:ietf:params:oauth:token-type:access_token"},
    },
)
```

**Successful Response (200 OK):**
```json
{
  "access_token": "eyJhbGciOiJFZERTQSIsInR5cCI6IkpXVCIsImtpZCI6IjEyMzQ1Njc4LTkwYWItY2RlZi0xMjM0LTU2Nzg5MGFiY2RlZiJ9...",
  "issued_token_type": "urn:ietf:params:oauth:token-type:access_token",
  "token_type": "Bearer"
}
```

The returned `access_token` is a JWT that can be used to authenticate API requests to Keyline or other services that accept Keyline tokens.

**Error Response (400 or 401):**
```json
{
  "error": "invalid_grant",
  "error_description": "Invalid subject token"
}
```

## Token Exchange Implementation Details

The token exchange process in Keyline performs the following validations (see `internal/handlers/oidc.go`, `handleTokenExchange` function):

### Validation Steps

1. **Parse and Validate JWT Structure**
   - Extracts the `kid` from JWT header
   - Validates JWT structure and signature

2. **Verify Issuer and Subject Match**
   - Checks that `iss` equals `sub` (self-issued token requirement)
   - This ensures the token is signed by the service user itself

3. **Verify User Exists and is a Service User**
   - Looks up the user by the `sub` claim
   - Confirms the user is marked as a service user
   - Confirms the user belongs to the specified virtual server

4. **Verify Public Key**
   - Retrieves the credential with type `CredentialTypeServiceUserKey`
   - Matches the `kid` with the stored key ID
   - Decodes the stored public key (PEM format)
   - Uses the public key to verify the JWT signature

5. **Validate Claims**
   - Checks `aud` (audience) contains exactly one application name
   - Validates `scopes` claim is present and contains "openid"
   - Validates the target application exists in the virtual server

6. **Generate Access Token**
   - Creates a new JWT access token signed by Keyline's key
   - Token includes user ID, scopes, application ID, and custom claims
   - Token expires after 5 minutes (configurable)

### Security Considerations

- **Private Key Security**: The service user's private key must be kept secure and never committed to source control
- **Key Rotation**: Public keys can be rotated by associating a new key and updating the application to use the new `kid`
- **Token Expiry**: Access tokens are short-lived (5 minutes by default) to limit exposure
- **Scope Validation**: Only requested scopes that are valid for the application are granted
- **Issuer Validation**: The self-issued token pattern ensures only the service user can authenticate as itself

## Complete Example

Here's a complete end-to-end example based on the e2e test in `tests/e2e/serviceuserlogin_test.go`:

```go
package main

import (
    "crypto/x509"
    "encoding/json"
    "encoding/pem"
    "fmt"
    "net/http"
    "net/url"
    
    "github.com/golang-jwt/jwt/v5"
)

const (
    serviceUserPrivateKey = `-----BEGIN PRIVATE KEY-----
MFECAQEwBQYDK2VwBCIEIDlOHCg/gH43TB4S1n/2g33iti99sEkEFYwVdAkyKoqw
gSEA3M7NYNpucIwsMNDHPswe1yvLtMzIau2ddMB2FX40few=
-----END PRIVATE KEY-----`
)

func main() {
    // Configuration
    keylineURL := "https://keyline.example.com"
    virtualServer := "my-virtual-server"
    serviceUserId := "12345678-90ab-cdef-1234-567890abcdef"
    keyId := "abcdefgh-1234-5678-90ab-cdefghijklmn"
    applicationName := "my-application"
    
    // Step 1: Parse the private key
    block, _ := pem.Decode([]byte(serviceUserPrivateKey))
    if block == nil {
        panic("failed to decode PEM")
    }
    
    key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
    if err != nil {
        panic(err)
    }
    
    // Step 2: Create JWT claims
    claims := jwt.MapClaims{
        "aud":    applicationName,
        "iss":    serviceUserId,
        "sub":    serviceUserId,
        "scopes": "openid profile email",
    }
    
    // Step 3: Sign the JWT
    jwtToken := jwt.NewWithClaims(jwt.SigningMethodEdDSA, claims)
    jwtToken.Header["kid"] = keyId
    signedJWT, err := jwtToken.SignedString(key)
    if err != nil {
        panic(err)
    }
    
    // Step 4: Exchange for access token
    resp, err := http.PostForm(
        fmt.Sprintf("%s/oidc/%s/token", keylineURL, virtualServer),
        url.Values{
            "grant_type":         {"urn:ietf:params:oauth:grant-type:token-exchange"},
            "subject_token":      {signedJWT},
            "subject_token_type": {"urn:ietf:params:oauth:token-type:access_token"},
        },
    )
    if err != nil {
        panic(err)
    }
    defer resp.Body.Close()
    
    // Step 5: Parse response
    if resp.StatusCode != http.StatusOK {
        fmt.Printf("Error: received status code %d\n", resp.StatusCode)
        return
    }
    
    var responseJson map[string]interface{}
    err = json.NewDecoder(resp.Body).Decode(&responseJson)
    if err != nil {
        panic(err)
    }
    
    accessToken := responseJson["access_token"].(string)
    fmt.Printf("Successfully obtained access token: %s\n", accessToken)
    
    // Step 6: Use the access token
    req, err := http.NewRequest("GET", fmt.Sprintf("%s/api/some-endpoint", keylineURL), nil)
    if err != nil {
        panic(err)
    }
    req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", accessToken))
    
    // Make authenticated request...
}
```

## Supported Algorithms

Keyline supports the following signing algorithms for service user keys:

- **EdDSA** (Ed25519) - Recommended for new implementations
- **RS256** (RSA with SHA-256) - For compatibility with existing systems

The algorithm is determined by the public key type when you associate the key with the service user.

## Troubleshooting

### "Invalid subject token"

- Verify the JWT is signed with the correct private key
- Ensure the `kid` header matches the key ID from the associate key response
- Check that the JWT structure and claims are correct

### "User is not a service user"

- The user account must be created as a service user, not a regular user
- Verify you're using the correct user ID in the `iss` and `sub` claims

### "Credential not found"

- Ensure the public key has been associated with the service user
- Verify the `kid` in the JWT header matches a registered key
- Check that the key hasn't been revoked

### "Invalid issuer"

- The `iss` claim must equal the `sub` claim (both must be the service user ID)
- This is a security requirement for service user token exchange

### "Application not found"

- Verify the `aud` claim contains the correct application name
- Ensure the application is registered in the same virtual server

## API Reference

### Create Service User Command

**Command:** `CreateServiceUser`

**Parameters:**
- `VirtualServerName` (string) - Name of the virtual server
- `Username` (string) - Username for the service user

**Response:**
- `Id` (UUID) - The service user's unique identifier

**Required Permission:** `ServiceUserCreate`

### Associate Service User Public Key Command

**Command:** `AssociateServiceUserPublicKey`

**Parameters:**
- `VirtualServerName` (string) - Name of the virtual server
- `ServiceUserId` (UUID) - ID of the service user
- `PublicKey` (string) - PEM-encoded public key

**Response:**
- `Id` (UUID) - The key ID to use in JWT `kid` header

**Required Permission:** `ServiceUserAssociateKey`

### Token Exchange Endpoint

**Endpoint:** `POST /oidc/{virtualServerName}/token`

**Content-Type:** `application/x-www-form-urlencoded`

**Parameters:**
- `grant_type` (required) - Must be `urn:ietf:params:oauth:grant-type:token-exchange`
- `subject_token` (required) - The signed JWT
- `subject_token_type` (required) - Must be `urn:ietf:params:oauth:token-type:access_token`

**Success Response (200 OK):**
```json
{
  "access_token": "string",
  "issued_token_type": "urn:ietf:params:oauth:token-type:access_token",
  "token_type": "Bearer"
}
```

**Error Response (400/401):**
```json
{
  "error": "string",
  "error_description": "string"
}
```

## Related Documentation

- [OIDC Discovery](/.well-known/openid-configuration) - OpenID Connect configuration endpoint
- [JWKS Endpoint](/.well-known/jwks.json) - Public keys for token verification
- [OAuth 2.0 Token Exchange (RFC 8693)](https://datatracker.ietf.org/doc/html/rfc8693) - Token exchange specification
- [Testing Guide](onboarding/06-testing-guide.md) - Writing tests for authentication flows

## Source Code References

- **Handler Implementation:** `internal/handlers/oidc.go` - `handleTokenExchange` function (lines 1454-1695)
- **E2E Test:** `tests/e2e/serviceuserlogin_test.go` - Complete working example
- **Create Service User Command:** `internal/commands/CreateServiceUser.go`
- **Associate Key Command:** `internal/commands/AssociateServiceUserPublicKey.go`
- **Test Harness Setup:** `tests/e2e/harness.go` - `initTest` function shows service user setup
