//go:build e2e

package e2e

import (
	"context"

	"github.com/The127/Keyline/api"
	"github.com/The127/Keyline/config"
	"github.com/The127/Keyline/internal/authentication"
	"github.com/The127/Keyline/internal/commands"
	"github.com/The127/Keyline/internal/database"
	"github.com/The127/Keyline/internal/middlewares"
	"github.com/The127/Keyline/utils"

	"github.com/The127/ioc"
	"github.com/The127/mediatr"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func init() {
	for _, backend := range testBackends {
		backend := backend
		Describe("Resource server scope ["+backend.name+"]", Ordered, func() {
			var h *harness

			BeforeAll(func() {
				if backend.dbMode == config.DatabaseModePostgres && !postgresBackendAvailable() {
					Skip("Postgres not available")
				}
				h = newE2eTestHarness(backend.dbMode, serviceUserLogin)

				for _, slug := range []string{"scope-project-a", "scope-project-b"} {
					_, err := h.Client().Project().Create(h.Ctx(), api.CreateProjectRequestDto{
						Slug: slug,
						Name: slug,
					})
					Expect(err).ToNot(HaveOccurred())
				}
			})

			AfterAll(func() {
				if h != nil {
					h.Close()
				}
			})

			It("allows the same scope in two projects", func() {
				Expect(createResourceServerWithScope(h, "scope-project-a", "clusters", "k8s")).To(Succeed())

				err := createResourceServerWithScope(h, "scope-project-b", "clusters", "k8s")

				Expect(err).ToNot(HaveOccurred())
			})

			It("refuses the same scope twice in one project", func() {
				if backend.dbMode == config.DatabaseModeMemory {
					Skip("the memory backend enforces no unique constraints")
				}
				Expect(createResourceServerWithScope(h, "scope-project-a", "dashboards", "grafana")).To(Succeed())

				err := createResourceServerWithScope(h, "scope-project-a", "metrics", "grafana")

				Expect(err).To(MatchError(Or(
					ContainSubstring("resource_server_scopes_project_id_scope_key"),
					ContainSubstring("resource_server_scopes.project_id, resource_server_scopes.scope"),
				)))
			})
		})
	}
}

func createResourceServerWithScope(h *harness, projectSlug string, resourceServerSlug string, scope string) error {
	subscope := h.Scope().NewScope()
	defer utils.PanicOnError(subscope.Close, "closing scope")

	ctx := middlewares.ContextWithScope(context.Background(), subscope)
	ctx = authentication.ContextWithCurrentUser(ctx, authentication.SystemUser())

	m := ioc.GetDependency[mediatr.Mediator](subscope)
	dbContext := ioc.GetDependency[database.Context](subscope)

	resourceServer, err := mediatr.Send[*commands.CreateResourceServerResponse](ctx, m, commands.CreateResourceServer{
		VirtualServerName: h.VirtualServer(),
		ProjectSlug:       projectSlug,
		Slug:              resourceServerSlug,
		Name:              resourceServerSlug,
	})
	if err != nil {
		return err
	}

	err = dbContext.SaveChanges(ctx)
	if err != nil {
		return err
	}

	_, err = mediatr.Send[*commands.CreateResourceServerScopeResponse](ctx, m, commands.CreateResourceServerScope{
		VirtualServerName: h.VirtualServer(),
		ProjectSlug:       projectSlug,
		ResourceServerId:  resourceServer.Id,
		Scope:             scope,
		Name:              scope,
	})
	if err != nil {
		return err
	}

	return dbContext.SaveChanges(ctx)
}
