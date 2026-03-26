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

var _ = Describe("Group flow", Ordered, func() {
	var h *harness
	var groupId uuid.UUID

	BeforeAll(func() {
		h = newIntegrationTestHarness()
	})

	AfterAll(func() {
		h.Close()
	})

	It("should create a group successfully", func() {
		req := commands.CreateGroup{
			VirtualServerName: h.VirtualServer(),
			Name:              "test-group",
			Description:       "A test group",
		}
		response, err := mediatr.Send[*commands.CreateGroupResponse](h.Ctx(), h.Mediator(), req)
		Expect(err).ToNot(HaveOccurred())
		groupId = response.Id

		Expect(h.dbContext.SaveChanges(h.ctx)).ToNot(HaveOccurred())
	})

	It("should list groups and find created group", func() {
		req := queries.ListGroups{
			VirtualServerName: h.VirtualServer(),
			SearchText:        "test-group",
		}
		response, err := mediatr.Send[*queries.ListGroupsResponse](h.Ctx(), h.Mediator(), req)
		Expect(err).ToNot(HaveOccurred())
		Expect(response.Items).To(ContainElement(gstruct.MatchFields(gstruct.IgnoreExtras, gstruct.Fields{
			"Id":   Equal(groupId),
			"Name": Equal("test-group"),
		})))
	})

	It("should get group by id", func() {
		req := queries.GetGroupQuery{
			VirtualServerName: h.VirtualServer(),
			GroupId:           groupId,
		}
		resp, err := mediatr.Send[*queries.GetGroupQueryResult](h.Ctx(), h.Mediator(), req)
		Expect(err).ToNot(HaveOccurred())
		Expect(resp.Id).To(Equal(groupId))
		Expect(resp.Name).To(Equal("test-group"))
		Expect(resp.Description).To(Equal("A test group"))
	})

	It("should patch group successfully", func() {
		cmd := commands.PatchGroup{
			VirtualServerName: h.VirtualServer(),
			GroupId:           groupId,
			Name:              utils.Ptr("updated-group"),
			Description:       utils.Ptr("Updated description"),
		}
		_, err := mediatr.Send[*commands.PatchGroupResponse](h.Ctx(), h.Mediator(), cmd)
		Expect(err).ToNot(HaveOccurred())

		Expect(h.dbContext.SaveChanges(h.ctx)).ToNot(HaveOccurred())
	})

	It("should reflect updated values", func() {
		req := queries.GetGroupQuery{
			VirtualServerName: h.VirtualServer(),
			GroupId:           groupId,
		}
		resp, err := mediatr.Send[*queries.GetGroupQueryResult](h.Ctx(), h.Mediator(), req)
		Expect(err).ToNot(HaveOccurred())
		Expect(resp.Name).To(Equal("updated-group"))
		Expect(resp.Description).To(Equal("Updated description"))
	})

	It("should delete group successfully", func() {
		cmd := commands.DeleteGroup{
			VirtualServerName: h.VirtualServer(),
			GroupId:           groupId,
		}
		_, err := mediatr.Send[*commands.DeleteGroupResponse](h.Ctx(), h.Mediator(), cmd)
		Expect(err).ToNot(HaveOccurred())

		Expect(h.dbContext.SaveChanges(h.ctx)).ToNot(HaveOccurred())
	})

	It("should not find deleted group in list", func() {
		req := queries.ListGroups{
			VirtualServerName: h.VirtualServer(),
			SearchText:        "updated-group",
		}
		response, err := mediatr.Send[*queries.ListGroupsResponse](h.Ctx(), h.Mediator(), req)
		Expect(err).ToNot(HaveOccurred())
		for _, item := range response.Items {
			Expect(item.Id).ToNot(Equal(groupId))
		}
	})

	It("should handle delete of non-existent group gracefully", func() {
		cmd := commands.DeleteGroup{
			VirtualServerName: h.VirtualServer(),
			GroupId:           uuid.New(),
		}
		_, err := mediatr.Send[*commands.DeleteGroupResponse](h.Ctx(), h.Mediator(), cmd)
		Expect(err).ToNot(HaveOccurred())
	})
})
