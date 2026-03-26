package architecture

import (
	"testing"

	"github.com/mstrYoda/go-arctest/pkg/arctest"
)

func mustDependOn(t *testing.T, source, target *arctest.Layer) {
	t.Helper()
	if err := source.DependsOnLayer(target); err != nil {
		t.Fatalf("Failed to add dependency %s -> %s: %v", source.Name, target.Name, err)
	}
}

func newArch(t *testing.T) *arctest.Architecture {
	t.Helper()
	arch, err := arctest.New("../../")
	if err != nil {
		t.Fatalf("Failed to create architecture: %v", err)
	}
	err = arch.ParsePackages(
		"internal/handlers",
		"internal/commands",
		"internal/queries",
		"internal/services",
		"internal/repositories",
		"internal/middlewares",
		"internal/authentication",
		"internal/behaviours",
		"internal/server",
		"internal/setup",
		"internal/config",
		"internal/database",
		"internal/events",
		"internal/jobs",
		"internal/logging",
		"internal/metrics",
		"internal/caching",
		"internal/change",
		"internal/jsonTypes",
		"internal/messages",
		"internal/quorum",
		"internal/retry",
		"utils",
	)
	if err != nil {
		t.Fatalf("Failed to parse packages: %v", err)
	}
	return arch
}

// TestHandlersMustNotImportPostgres ensures handlers never bypass the repository
// abstraction to access the database implementation directly.
func TestHandlersMustNotImportPostgres(t *testing.T) {
	arch := newArch(t)

	rule, err := arch.DoesNotDependOn(
		"internal/handlers",
		".*repositories/postgres.*",
	)
	if err != nil {
		t.Fatalf("Failed to create rule: %v", err)
	}

	valid, violations := arch.ValidateDependenciesWithRules([]*arctest.DependencyRule{rule})
	if !valid {
		for _, v := range violations {
			t.Errorf("Violation: %s", v)
		}
	}
}

// TestCommandsMustNotImportHandlers ensures commands (business logic) never
// depend on the HTTP layer.
func TestCommandsMustNotImportHandlers(t *testing.T) {
	arch := newArch(t)

	rule, err := arch.DoesNotDependOn(
		"internal/commands",
		".*internal/handlers.*",
	)
	if err != nil {
		t.Fatalf("Failed to create rule: %v", err)
	}

	valid, violations := arch.ValidateDependenciesWithRules([]*arctest.DependencyRule{rule})
	if !valid {
		for _, v := range violations {
			t.Errorf("Violation: %s", v)
		}
	}
}

// TestQueriesMustNotImportHandlers ensures queries never depend on the HTTP layer.
func TestQueriesMustNotImportHandlers(t *testing.T) {
	arch := newArch(t)

	rule, err := arch.DoesNotDependOn(
		"internal/queries",
		".*internal/handlers.*",
	)
	if err != nil {
		t.Fatalf("Failed to create rule: %v", err)
	}

	valid, violations := arch.ValidateDependenciesWithRules([]*arctest.DependencyRule{rule})
	if !valid {
		for _, v := range violations {
			t.Errorf("Violation: %s", v)
		}
	}
}

// TestPostgresMustNotImportBusinessLogic ensures the database implementation
// layer does not depend on commands, queries, handlers, or services.
func TestPostgresMustNotImportBusinessLogic(t *testing.T) {
	arch := newArch(t)

	targets := []struct {
		name    string
		pattern string
	}{
		{"commands", ".*internal/commands.*"},
		{"queries", ".*internal/queries.*"},
		{"handlers", ".*internal/handlers.*"},
		{"services", ".*internal/services.*"},
	}

	for _, target := range targets {
		t.Run(target.name, func(t *testing.T) {
			rule, err := arch.DoesNotDependOn(
				"internal/repositories/postgres",
				target.pattern,
			)
			if err != nil {
				t.Fatalf("Failed to create rule: %v", err)
			}

			valid, violations := arch.ValidateDependenciesWithRules([]*arctest.DependencyRule{rule})
			if !valid {
				for _, v := range violations {
					t.Errorf("Violation: %s", v)
				}
			}
		})
	}
}

// TestServicesMustNotImportHandlers ensures services never depend on the HTTP layer.
func TestServicesMustNotImportHandlers(t *testing.T) {
	arch := newArch(t)

	rule, err := arch.DoesNotDependOn(
		"internal/services",
		".*internal/handlers.*",
	)
	if err != nil {
		t.Fatalf("Failed to create rule: %v", err)
	}

	valid, violations := arch.ValidateDependenciesWithRules([]*arctest.DependencyRule{rule})
	if !valid {
		for _, v := range violations {
			t.Errorf("Violation: %s", v)
		}
	}
}

// TestUtilsMustNotImportInternal ensures the utils package stays independent
// and does not pull in any internal packages.
func TestUtilsMustNotImportInternal(t *testing.T) {
	arch := newArch(t)

	rule, err := arch.DoesNotDependOn(
		"^utils$",
		".*internal/.*",
	)
	if err != nil {
		t.Fatalf("Failed to create rule: %v", err)
	}

	valid, violations := arch.ValidateDependenciesWithRules([]*arctest.DependencyRule{rule})
	if !valid {
		for _, v := range violations {
			t.Errorf("Violation: %s", v)
		}
	}
}

