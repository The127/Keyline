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
		Describe("Project flow ["+backend.name+"]", Ordered, func() {
			var h *harness

			projectSlug := "test-project"
			var projectId uuid.UUID

			BeforeAll(func() {
				if backend.dbMode == config.DatabaseModePostgres && !postgresBackendAvailable() {
					Skip("Postgres not available")
				}
				h = newIntegrationTestHarness(backend.dbMode)
			})

			AfterAll(func() {
				if h != nil {
					h.Close()
				}
			})

			It("should create a project successfully", func() {
				req := commands.CreateProject{
					VirtualServerName: h.VirtualServer(),
					Slug:              projectSlug,
					Name:              "Name",
					Description:       "Description",
				}
				response, err := mediatr.Send[*commands.CreateProjectResponse](h.Ctx(), h.Mediator(), req)
				Expect(err).ToNot(HaveOccurred())
				projectId = response.Id

				Expect(h.dbContext.SaveChanges(h.ctx)).ToNot(HaveOccurred())
			})

			It("should list the project successfully", func() {
				req := queries.ListProjects{
					VirtualServerName: h.VirtualServer(),
					SearchText:        "test",
				}
				response, err := mediatr.Send[*queries.ListProjectsResponse](h.Ctx(), h.Mediator(), req)
				Expect(err).ToNot(HaveOccurred())
				Expect(response.Items).To(ContainElement(gstruct.MatchFields(gstruct.IgnoreExtras, gstruct.Fields{
					"Id":   Equal(projectId),
					"Slug": Equal(projectSlug),
				})))
			})

			It("should edit the project successfully", func() {
				req := commands.PatchProject{
					VirtualServerName: h.VirtualServer(),
					Slug:              projectSlug,
					Name:              utils.Ptr("Updated Name"),
					Description:       utils.Ptr("Updated Description"),
				}
				_, err := mediatr.Send[*commands.PatchProjectResponse](h.Ctx(), h.Mediator(), req)
				Expect(err).ToNot(HaveOccurred())

				Expect(h.dbContext.SaveChanges(h.ctx)).ToNot(HaveOccurred())
			})

			It("should reflect the updated values", func() {
				req := queries.GetProject{
					VirtualServerName: h.VirtualServer(),
					ProjectSlug:       projectSlug,
				}
				project, err := mediatr.Send[*queries.GetProjectResponse](h.Ctx(), h.Mediator(), req)
				Expect(err).ToNot(HaveOccurred())
				Expect(project.Name).To(Equal("Updated Name"))
				Expect(project.Description).To(Equal("Updated Description"))
			})
		})
	}
}
