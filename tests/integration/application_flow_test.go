package integration

import (
	"Keyline/internal/commands"
	"Keyline/internal/queries"
	"Keyline/internal/repositories"
	"Keyline/utils"

	"github.com/The127/mediatr"

	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gstruct"
)

var _ = Describe("Application flow", Ordered, func() {
	var h *harness

	projectSlug := "test-project"
	var applicationId uuid.UUID

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

	It("should persist public application successfully", func() {
		req := commands.CreateApplication{
			VirtualServerName:      h.VirtualServer(),
			ProjectSlug:            projectSlug,
			Name:                   "test-app",
			DisplayName:            "Test App",
			Type:                   repositories.ApplicationTypePublic,
			RedirectUris:           []string{"http://localhost:8080/callback"},
			PostLogoutRedirectUris: []string{"http://localhost:8080/logout"},
		}
		response, err := mediatr.Send[*commands.CreateApplicationResponse](h.Ctx(), h.Mediator(), req)
		Expect(err).ToNot(HaveOccurred())
		applicationId = response.Id
	})

	It("should list applications successfully", func() {
		req := queries.ListApplications{
			VirtualServerName: h.VirtualServer(),
			ProjectSlug:       projectSlug,
			SearchText:        "test-app",
		}
		response, err := mediatr.Send[*queries.ListApplicationsResponse](h.Ctx(), h.Mediator(), req)
		Expect(err).ToNot(HaveOccurred())
		Expect(response.Items).To(ContainElement(gstruct.MatchFields(gstruct.IgnoreExtras, gstruct.Fields{
			"Id":   Equal(applicationId),
			"Name": Equal("test-app"),
		})))
	})

	It("should edit application successfully", func() {
		cmd := commands.PatchApplication{
			VirtualServerName: h.VirtualServer(),
			ProjectSlug:       projectSlug,
			ApplicationId:     applicationId,
			DisplayName:       utils.Ptr("Updated Test App"),
		}
		_, err := mediatr.Send[*commands.PatchApplicationResponse](h.Ctx(), h.Mediator(), cmd)
		Expect(err).ToNot(HaveOccurred())
	})

	It("should reflect updated values", func() {
		req := queries.GetApplication{
			VirtualServerName: h.VirtualServer(),
			ProjectSlug:       projectSlug,
			ApplicationId:     applicationId,
		}
		app, err := mediatr.Send[*queries.GetApplicationResult](h.Ctx(), h.Mediator(), req)
		Expect(err).ToNot(HaveOccurred())
		Expect(app.DisplayName).To(Equal("Updated Test App"))
	})

	It("should delete application successfully", func() {
		cmd := commands.DeleteApplication{
			VirtualServerName: h.VirtualServer(),
			ProjectSlug:       projectSlug,
			ApplicationId:     applicationId,
		}
		_, err := mediatr.Send[*commands.DeleteApplicationResponse](h.Ctx(), h.Mediator(), cmd)
		Expect(err).ToNot(HaveOccurred())
	})

	It("should not list deleted application", func() {
		req := queries.ListApplications{
			VirtualServerName: h.VirtualServer(),
			ProjectSlug:       projectSlug,
			SearchText:        "test-app",
		}
		response, err := mediatr.Send[*queries.ListApplicationsResponse](h.Ctx(), h.Mediator(), req)
		Expect(err).ToNot(HaveOccurred())
		Expect(response.Items).To(BeEmpty())
	})
})
