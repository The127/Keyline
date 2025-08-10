package ioc

import (
	"Keyline/logging"
	"Keyline/utils"
	"reflect"
)

type ProviderFunc[TDependency any] func() TDependency

func (f ProviderFunc[TDependency]) untyped() ProviderFunc[any] {
	return func() any {
		return f()
	}
}

// DependencyCollection is a collection of registered types.
// A type can be registered with a interface and queried by that interface.
type DependencyCollection struct {
	providers map[reflect.Type]ProviderFunc[any]
}

func NewDependencyCollection() *DependencyCollection {
	return &DependencyCollection{
		providers: make(map[reflect.Type]ProviderFunc[any]),
	}
}

func (dc *DependencyCollection) clone() *DependencyCollection {
	other := NewDependencyCollection()
	for t, provider := range dc.providers {
		other.providers[t] = provider
	}
	return other
}

func Register[TDependency any](dc *DependencyCollection, provider ProviderFunc[TDependency]) {
	dc.providers[utils.TypeOf[TDependency]()] = provider.untyped()
}

func (dc *DependencyCollection) BuildProvider() *DependencyProvider {
	return &DependencyProvider{
		dc: dc.clone(),
	}
}

type DependencyProvider struct {
	dc *DependencyCollection
}

func GetDependency[TDependency any](dp *DependencyProvider) TDependency {
	dependencyType := utils.TypeOf[TDependency]()
	providerFunction, ok := dp.dc.providers[dependencyType]
	if !ok {
		logging.Logger.Fatalf("could not provde dependency for %s", dependencyType.Name())
	}

	return providerFunction().(TDependency)
}

type Greeter interface {
	Greet(name string)
}

type ConsoleGreeter struct {
}

func (c *ConsoleGreeter) Greet(name string) {
	print("Hello " + name)
}
