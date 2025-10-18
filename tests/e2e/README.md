# Keyline End-to-End (E2E) Tests

End-to-end tests validate the complete Keyline system by running the actual API server and making real HTTP requests. These tests ensure that all components work together correctly in a production-like environment.

## Overview

The E2E test suite uses:

- **[Ginkgo](https://onsi.github.io/ginkgo/)**: BDD-style testing framework
- **[Gomega](https://onsi.github.io/gomega/)**: Matcher/assertion library
- **Test Harness**: Custom test infrastructure for isolated test environments
- **Keyline API Client**: Type-safe Go client for API interactions

## Architecture

### Test Components

1. **Test Harness** (`harness.go`)
   - Creates isolated test environments for each test suite
   - Manages database lifecycle (create, migrate, cleanup)
   - Starts the API server on a unique port
   - Provides a configured API client
   - Handles time mocking for time-dependent tests

2. **Test Suite** (`suite_test.go`)
   - Entry point for the test runner
   - Initializes logging and Ginkgo/Gomega

3. **Test Specs** (e.g., `application_flow_test.go`)
   - Individual test scenarios using BDD style
   - Organized with `Describe`, `It`, `BeforeAll`, `AfterAll` blocks

### Test Isolation

Each test suite gets:
- **Unique database**: Randomly named PostgreSQL database for complete isolation
- **Unique server port**: Avoids port conflicts when running tests in parallel
- **Fresh state**: No data pollution between test suites
- **Clean shutdown**: Automatic cleanup after tests complete

## Prerequisites

Before running E2E tests, ensure the following services are running:

```bash
# Start dependencies with Docker Compose
docker compose up -d

# Or start PostgreSQL manually
# PostgreSQL should be running on localhost:5732
# Default credentials: user/password
```

## Running E2E Tests

### Using Just (Recommended)

```bash
# Run all E2E tests
just e2e

# Run full CI pipeline (includes E2E tests)
just ci
```

### Using Go Test

```bash
# Run E2E tests with the e2e build tag
go test -tags=e2e ./tests/e2e/...

# Run with race detector
go test -race -tags=e2e ./tests/e2e/...

# Run with verbose output
go test -v -tags=e2e ./tests/e2e/...

# Run specific test
go test -tags=e2e -run "Application flow" ./tests/e2e/...
```

### Using Ginkgo CLI

```bash
# Install Ginkgo CLI
go install github.com/onsi/ginkgo/v2/ginkgo

# Run tests
ginkgo -tags=e2e ./tests/e2e/

# Run with focus (only focused specs)
ginkgo -tags=e2e --focus "creates application" ./tests/e2e/

# Run in parallel (experimental - requires careful test isolation)
ginkgo -tags=e2e -p ./tests/e2e/
```

## Writing E2E Tests

### Basic Test Structure

```go
package e2e

import (
    "Keyline/internal/handlers"

    . "github.com/onsi/ginkgo/v2"
    . "github.com/onsi/gomega"
)

var _ = Describe("Feature Name", Ordered, func() {
    var h *harness

    BeforeAll(func() {
        // Create test harness once for all tests in this suite
        h = newE2eTestHarness()
    })

    AfterAll(func() {
        // Clean up after all tests
        h.Close()
    })

    It("does something", func() {
        // Your test code here
    })

    It("does something else", func() {
        // Another test
    })
})
```

### Example: Testing Application Creation

```go
var _ = Describe("Application Management", Ordered, func() {
    var h *harness

    BeforeAll(func() {
        h = newE2eTestHarness()
    })

    AfterAll(func() {
        h.Close()
    })

    It("creates a public application", func() {
        // Arrange
        createDto := handlers.CreateApplicationRequestDto{
            Name:           "test-app",
            DisplayName:    "Test Application",
            RedirectUris:   []string{"http://localhost:3000/callback"},
            PostLogoutUris: []string{"http://localhost:3000/logout"},
            Type:           "public",
        }

        // Act
        app, err := h.Client().Application().Create(h.Ctx(), createDto)

        // Assert
        Expect(err).ToNot(HaveOccurred())
        Expect(app.Id).ToNot(BeEmpty())
        Expect(app.Secret).To(BeNil()) // Public apps don't have secrets
    })

    It("creates a confidential application with secret", func() {
        createDto := handlers.CreateApplicationRequestDto{
            Name:           "confidential-app",
            DisplayName:    "Confidential Application",
            RedirectUris:   []string{"http://localhost:3000/callback"},
            PostLogoutUris: []string{"http://localhost:3000/logout"},
            Type:           "confidential",
        }

        app, err := h.Client().Application().Create(h.Ctx(), createDto)

        Expect(err).ToNot(HaveOccurred())
        Expect(app.Id).ToNot(BeEmpty())
        Expect(app.Secret).ToNot(BeNil()) // Confidential apps have secrets
        Expect(*app.Secret).ToNot(BeEmpty())
    })

    It("rejects unauthorized requests", func() {
        createDto := handlers.CreateApplicationRequestDto{
            Name:           "test-app",
            DisplayName:    "Test Application",
            RedirectUris:   []string{"http://localhost:3000/callback"},
            PostLogoutUris: []string{"http://localhost:3000/logout"},
            Type:           "public",
        }

        _, err := h.Client().Application().Create(h.Ctx(), createDto)

        Expect(err).To(HaveOccurred())
        Expect(err.Error()).To(ContainSubstring("401 Unauthorized"))
    })
})
```

### Using the Test Harness

The test harness provides several useful methods and properties:

```go
// Get the API client
client := h.Client()

// Get the context (with authentication and scope)
ctx := h.Ctx()

// Get the virtual server name
vs := h.VirtualServer() // Returns "test-vs"

// Set the current time (for time-dependent tests)
h.SetTime(time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC))

// Clean up resources (called automatically in AfterAll)
h.Close()
```

### Testing with Authentication

Testing with authentication is a WIP.

### Testing Time-Dependent Features

```go
It("handles token expiration", func() {
    // Set initial time
    initialTime := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
    h.SetTime(initialTime)

    // Create a token
    token := createToken(h.Ctx(), h.Client())

    // Advance time by 2 hours
    h.SetTime(initialTime.Add(2 * time.Hour))

    // Token should now be expired
    err := validateToken(h.Ctx(), h.Client(), token)
    Expect(err).To(MatchError("token expired"))
})
```

## Test Harness Details

### Database Management

The harness creates a unique database for each test suite:

```go
// Database name format
dbName := "keyline_test_" + uuid.New().String() // e.g., keyline_test_550e8400e29b41d4a716446655440000

// Automatic cleanup
// Database is dropped when h.Close() is called
```

### Server Configuration

Each test suite gets its own server:

```go
// Port allocation (avoids conflicts)
port := 25001 // Incremented for each new harness

// Server URL
externalUrl := fmt.Sprintf("http://localhost:%d", port)
```

### Initial Test Data

The harness automatically creates:

1. **Virtual Server**: Named "test-vs"
   - Signing algorithm: EdDSA
   - Registration enabled

2. **Admin User**: For authenticated operations
   - Username: `test-admin-user`
   - Email: `test-admin-user@localhost`
   - Password: Pre-hashed (matches config) WIP

3. **Admin UI Application**: System application for admin operations

## Best Practices

### 1. Use Ordered Tests

```go
// ✓ Good: Dependent tests, since each Describe blocks shares one DB and application instance
var _ = Describe("Complete Flow", Ordered, func() {
    It("step 1", func() { /* ... */ })
    It("step 2", func() { /* depends on step 1 */ })
})
```

### 2. Clean Up Resources

Clean up resources that are not created by the keyline server.
The test harness cleans up db and the server itself automatically.

### 3. Use Descriptive Test Names

```go
// ✓ Good: Clear and descriptive
It("creates a public application without a client secret", func() { /* ... */ })
It("rejects requests with invalid redirect URIs", func() { /* ... */ })

// ✗ Avoid: Vague descriptions
It("works", func() { /* ... */ })
It("test 1", func() { /* ... */ })
```

### 4. Test Both Happy and Error Paths

```go
Describe("Application Creation", func() {
    It("succeeds with valid data", func() { /* happy path */ })
    It("fails without authentication", func() { /* error path */ })
    It("fails with invalid type", func() { /* error path */ })
    It("fails with malformed redirect URI", func() { /* error path */ })
})
```

### 5. Use Gomega Matchers Effectively

```go
// ✓ Good: Descriptive matchers
Expect(err).ToNot(HaveOccurred())
Expect(app.Id).ToNot(BeEmpty())
Expect(app.Type).To(Equal("public"))
Expect(apps.Items).To(HaveLen(3))
Expect(err.Error()).To(ContainSubstring("401 Unauthorized"))

// ⚠ Less Descriptive
Expect(err == nil).To(BeTrue())
Expect(app.Id.String() != "").To(BeTrue())
```

### 6. Structure Complex Tests

```go
It("handles complex workflow", func() {
    // Arrange
    app := createTestApplication(h)
    user := createTestUser(h)
    
    // Act
    result := performComplexOperation(h, app, user)
    
    // Assert
    Expect(result.Success).To(BeTrue())
    Expect(result.Data).ToNot(BeNil())
})
```

## CI/CD Integration

E2E tests are included in the CI pipeline and run after the integration tests have passed.

## Related Documentation

- [API Client Documentation](../../client/README.md) - Learn about the Keyline API client
- [Main README](../../README.md) - Project overview and setup
- [Ginkgo Documentation](https://onsi.github.io/ginkgo/) - Testing framework
- [Gomega Documentation](https://onsi.github.io/gomega/) - Matcher library

## File Structure

```
tests/e2e/
├── README.md                  # This file
├── suite_test.go              # Test suite entry point
├── harness.go                 # Test harness implementation
├── application_flow_test.go   # Application-related E2E tests
└── ... (additional test files)
```

## Contributing

When adding new E2E tests:

1. Create a new file with `*_test.go` suffix
2. Add `//go:build e2e` build tag at the top
3. Use the test harness for setup
4. Follow the BDD style with Ginkgo/Gomega
5. Include both happy and error paths
6. Add descriptive test names
7. Clean up resources in `AfterAll` or within tests
8. Update this README if adding new patterns or utilities

---

For questions or issues, please refer to the [main project documentation](../../README.md) or open an issue on GitHub.
