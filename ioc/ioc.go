package ioc

import (
	"Keyline/logging"
	"Keyline/utils"
	"errors"
	"maps"
	"reflect"
	"sync"
)

type ProviderFunc[TDependency any] func(dp *DependencyProvider) TDependency

func (f ProviderFunc[TDependency]) untyped() ProviderFunc[any] {
	return func(dp *DependencyProvider) any {
		return f(dp)
	}
}

type CloseHandler[TDependency any] func(TDependency) error

func (c CloseHandler[TDependency]) untyped() CloseHandler[any] {
	return func(dependency any) error {
		return c(dependency.(TDependency))
	}
}

// DependencyCollection is a collection of registered types.
// A type can be registered with a interface and queried by that interface.
type DependencyCollection struct {
	transientProviders map[reflect.Type]ProviderFunc[any]
	scopedProviders    map[reflect.Type]ProviderFunc[any]
	singletonProviders map[reflect.Type]ProviderFunc[any]
	closeHandlers      map[reflect.Type]CloseHandler[any]
}

func NewDependencyCollection() *DependencyCollection {
	return &DependencyCollection{
		transientProviders: make(map[reflect.Type]ProviderFunc[any]),
		scopedProviders:    make(map[reflect.Type]ProviderFunc[any]),
		singletonProviders: make(map[reflect.Type]ProviderFunc[any]),
		closeHandlers:      make(map[reflect.Type]CloseHandler[any]),
	}
}

func (dc *DependencyCollection) clone() *DependencyCollection {
	other := NewDependencyCollection()

	other.transientProviders = maps.Clone(dc.transientProviders)
	other.scopedProviders = maps.Clone(dc.scopedProviders)
	other.singletonProviders = maps.Clone(dc.singletonProviders)
	other.closeHandlers = maps.Clone(dc.closeHandlers)

	return other
}

func RegisterTransient[TDependency any](dc *DependencyCollection, provider ProviderFunc[TDependency]) {
	dc.transientProviders[utils.TypeOf[TDependency]()] = provider.untyped()
}

func RegisterScoped[TDependency any](dc *DependencyCollection, provider ProviderFunc[TDependency]) {
	dc.scopedProviders[utils.TypeOf[TDependency]()] = provider.untyped()
}

func RegisterSingleton[TDependency any](dc *DependencyCollection, provider ProviderFunc[TDependency]) {
	dc.singletonProviders[utils.TypeOf[TDependency]()] = provider.untyped()
}

func RegisterCloseHandler[TDependency any](dc *DependencyCollection, handler CloseHandler[TDependency]) {
	dc.closeHandlers[utils.TypeOf[TDependency]()] = handler.untyped()
}

func (dc *DependencyCollection) BuildProvider() *DependencyProvider {
	return &DependencyProvider{
		dc:                 dc.clone(),
		scopedInstances:    map[reflect.Type]any{},
		singletonInstances: map[reflect.Type]any{},
	}
}

type DependencyProvider struct {
	parentProvider     *DependencyProvider
	dc                 *DependencyCollection
	scopedInstances    map[reflect.Type]any
	singletonInstances map[reflect.Type]any
}

func (dp *DependencyProvider) NewScope() *DependencyProvider {
	provider := dp.dc.BuildProvider()
	provider.parentProvider = dp
	return provider
}

func (dp *DependencyProvider) GetRoot() *DependencyProvider {
	if dp.parentProvider == nil {
		return dp
	}

	return dp.parentProvider.GetRoot()
}

func (dp *DependencyProvider) Close() error {
	var errs []error

	for dependencyType, closeHandler := range dp.dc.closeHandlers {
		dependency, ok := dp.scopedInstances[dependencyType]
		if ok {
			err := closeHandler(dependency)
			if err != nil {
				errs = append(errs, err)
			}
		}

		dependency, ok = dp.singletonInstances[dependencyType]
		if ok {
			err := closeHandler(dependency)
			if err != nil {
				errs = append(errs, err)
			}
		}
	}

	return errors.Join(errs...)
}

func GetDependency[TDependency any](dp *DependencyProvider) TDependency {
	dependencyType := utils.TypeOf[TDependency]()

	dependency, ok := dp.getSingletonDependency(dependencyType)
	if ok {
		return dependency.(TDependency)
	}

	dependency, ok = dp.getScopedDependency(dependencyType)
	if ok {
		return dependency.(TDependency)
	}

	dependency, ok = dp.getTransientDependency(dependencyType)
	if ok {
		return dependency.(TDependency)
	}

	logging.Logger.Fatalf("could not provide dependency for %s", dependencyType.Name())
	panic(dependencyType)
}

var singletonMutexMapLock sync.Mutex
var singletonMutexes = make(map[reflect.Type]*sync.Mutex)

func getSingletonMutex(dependencyType reflect.Type) *sync.Mutex {
	singletonMutexMapLock.Lock()
	defer singletonMutexMapLock.Unlock()

	if mutex, ok := singletonMutexes[dependencyType]; ok {
		return mutex
	}

	mutex := &sync.Mutex{}
	singletonMutexes[dependencyType] = mutex
	return mutex
}

func (dp *DependencyProvider) getSingletonDependency(dependencyType reflect.Type) (any, bool) {
	rootProvider := dp.GetRoot()
	dependency, ok := rootProvider.singletonInstances[dependencyType]
	if ok {
		return dependency, true
	}

	providerFunc, ok := rootProvider.dc.singletonProviders[dependencyType]
	if !ok {
		return nil, false
	}

	mutex := getSingletonMutex(dependencyType)
	mutex.Lock()
	defer mutex.Unlock()

	// check singleton instances again after acquiring the mutex
	dependency, ok = rootProvider.singletonInstances[dependencyType]
	if ok {
		return dependency, true
	}

	// no one else created the dependency in the meantime, we create and save it for reuse
	dependency = providerFunc(dp)
	rootProvider.singletonInstances[dependencyType] = dependency

	return dependency, true
}

func (dp *DependencyProvider) getScopedDependency(dependencyType reflect.Type) (any, bool) {
	dependency, ok := dp.scopedInstances[dependencyType]
	if ok {
		return dependency, true
	}

	providerFunc, ok := dp.dc.scopedProviders[dependencyType]
	if !ok {
		return nil, false
	}

	dependency = providerFunc(dp)
	dp.scopedInstances[dependencyType] = dependency

	return dependency, true
}

func (dp *DependencyProvider) getTransientDependency(dependencyType reflect.Type) (any, bool) {
	providerFunc, ok := dp.dc.transientProviders[dependencyType]
	if !ok {
		return nil, false
	}
	return providerFunc(dp), true
}
