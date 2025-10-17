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
	var harness *integrationTestHarness

	var applicationId uuid.UUID

	BeforeAll(func() {
		harness = newIntegrationTestHarness()
	})

	AfterAll(func() {
		harness.Close()
	})

	It("should persist public application successfully", func() {
		req := commands.CreateApplication{
			VirtualServerName:      harness.VirtualServer(),
			Name:                   "test-app",
			DisplayName:            "Test App",
			Type:                   repositories.ApplicationTypePublic,
			RedirectUris:           []string{"http://localhost:8080/callback"},
			PostLogoutRedirectUris: []string{"http://localhost:8080/logout"},
		}
		response, err := mediator.Send[*commands.CreateApplicationResponse](harness.Ctx(), harness.Mediator(), req)
		Expect(err).ToNot(HaveOccurred())
		applicationId = response.Id
	})

	It("should list applications successfully", func() {
		req := queries.ListApplications{
			VirtualServerName: harness.VirtualServer(),
			SearchText:        "test-app",
		}
		response, err := mediator.Send[*queries.ListApplicationsResponse](harness.ctx, harness.Mediator(), req)
		Expect(err).ToNot(HaveOccurred())
		Expect(response.Items).To(ContainElement(gstruct.MatchFields(gstruct.IgnoreExtras, gstruct.Fields{
			"Id":   Equal(applicationId),
			"Name": Equal("test-app"),
		})))
	})

	It("should edit application successfully", func() {
		cmd := commands.PatchApplication{
			VirtualServerName: harness.VirtualServer(),
			ApplicationId:     applicationId,
			DisplayName:       utils.Ptr("Updated Test App"),
		}
		_, err := mediator.Send[*commands.PatchApplicationResponse](harness.Ctx(), harness.Mediator(), cmd)
		Expect(err).ToNot(HaveOccurred())
	})

	It("should reflect updated values", func() {
		req := queries.GetApplication{
			VirtualServerName: harness.VirtualServer(),
			ApplicationId:     applicationId,
		}
		app, err := mediator.Send[*queries.GetApplicationResult](harness.Ctx(), harness.Mediator(), req)
		Expect(err).ToNot(HaveOccurred())
		Expect(app.DisplayName).To(Equal("Updated Test App"))
	})
})
