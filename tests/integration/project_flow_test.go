package integration

import (
	"Keyline/internal/commands"
	"Keyline/internal/queries"
	"Keyline/mediator"
	"Keyline/utils"

	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gstruct"
)

var _ = Describe("Project flow", Ordered, func() {
	var h *harness

	projectSlug := "test-project"
	var projectId uuid.UUID

	BeforeAll(func() {
		h = newIntegrationTestHarness()
	})

	AfterAll(func() {
		h.Close()
	})

	It("should create a project successfully", func() {
		req := commands.CreateProject{
			VirtualServerName: h.VirtualServer(),
			Slug:              projectSlug,
			Name:              "Name",
			Description:       "Description",
		}
		response, err := mediator.Send[*commands.CreateProjectResponse](h.Ctx(), h.Mediator(), req)
		Expect(err).ToNot(HaveOccurred())
		projectId = response.Id
	})

	It("should list the project successfully", func() {
		req := queries.ListProjects{
			VirtualServerName: h.VirtualServer(),
			SearchText:        "test",
		}
		response, err := mediator.Send[*queries.ListProjectsResponse](h.Ctx(), h.Mediator(), req)
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
		_, err := mediator.Send[*commands.PatchProjectResponse](h.Ctx(), h.Mediator(), req)
		Expect(err).ToNot(HaveOccurred())
	})

	It("should refelct the updated values", func() {
		req := queries.GetProject{
			VirtualServerName: h.VirtualServer(),
			ProjectSlug:       projectSlug,
		}
		project, err := mediator.Send[*queries.GetProjectResponse](h.Ctx(), h.Mediator(), req)
		Expect(err).ToNot(HaveOccurred())
		Expect(project.Name).To(Equal("Updated Name"))
		Expect(project.Description).To(Equal("Updated Description"))
	})
})