// TestRepositoryMocksMustNotImportPostgres ensures mock repositories don't
// depend on the concrete postgres implementation.
func TestRepositoryMocksMustNotImportPostgres(t *testing.T) {
	arch := newArch(t)

	rule, err := arch.DoesNotDependOn(
		"internal/repositories/mocks",
		".*repositories/postgres.*",
	)
	if err != nil {
		t.Fatalf("Failed to create rule: %v", err)
	}

	valid, violations := arch.ValidateDependenciesWithRules([]*arctest.DependencyRule{rule})
	if !valid {
		for _, v := range violations {
			t.Errorf("Violation: %s", v)
		}
	}
}

// TestCommandsMustNotImportPostgres ensures commands use repository interfaces,
// not the concrete postgres implementation.
func TestCommandsMustNotImportPostgres(t *testing.T) {
	arch := newArch(t)

	rule, err := arch.DoesNotDependOn(
		"internal/commands",
		".*repositories/postgres.*",
	)
	if err != nil {
		t.Fatalf("Failed to create rule: %v", err)
	}

	valid, violations := arch.ValidateDependenciesWithRules([]*arctest.DependencyRule{rule})
	if !valid {
		for _, v := range violations {
			t.Errorf("Violation: %s", v)
		}
	}
}

// TestQueriesMustNotImportPostgres ensures queries use repository interfaces,
// not the concrete postgres implementation.
func TestQueriesMustNotImportPostgres(t *testing.T) {
	arch := newArch(t)

	rule, err := arch.DoesNotDependOn(
		"internal/queries",
		".*repositories/postgres.*",
	)
	if err != nil {
		t.Fatalf("Failed to create rule: %v", err)
	}

	valid, violations := arch.ValidateDependenciesWithRules([]*arctest.DependencyRule{rule})
	if !valid {
		for _, v := range violations {
			t.Errorf("Violation: %s", v)
		}
	}
}

// TestServicesMustNotImportPostgres ensures services use repository interfaces,
// not the concrete postgres implementation.
func TestServicesMustNotImportPostgres(t *testing.T) {
	arch := newArch(t)

	rule, err := arch.DoesNotDependOn(
		"internal/services",
		".*repositories/postgres.*",
	)
	if err != nil {
		t.Fatalf("Failed to create rule: %v", err)
	}

	valid, violations := arch.ValidateDependenciesWithRules([]*arctest.DependencyRule{rule})
	if !valid {
		for _, v := range violations {
			t.Errorf("Violation: %s", v)
		}
	}
}

// TestLayeredArchitecture validates the overall clean architecture layer
// dependencies using the layered architecture API. Only explicitly allowed
// dependencies are permitted between layers.
func TestLayeredArchitecture(t *testing.T) {
	arch := newArch(t)

	handlersLayer, err := arctest.NewLayer("Handlers", "^internal/handlers$")
	if err != nil {
		t.Fatalf("Failed to create handlers layer: %v", err)
	}

	commandsLayer, err := arctest.NewLayer("Commands", "^internal/commands$")
	if err != nil {
		t.Fatalf("Failed to create commands layer: %v", err)
	}

	queriesLayer, err := arctest.NewLayer("Queries", "^internal/queries$")
	if err != nil {
		t.Fatalf("Failed to create queries layer: %v", err)
	}

	servicesLayer, err := arctest.NewLayer("Services", "^internal/services$")
	if err != nil {
		t.Fatalf("Failed to create services layer: %v", err)
	}

	repositoriesLayer, err := arctest.NewLayer("Repositories", "^internal/repositories$")
	if err != nil {
		t.Fatalf("Failed to create repositories layer: %v", err)
	}

	postgresLayer, err := arctest.NewLayer("Postgres", "^internal/repositories/postgres$")
	if err != nil {
		t.Fatalf("Failed to create postgres layer: %v", err)
	}

	databaseLayer, err := arctest.NewLayer("Database", "^internal/database$")
	if err != nil {
		t.Fatalf("Failed to create database layer: %v", err)
	}

	layeredArch := arch.NewLayeredArchitecture(
		handlersLayer,
		commandsLayer,
		queriesLayer,
		servicesLayer,
		repositoriesLayer,
		postgresLayer,
		databaseLayer,
	)

	// Handlers can depend on commands, queries, services, repositories (interfaces), database
	mustDependOn(t, handlersLayer, commandsLayer)
	mustDependOn(t, handlersLayer, queriesLayer)
	mustDependOn(t, handlersLayer, servicesLayer)
	mustDependOn(t, handlersLayer, repositoriesLayer)
	mustDependOn(t, handlersLayer, databaseLayer)

	// Commands can depend on services, repositories (interfaces), database
	mustDependOn(t, commandsLayer, servicesLayer)
	mustDependOn(t, commandsLayer, repositoriesLayer)
	mustDependOn(t, commandsLayer, databaseLayer)

	// Queries can depend on services, repositories (interfaces), database
	mustDependOn(t, queriesLayer, servicesLayer)
	mustDependOn(t, queriesLayer, repositoriesLayer)
	mustDependOn(t, queriesLayer, databaseLayer)

	// Services can depend on repositories (interfaces), database
	mustDependOn(t, servicesLayer, repositoriesLayer)
	mustDependOn(t, servicesLayer, databaseLayer)

	// Postgres implementation can depend on repository interfaces, database
	mustDependOn(t, postgresLayer, repositoriesLayer)
	mustDependOn(t, postgresLayer, databaseLayer)

	// Database context manages repositories (provides them via DI)
	mustDependOn(t, databaseLayer, repositoriesLayer)

	violations, err := layeredArch.Check()
	if err != nil {
		t.Fatalf("Failed to check layered architecture: %v", err)
	}

	for _, v := range violations {
		t.Errorf("Layer violation: %s", v)
	}
}
