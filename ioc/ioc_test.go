package ioc

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

type RocketLauncher interface {
	Launch() (int, error)
}

type SaturnV struct {
}

func (s SaturnV) Launch() (int, error) {
	return 400_000, nil
}

type StorageSystem interface {
	Store(item string)
	IsAvailable(item string) bool
}

type Warehouse struct {
	items map[string]bool
}

func (w *Warehouse) Store(item string) {
	w.items[item] = true
}

func (w *Warehouse) IsAvailable(item string) bool {
	_, ok := w.items[item]
	return ok
}

type Connector interface {
	Open()
	IsOpen() bool
	Close()
}

type TestConnector struct {
	isOpen bool
}

func (t *TestConnector) Open() {
	t.isOpen = true
}

func (t *TestConnector) IsOpen() bool {
	return t.isOpen
}

func (t *TestConnector) Close() {
	t.isOpen = false
}

func TestDependencyIsResolved(t *testing.T) {
	// arrange
	dependencyCollection := NewDependencyCollection()
	RegisterTransient(dependencyCollection, func(dp *DependencyProvider) RocketLauncher {
		return SaturnV{}
	})

	dependencyProvider := dependencyCollection.BuildProvider()
	rocketLauncher := GetDependency[RocketLauncher](dependencyProvider)

	// act
	distance, err := rocketLauncher.Launch()

	// assert
	require.NoError(t, err)
	assert.Greater(t, distance, 100_000)
}

func TestSingletonIsSame(t *testing.T) {
	// arrange
	dependencyCollection := NewDependencyCollection()
	RegisterSingleton(dependencyCollection, func(dp *DependencyProvider) StorageSystem {
		return &Warehouse{
			items: make(map[string]bool),
		}
	})

	dependencyProvider := dependencyCollection.BuildProvider()
	scope := dependencyProvider.NewScope()

	// act
	storageSystem := GetDependency[StorageSystem](scope)
	storageSystem.Store("Diamonds")

	storageSystem2 := GetDependency[StorageSystem](dependencyProvider)
	hasDiamonds := storageSystem2.IsAvailable("Diamonds")

	// assert
	assert.True(t, hasDiamonds)
}

func TestScopeIsSameInSameScope(t *testing.T) {
	// arrange
	dependencyCollection := NewDependencyCollection()
	RegisterScoped(dependencyCollection, func(dp *DependencyProvider) StorageSystem {
		return &Warehouse{
			items: make(map[string]bool),
		}
	})

	dependencyProvider := dependencyCollection.BuildProvider()

	// act
	storageSystem := GetDependency[StorageSystem](dependencyProvider)
	storageSystem.Store("Diamonds")

	storageSystem2 := GetDependency[StorageSystem](dependencyProvider)
	hasDiamonds := storageSystem2.IsAvailable("Diamonds")

	// assert
	assert.True(t, hasDiamonds)
}

func TestScopeIsDifferentInOtherScope(t *testing.T) {
	// arrange
	dependencyCollection := NewDependencyCollection()
	RegisterScoped(dependencyCollection, func(dp *DependencyProvider) StorageSystem {
		return &Warehouse{
			items: make(map[string]bool),
		}
	})

	dependencyProvider := dependencyCollection.BuildProvider()
	scope1 := dependencyProvider.NewScope()
	scope2 := dependencyProvider.NewScope()

	// act
	storageSystem := GetDependency[StorageSystem](scope1)
	storageSystem.Store("Diamonds")

	storageSystem2 := GetDependency[StorageSystem](scope2)
	hasDiamonds := storageSystem2.IsAvailable("Diamonds")

	// assert
	assert.False(t, hasDiamonds)
}

func TestScopeIsNotInherited(t *testing.T) {
	// arrange
	dependencyCollection := NewDependencyCollection()
	RegisterScoped(dependencyCollection, func(dp *DependencyProvider) StorageSystem {
		return &Warehouse{
			items: make(map[string]bool),
		}
	})

	dependencyProvider := dependencyCollection.BuildProvider()
	scope1 := dependencyProvider.NewScope()
	scope2 := scope1.NewScope()

	// act
	storageSystem := GetDependency[StorageSystem](scope1)
	storageSystem.Store("Diamonds")

	storageSystem2 := GetDependency[StorageSystem](scope2)
	hasDiamonds := storageSystem2.IsAvailable("Diamonds")

	// assert
	assert.False(t, hasDiamonds)
}

func TestCloseHandler(t *testing.T) {
	// arrange
	dependencyCollection := NewDependencyCollection()
	RegisterScoped(dependencyCollection, func(dp *DependencyProvider) Connector {
		return &TestConnector{}
	})
	RegisterCloseHandler(dependencyCollection, func(connector Connector) error {
		connector.Close()
		return nil
	})

	dependencyProvider := dependencyCollection.BuildProvider()
	scope1 := dependencyProvider.NewScope()

	// act
	connector := GetDependency[Connector](scope1)
	connector.Open()
	err := scope1.Close()

	// assert
	require.NoError(t, err)
	assert.False(t, connector.IsOpen())
}
