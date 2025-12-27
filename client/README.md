# Keyline API Client

The Keyline API Client is a Go package that provides a convenient, type-safe way to interact with the Keyline API. It's designed to simplify building custom tools or integrations.

## Overview

The client is built on three core components:

1. **Client** - Main entry point that provides access to resource-specific clients
2. **Transport** - Handles HTTP communication, request/response processing, and virtual server routing
3. **Resource Clients** - Specialized clients for different API resources (e.g., ApplicationClient)

## Installation

The client is part of the Keyline repository and can be used directly:

```go
import "Keyline/client"
```

## Quick Start

### Basic Usage

```go
package main

import (
	"Keyline/client"
	"Keyline/internal/handlers"
	"context"
	"fmt"
)

func main() {
	// Create a new client
	c := client.NewClient(
		"http://localhost:8081", // Base URL of Keyline API
		"my-virtual-server",     // Virtual server name
	)

	ctx := context.Background()

	// Create an application
	app, err := c.Application().Create(ctx, handlers.CreateApplicationRequestDto{
		Name:           "my-app",
		DisplayName:    "My Application",
		RedirectUris:   []string{"http://localhost:3000/callback"},
		PostLogoutUris: []string{"http://localhost:3000/logout"},
		Type:           "public",
	})
	if err != nil {
		panic(err)
	}

	fmt.Printf("Created application with ID: %s\n", app.Id)
}
```

## Architecture

### Transport Layer

The `Transport` handles low-level HTTP communication:

- **URL Construction**: Automatically constructs full URLs with virtual server routing
- **Request Building**: Creates properly formatted HTTP requests
- **Error Handling**: Converts HTTP errors into Go errors
- **Customization**: Supports custom HTTP clients and middleware via options

### Virtual Server Routing

All API requests are automatically routed through the virtual server path:

```
Base URL: http://localhost:8081
Virtual Server: my-virtual-server
Endpoint: /applications

Result: http://localhost:8081/api/virtual-servers/my-virtual-server/applications
```

## Client Options

The client supports several configuration options:

### Custom HTTP Client

Provide your own `http.Client` for custom timeouts, TLS configuration, etc.:

```go
import (
    "net/http"
    "time"
)

httpClient := &http.Client{
    Timeout: 30 * time.Second,
}

c := client.NewClient(
    "http://localhost:8081",
    "my-virtual-server",
    client.WithClient(httpClient),
)
```

### Custom Round Tripper (Middleware)

Add authentication, logging, or other middleware:

```go
// Authentication middleware
authMiddleware := func(next http.RoundTripper) http.RoundTripper {
    return roundTripperFunc(func(req *http.Request) (*http.Response, error) {
        // Add bearer token
        req.Header.Set("Authorization", "Bearer "+token)
        return next.RoundTrip(req)
    })
}

c := client.NewClient(
    "http://localhost:8081",
    "my-virtual-server",
    client.WithRoundTripper(authMiddleware),
)

// Helper type for function-based round trippers
type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(r *http.Request) (*http.Response, error) {
    return f(r)
}
```

### Custom Base URL

Override the base URL at runtime:

```go
c := client.NewClient(
    "http://localhost:8081",
    "my-virtual-server",
    client.WithBaseURL("https://api.example.com"),
)
```

## Error Handling

The client returns structured errors that can be inspected:

```go
app, err := c.Application().Create(ctx, createDto)
if err != nil {
    // Check for API errors
    if apiErr, ok := err.(client.ApiError); ok {
        fmt.Printf("API Error: %s (HTTP %d)\n", apiErr.Message, apiErr.Code)
        
        switch apiErr.Code {
        case 401:
            // Handle unauthorized
        case 403:
            // Handle forbidden
        case 404:
            // Handle not found
        default:
            // Handle other errors
        }
    } else {
        // Handle other types of errors (network, etc.)
        fmt.Printf("Error: %v\n", err)
    }
}
```

## Complete Example with Authentication

Here's a complete example showing authentication and error handling:

