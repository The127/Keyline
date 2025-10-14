# IoC Container for Go

This package provides a lightweight inversion-of-control (IoC) container
with support for `transient`, `scoped`, and `singleton` lifetimes.

## Features
- Type-safe registration using Go generics
- Transient, scoped, and singleton lifetimes
- Scope hierarchy (`NewScope`)
- Resource cleanup with close handlers
- Thread-safe singleton initialization (using `sync.Mutex`)

## Usage

```go
type RocketLauncher interface {
    Launch() (int, error)
}

type SaturnV struct{}

func (s SaturnV) Launch() (int, error) {
    return 400_000, nil
}

func main() {
    dc := NewDependencyCollection()

    // register transient
    RegisterTransient(dc, func(dp *DependencyProvider) RocketLauncher {
        return SaturnV{}
    })

    // register singleton
    RegisterSingleton(dc, func(dp *DependencyProvider) *Warehouse {
        return &Warehouse{items: map[string]bool{}}
    })

    provider := dc.BuildProvider()
    scope := provider.NewScope()

    // resolve dependency
    rocket := GetDependency[RocketLauncher](scope)
    distance, _ := rocket.Launch()
    fmt.Println("Launched to", distance, "km")
}
``` 

## Lifetimes

### Transient
A new instance is created every time the dependency is resolved.

### Scoped
A single instance is created per scope.
Each call to provider.NewScope() creates a new scope.

### Singleton
A single instance is created at most once per root provider.
Shared across all scopes.

## Close Handlers

Singleton and scoped resources can be cleaned up when a scope or root provider is closed:

```go
RegisterScoped(dc, func(dp *DependencyProvider) *Connection {
    return &Connection{}
})

RegisterCloseHandler(dc, func(conn *Connection) error {
    conn.Close()
    return nil
})

scope := provider.NewScope()
conn := GetDependency[*Connection](scope)
_ = scope.Close() // conn.Close() is called
```

For transient services the caller is responsible to handle closing them.
It is recommended to not use transient services that need to be closed or disposed.

## Thread Safety

Singletons are safe for concurrent access.
Thread safety of a singleton is subject to that singletons implementation.

Scoped and Transient dependencies should not be shared across goroutines
unless you add your own synchronization.