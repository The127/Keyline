# Mediator Pattern for Go

This package provides a lightweight mediator pattern implementation with support for request/response handlers, behaviors (middleware), and event handling.

## Features
- Type-safe registration using Go generics
- Request/response pattern with typed handlers
- Behavior pipeline (similar to middleware)
- Event/notification pattern with multiple handlers
- Decoupled communication between components

## Usage

### Basic Request/Response

```go
package main

import (
    "context"
    "Keyline/mediator"
)

// Define a request type
type GetUserRequest struct {
    UserID string
}

// Define a response type
type GetUserResponse struct {
    Username string
    Email    string
}

func main() {
    m := mediator.NewMediator()

    // Register a handler for the request
    mediator.RegisterHandler(m, func(ctx context.Context, request GetUserRequest) (GetUserResponse, error) {
        // Handle the request
        return GetUserResponse{
            Username: "john_doe",
            Email:    "john@example.com",
        }, nil
    })

    // Send a request and receive a response
    response, err := mediator.Send[GetUserResponse](context.Background(), m, GetUserRequest{
        UserID: "123",
    })
    if err != nil {
        // Handle error
    }
    
    // Use the response
    println(response.Username)
}
```

### Behaviors (Middleware)

Behaviors allow you to add cross-cutting concerns like logging, validation, or authorization to your request pipeline:

```go
// Define a behavior for logging
mediator.RegisterBehaviour(m, func(ctx context.Context, request GetUserRequest, next mediator.Next) (any, error) {
    println("Before handling request")
    
    // Call the next behavior or handler
    response, err := next()
    
    println("After handling request")
    return response, err
})

// Register the handler
mediator.RegisterHandler(m, func(ctx context.Context, request GetUserRequest) (GetUserResponse, error) {
    return GetUserResponse{Username: "john_doe"}, nil
})

// The behavior will be executed before and after the handler
response, err := mediator.Send[GetUserResponse](context.Background(), m, GetUserRequest{UserID: "123"})
```

Behaviors are executed in the order they are registered, forming a pipeline:

```
Request → Behaviour 1 → Behaviour 2 → Handler → Behaviour 2 → Behaviour 1 → Response
```

### Event Handling

Events allow you to notify multiple handlers about something that has happened:

```go
// Define an event type
type UserCreatedEvent struct {
    UserID   string
    Username string
}

// Register multiple event handlers
mediator.RegisterEventHandler(m, func(ctx context.Context, evt UserCreatedEvent) error {
    println("Sending welcome email to", evt.Username)
    return nil
})

mediator.RegisterEventHandler(m, func(ctx context.Context, evt UserCreatedEvent) error {
    println("Creating user profile for", evt.Username)
    return nil
})

// Send an event to all registered handlers
err := mediator.SendEvent(context.Background(), m, UserCreatedEvent{
    UserID:   "123",
    Username: "john_doe",
})
```

## Real-World Example

Here's a complete example showing how handlers, behaviors, and events work together:

```go
package main

import (
    "context"
    "errors"
    "Keyline/mediator"
)

type CreateUserCommand struct {
    Username string
    Email    string
}

type CreateUserResult struct {
    UserID string
}

type UserCreatedEvent struct {
    UserID   string
    Username string
}

func main() {
    m := mediator.NewMediator()

    // Register a validation behavior
    mediator.RegisterBehaviour(m, func(ctx context.Context, request CreateUserCommand, next mediator.Next) (any, error) {
        if request.Username == "" {
            return nil, errors.New("username is required")
        }
        if request.Email == "" {
            return nil, errors.New("email is required")
        }
        return next()
    })

    // Register the command handler
    mediator.RegisterHandler(m, func(ctx context.Context, cmd CreateUserCommand) (CreateUserResult, error) {
        // Create the user in database
        userID := "user-123"
        
        // Publish an event
        mediator.SendEvent(ctx, m, UserCreatedEvent{
            UserID:   userID,
            Username: cmd.Username,
        })
        
        return CreateUserResult{UserID: userID}, nil
    })

    // Register event handlers
    mediator.RegisterEventHandler(m, func(ctx context.Context, evt UserCreatedEvent) error {
        println("Sending welcome email to", evt.Username)
        return nil
    })

    // Execute the command
    result, err := mediator.Send[CreateUserResult](context.Background(), m, CreateUserCommand{
        Username: "john_doe",
        Email:    "john@example.com",
    })
    if err != nil {
        panic(err)
    }
    
    println("User created with ID:", result.UserID)
}
```

## Key Concepts

### Handlers

Handlers process a request and return a response. Each request type can have exactly one handler registered.

```go
type HandlerFunc[TRequest any, TResponse any] func(ctx context.Context, request TRequest) (TResponse, error)
```

### Behaviors

Behaviors are like middleware that wrap around handlers. They can:
- Execute logic before and after the handler
- Short-circuit the pipeline by returning early
- Modify the request or response
- Add cross-cutting concerns (logging, validation, authorization, etc.)

```go
type BehaviourFunc[TRequest any] func(ctx context.Context, request TRequest, next Next) (any, error)
```

Behaviors are applied to all requests that match their type constraint. If a behavior is registered for a base type or interface, it will be applied to all requests that implement that type.

### Events

Events represent notifications that something has happened. Unlike requests:
- Events can have multiple handlers
- Event handlers don't return values (except errors)
- Events are processed sequentially
- If an event handler returns an error, processing stops

```go
type EventHandlerFunc[TEvent any] func(ctx context.Context, evt TEvent) error
```

## Integration with IoC Container

The mediator is typically registered as a singleton in the IoC container:

```go
import (
    "Keyline/ioc"
    "Keyline/mediator"
)

func setupMediator(dc *ioc.DependencyCollection) {
    m := mediator.NewMediator()

    // Register handlers, behaviors, and event handlers
    mediator.RegisterHandler(m, HandleCreateUser)
    mediator.RegisterHandler(m, HandleGetUser)
    mediator.RegisterBehaviour(m, ValidationBehaviour)
    mediator.RegisterEventHandler(m, OnUserCreated)

    // Register mediator as singleton
    ioc.RegisterSingleton(dc, func(dp *ioc.DependencyProvider) mediator.Mediator {
        return m
    })
}
```

## Benefits

1. **Decoupling** - Request senders don't need to know about handlers
2. **Single Responsibility** - Each handler has one job
3. **Open/Closed Principle** - Add new handlers without modifying existing code
4. **Cross-Cutting Concerns** - Behaviors handle common logic like validation and logging
5. **Event-Driven Architecture** - Decouple components using events
6. **Type Safety** - Generic implementation ensures compile-time type checking

## Thread Safety

The mediator itself is thread-safe for registration operations (handlers, behaviors, events should be registered during application startup).

Handler execution is thread-safe as long as the handlers themselves are thread-safe. The same mediator instance can be used concurrently from multiple goroutines.
