package integration

import (
	"github.com/The127/Keyline/internal/commands"
	"github.com/The127/Keyline/config"
	"github.com/The127/Keyline/internal/queries"
	"github.com/The127/Keyline/utils"

	"github.com/The127/mediatr"

	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gstruct"
)

func init() {
	for _, backend := range testBackends {
		backend := backend
		Describe("ResourceServer flow ["+backend.name+"]", Ordered, func() {
			var h *harness

			projectSlug := "test-project"
			var resourceServerId uuid.UUID

			BeforeAll(func() {
				if backend.dbMode == config.DatabaseModePostgres && !postgresBackendAvailable() {
					Skip("Postgres not available")
				}
				h = newIntegrationTestHarness(backend.dbMode)

				req := commands.CreateProject{
					VirtualServerName: h.VirtualServer(),
					Slug:              projectSlug,
					Name:              "Name",
					Description:       "Description",
				}
				_, err := mediatr.Send[*commands.CreateProjectResponse](h.Ctx(), h.Mediator(), req)
				Expect(err).ToNot(HaveOccurred())

				Expect(h.dbContext.SaveChanges(h.ctx)).ToNot(HaveOccurred())
			})

			AfterAll(func() {
				if h != nil {
					h.Close()
				}
			})

			It("should create a resource server successfully", func() {
				req := commands.CreateResourceServer{
					VirtualServerName: h.VirtualServer(),
					ProjectSlug:       projectSlug,
					Slug:              "test-resource-server",
					Name:              "Test Resource Server",
					Description:       "Description",
				}
				response, err := mediatr.Send[*commands.CreateResourceServerResponse](h.Ctx(), h.Mediator(), req)
				Expect(err).ToNot(HaveOccurred())
				resourceServerId = response.Id

				Expect(h.dbContext.SaveChanges(h.ctx)).ToNot(HaveOccurred())
			})

			It("should list resource servers successfully", func() {
				req := queries.ListResourceServers{
					VirtualServerName: h.VirtualServer(),
					ProjectSlug:       projectSlug,
					SearchText:        "test-resource-server",
				}
				resp, err := mediatr.Send[*queries.ListResourceServersResponse](h.Ctx(), h.Mediator(), req)
				Expect(err).ToNot(HaveOccurred())
				Expect(resp.Items).To(ContainElement(gstruct.MatchFields(gstruct.IgnoreExtras, gstruct.Fields{
					"Id":   Equal(resourceServerId),
					"Slug": Equal("test-resource-server"),
					"Name": Equal("Test Resource Server"),
				})))
			})

			It("should edit resource server successfully", func() {
				req := commands.PatchResourceServer{
					VirtualServerName: h.VirtualServer(),
					ProjectSlug:       projectSlug,
					ResourceServerId:  resourceServerId,
					Name:              utils.Ptr("Updated Test Resource Server"),
				}
				_, err := mediatr.Send[*commands.PatchResourceServerResponse](h.Ctx(), h.Mediator(), req)
				Expect(err).ToNot(HaveOccurred())

				Expect(h.dbContext.SaveChanges(h.ctx)).ToNot(HaveOccurred())
			})

			It("should reflect updated values", func() {
				req := queries.GetResourceServer{
					VirtualServerName: h.VirtualServer(),
					ProjectSlug:       projectSlug,
					ResourceServerId:  resourceServerId,
				}
				resp, err := mediatr.Send[*queries.GetResourceServerResponse](h.Ctx(), h.Mediator(), req)
				Expect(err).ToNot(HaveOccurred())
				Expect(resp.Name).To(Equal("Updated Test Resource Server"))
			})
		})
	}
}
