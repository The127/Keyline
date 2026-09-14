//go:build e2e

package e2e

import (
	"context"

	"github.com/The127/Keyline/client/api"
	"github.com/The127/Keyline/config"
	"github.com/The127/Keyline/internal/authentication"
	"github.com/The127/Keyline/internal/commands"
	"github.com/The127/Keyline/internal/database"
	"github.com/The127/Keyline/internal/middlewares"
	"github.com/The127/Keyline/internal/queries"
	"github.com/The127/Keyline/utils"

	"github.com/The127/ioc"
	"github.com/The127/mediatr"
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

type resourceServerScopeFixture struct {
	ProjectSlug        string
	ResourceServerSlug string
	Scope              string
	Name               string
	Description        string
}

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
				_, _, err := createResourceServerWithScope(h, resourceServerScopeFixture{
					ProjectSlug:        "scope-project-a",
					ResourceServerSlug: "clusters",
					Scope:              "k8s",
					Name:               "Kubernetes",
				})
				Expect(err).ToNot(HaveOccurred())

				_, _, err = createResourceServerWithScope(h, resourceServerScopeFixture{
					ProjectSlug:        "scope-project-b",
					ResourceServerSlug: "clusters",
					Scope:              "k8s",
					Name:               "Kubernetes",
				})

				Expect(err).ToNot(HaveOccurred())
			})

			It("refuses the same scope twice in one project", func() {
				if backend.dbMode == config.DatabaseModeMemory {
					Skip("the memory backend enforces no unique constraints")
				}
				_, _, err := createResourceServerWithScope(h, resourceServerScopeFixture{
					ProjectSlug:        "scope-project-a",
					ResourceServerSlug: "dashboards",
					Scope:              "grafana",
					Name:               "Grafana",
				})
				Expect(err).ToNot(HaveOccurred())

				_, _, err = createResourceServerWithScope(h, resourceServerScopeFixture{
					ProjectSlug:        "scope-project-a",
					ResourceServerSlug: "metrics",
					Scope:              "grafana",
					Name:               "Grafana",
				})

				Expect(err).To(MatchError(Or(
					ContainSubstring("resource_server_scopes_project_id_scope_key"),
					ContainSubstring("resource_server_scopes.project_id, resource_server_scopes.scope"),
				)))
			})

			It("reads a scope back", func() {
				resourceServerId, scopeId, err := createResourceServerWithScope(h, resourceServerScopeFixture{
					ProjectSlug:        "scope-project-a",
					ResourceServerSlug: "logs",
					Scope:              "loki",
					Name:               "Loki",
					Description:        "Read the logs",
				})
				Expect(err).ToNot(HaveOccurred())

				response, err := getResourceServerScope(h, "scope-project-a", resourceServerId, scopeId)

				Expect(err).ToNot(HaveOccurred())
				Expect(response.Scope).To(Equal("loki"))
				Expect(response.Name).To(Equal("Loki"))
				Expect(response.Description).To(Equal("Read the logs"))
			})

			It("lists the scopes of a resource server", func() {
				resourceServerId, scopeId, err := createResourceServerWithScope(h, resourceServerScopeFixture{
					ProjectSlug:        "scope-project-b",
					ResourceServerSlug: "traces",
					Scope:              "tempo",
					Name:               "Tempo",
					Description:        "Read the traces",
				})
				Expect(err).ToNot(HaveOccurred())

				response, err := listResourceServerScopes(h, "scope-project-b", resourceServerId)

				Expect(err).ToNot(HaveOccurred())
				Expect(response.TotalCount).To(Equal(1))
				Expect(response.Items).To(Equal([]queries.ListResourceServerScopesResponseItem{
					{
						Id:    scopeId,
						Name:  "Tempo",
						Scope: "tempo",
					},
				}))
			})
		})
	}
}

func systemUserScope(h *harness) (context.Context, *ioc.DependencyProvider) {
	subscope := h.Scope().NewScope()

	ctx := middlewares.ContextWithScope(context.Background(), subscope)
	ctx = authentication.ContextWithCurrentUser(ctx, authentication.SystemUser())

	return ctx, subscope
}

func createResourceServerWithScope(h *harness, fixture resourceServerScopeFixture) (uuid.UUID, uuid.UUID, error) {
	ctx, subscope := systemUserScope(h)
	defer utils.PanicOnError(subscope.Close, "closing scope")

	m := ioc.GetDependency[mediatr.Mediator](subscope)
	dbContext := ioc.GetDependency[database.Context](subscope)

	resourceServer, err := mediatr.Send[*commands.CreateResourceServerResponse](ctx, m, commands.CreateResourceServer{
		VirtualServerName: h.VirtualServer(),
		ProjectSlug:       fixture.ProjectSlug,
		Slug:              fixture.ResourceServerSlug,
		Name:              fixture.ResourceServerSlug,
	})
	if err != nil {
		return uuid.Nil, uuid.Nil, err
	}

	err = dbContext.SaveChanges(ctx)
	if err != nil {
		return uuid.Nil, uuid.Nil, err
	}

	resourceServerScope, err := mediatr.Send[*commands.CreateResourceServerScopeResponse](ctx, m, commands.CreateResourceServerScope{
		VirtualServerName: h.VirtualServer(),
		ProjectSlug:       fixture.ProjectSlug,
		ResourceServerId:  resourceServer.Id,
		Scope:             fixture.Scope,
		Name:              fixture.Name,
		Description:       fixture.Description,
	})
	if err != nil {
		return uuid.Nil, uuid.Nil, err
	}

	err = dbContext.SaveChanges(ctx)
	if err != nil {
		return uuid.Nil, uuid.Nil, err
	}

	return resourceServer.Id, resourceServerScope.Id, nil
}

func getResourceServerScope(h *harness, projectSlug string, resourceServerId uuid.UUID, scopeId uuid.UUID) (*queries.GetResourceServerScopeResponse, error) {
	ctx, subscope := systemUserScope(h)
	defer utils.PanicOnError(subscope.Close, "closing scope")

	m := ioc.GetDependency[mediatr.Mediator](subscope)

	return mediatr.Send[*queries.GetResourceServerScopeResponse](ctx, m, queries.GetResourceServerScope{
		VirtualServerName: h.VirtualServer(),
		ProjectSlug:       projectSlug,
		ResourceServerId:  resourceServerId,
		ScopeId:           scopeId,
	})
}

func listResourceServerScopes(h *harness, projectSlug string, resourceServerId uuid.UUID) (*queries.ListResourceServerScopesResponse, error) {
	ctx, subscope := systemUserScope(h)
	defer utils.PanicOnError(subscope.Close, "closing scope")

	m := ioc.GetDependency[mediatr.Mediator](subscope)

	return mediatr.Send[*queries.ListResourceServerScopesResponse](ctx, m, queries.ListRessouceServerScopes{
		VirtualServerName: h.VirtualServer(),
		ProjectSlug:       projectSlug,
		ResourceServerId:  resourceServerId,
	})
}
