package integration

import (
	"Keyline/internal/commands"
	"Keyline/internal/queries"
	"Keyline/internal/repositories"
	"Keyline/mediator"
	"Keyline/utils"

	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gstruct"
)

var _ = Describe("Application flow", Ordered, func() {
	var h *harness

	var applicationId uuid.UUID

	BeforeAll(func() {
		h = newIntegrationTestHarness()
	})

	AfterAll(func() {
		h.Close()
	})

	It("should persist public application successfully", func() {
		req := commands.CreateApplication{
			VirtualServerName:      h.VirtualServer(),
			Name:                   "test-app",
			DisplayName:            "Test App",
			Type:                   repositories.ApplicationTypePublic,
			RedirectUris:           []string{"http://localhost:8080/callback"},
			PostLogoutRedirectUris: []string{"http://localhost:8080/logout"},
		}
		response, err := mediator.Send[*commands.CreateApplicationResponse](h.Ctx(), h.Mediator(), req)
		Expect(err).ToNot(HaveOccurred())
		applicationId = response.Id
	})

	It("should list applications successfully", func() {
		req := queries.ListApplications{
			VirtualServerName: h.VirtualServer(),
			SearchText:        "test-app",
		}
		response, err := mediator.Send[*queries.ListApplicationsResponse](h.Ctx(), h.Mediator(), req)
		Expect(err).ToNot(HaveOccurred())
		Expect(response.Items).To(ContainElement(gstruct.MatchFields(gstruct.IgnoreExtras, gstruct.Fields{
			"Id":   Equal(applicationId),
			"Name": Equal("test-app"),
		})))
	})

	It("should edit application successfully", func() {
		cmd := commands.PatchApplication{
			VirtualServerName: h.VirtualServer(),
			ApplicationId:     applicationId,
			DisplayName:       utils.Ptr("Updated Test App"),
		}
		_, err := mediator.Send[*commands.PatchApplicationResponse](h.Ctx(), h.Mediator(), cmd)
		Expect(err).ToNot(HaveOccurred())
	})

	It("should reflect updated values", func() {
		req := queries.GetApplication{
			VirtualServerName: h.VirtualServer(),
			ApplicationId:     applicationId,
		}
		app, err := mediator.Send[*queries.GetApplicationResult](h.Ctx(), h.Mediator(), req)
		Expect(err).ToNot(HaveOccurred())
		Expect(app.DisplayName).To(Equal("Updated Test App"))
	})
})