```go
package main

import (
    "context"
    "fmt"
    "net/http"
    
    "Keyline/client"
    "Keyline/internal/handlers"
)

func main() {
    // Create authenticated client
    token := "your-bearer-token"
    
    c := client.NewClient(
        "http://localhost:8081",
        "my-virtual-server",
        client.WithRoundTripper(authMiddleware(token)),
    )

    ctx := context.Background()

    // Create an application
    app, err := c.Application().Create(ctx, handlers.CreateApplicationRequestDto{
        Name:           "example-app",
        DisplayName:    "Example Application",
        RedirectUris:   []string{"http://localhost:3000/callback"},
        PostLogoutUris: []string{"http://localhost:3000/logout"},
        Type:           "public",
    })
    if err != nil {
        handleError(err)
        return
    }

    fmt.Printf("✓ Created application: %s (ID: %s)\n", app.Name, app.Id)

    // List all applications
    apps, err := c.Application().List(ctx, client.ListApplicationParams{
        Page: 1,
        Size: 10,
    })
    if err != nil {
        handleError(err)
        return
    }

    fmt.Printf("✓ Found %d applications\n", len(apps.Items))
    for _, app := range apps.Items {
        fmt.Printf("  - %s: %s\n", app.Name, app.DisplayName)
    }
}

// authMiddleware adds bearer token authentication (just a readme example, not for production use)
func authMiddleware(token string) client.TransportOptions {
    return client.WithRoundTripper(func(next http.RoundTripper) http.RoundTripper {
        return roundTripperFunc(func(req *http.Request) (*http.Response, error) {
            req.Header.Set("Authorization", "Bearer "+token)
            return next.RoundTrip(req)
        })
    })
}

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(r *http.Request) (*http.Response, error) {
    return f(r)
}

func handleError(err error) {
    if apiErr, ok := err.(client.ApiError); ok {
        fmt.Printf("✗ API Error: %s (HTTP %d)\n", apiErr.Message, apiErr.Code)
    } else {
        fmt.Printf("✗ Error: %v\n", err)
    }
}
```

## Future Extensions

The client is designed to be extensible. Additional resource clients can be added by:

1. Creating a new interface in the client package
2. Implementing the interface with a struct that uses the Transport
3. Adding a method to the main Client interface to access the new resource client

Example structure for adding a User client:

```go
type UserClient interface {
    Create(ctx context.Context, dto handlers.CreateUserRequestDto) (handlers.CreateUserResponseDto, error)
    Get(ctx context.Context, id uuid.UUID) (handlers.GetUserResponseDto, error)
    // ... other methods
}

// In client.go
func (c *client) User() UserClient {
    return NewUserClient(c.transport)
}
```

## Best Practices

1. **Always use context**: Pass a proper context for cancellation and timeout support
2. **Handle errors properly**: Check for both API errors and network errors
3. **Reuse clients**: Create one client instance and reuse it across requests
4. **Use authentication middleware**: Add bearer tokens or basic auth via round trippers
5. **Test with the client**: Use it in integration tests for realistic API interactions
6. **Configure timeouts**: Set appropriate HTTP client timeouts for your use case

## Related Documentation

- [E2E Tests Documentation](../tests/e2e/README.md) - Learn how the client is used in end-to-end tests
- [API Documentation](http://localhost:8081/swagger/index.html) - Complete API reference
- [Main README](../README.md) - Project overview and setup

## Package Structure

```
client/
├── README.md          # This file
├── client.go          # Main client interface
├── transport.go       # HTTP transport layer
├── application.go     # Application resource client
└── application_test.go # Unit tests
```

## Contributing

When adding new resource clients:

1. Create the interface in a new file (e.g., `user.go`)
2. Implement all CRUD operations following the Application client pattern
3. Add comprehensive unit tests using httptest
4. Update the main Client interface to expose the new resource client
5. Document the new client in this README

---

For questions or issues, please refer to the [main project documentation](../README.md) or open an issue on GitHub.
