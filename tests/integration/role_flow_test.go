package integration

import (
	"Keyline/internal/commands"
	"Keyline/internal/queries"
	"Keyline/utils"

	"github.com/The127/mediatr"

	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gstruct"
)

var _ = Describe("Role flow", Ordered, func() {
	var h *harness

	projectSlug := "test-project"
	var roleId uuid.UUID

	BeforeAll(func() {
		h = newIntegrationTestHarness()

		req := commands.CreateProject{
			VirtualServerName: h.VirtualServer(),
			Slug:              projectSlug,
			Name:              "Name",
			Description:       "Description",
		}
		_, err := mediatr.Send[*commands.CreateProjectResponse](h.Ctx(), h.Mediator(), req)
		Expect(err).ToNot(HaveOccurred())
	})

	AfterAll(func() {
		h.Close()
	})

	It("should create a role successfully", func() {
		req := commands.CreateRole{
			VirtualServerName: h.VirtualServer(),
			ProjectSlug:       projectSlug,
			Name:              "test-role",
			Description:       "Description",
		}
		response, err := mediatr.Send[*commands.CreateRoleResponse](h.Ctx(), h.Mediator(), req)
		Expect(err).ToNot(HaveOccurred())
		roleId = response.Id

		Expect(h.dbContext.SaveChanges(h.ctx)).ToNot(HaveOccurred())
	})

	It("should list roles successfully", func() {
		req := queries.ListRoles{
			VirtualServerName: h.VirtualServer(),
			ProjectSlug:       projectSlug,
			SearchText:        "test-role",
		}
		response, err := mediatr.Send[*queries.ListRolesResponse](h.Ctx(), h.Mediator(), req)
		Expect(err).ToNot(HaveOccurred())
		Expect(response.Items).To(ContainElement(gstruct.MatchFields(gstruct.IgnoreExtras, gstruct.Fields{
			"Id":   Equal(roleId),
			"Name": Equal("test-role"),
		})))
	})

	It("should patch role successfully", func() {
		cmd := commands.PatchRole{
			VirtualServerName: h.VirtualServer(),
			ProjectSlug:       projectSlug,
			RoleId:            roleId,
			Name:              utils.Ptr("Updated Name"),
			Description:       utils.Ptr("Updated Description"),
		}
		_, err := mediatr.Send[*commands.PatchRoleResponse](h.Ctx(), h.Mediator(), cmd)
		Expect(err).ToNot(HaveOccurred())

		Expect(h.dbContext.SaveChanges(h.ctx)).ToNot(HaveOccurred())
	})

	It("should reflect updated values", func() {
		req := queries.GetRoleQuery{
			VirtualServerName: h.VirtualServer(),
			ProjectSlug:       projectSlug,
			RoleId:            roleId,
		}
		resp, err := mediatr.Send[*queries.GetRoleQueryResult](h.Ctx(), h.Mediator(), req)
		Expect(err).ToNot(HaveOccurred())
		Expect(resp.Name).To(Equal("Updated Name"))
		Expect(resp.Description).To(Equal("Updated Description"))
	})
})
