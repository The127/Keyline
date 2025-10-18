package e2e

import (
	"Keyline/internal/handlers"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Application flow", Ordered, func() {
	var h *harness

	BeforeAll(func() {
		h = newE2eTestHarness()
	})

	AfterAll(func() {
		h.Close()
	})

	It("rejects unauthorized requests", func() {
		_, err := h.Client().Application().Create(h.Ctx(), handlers.CreateApplicationRequestDto{
			Name:           "test-app",
			DisplayName:    "Test App",
			RedirectUris:   []string{"http://localhost:8080/callback"},
			PostLogoutUris: []string{"http://localhost:8080/logout"},
			Type:           "public",
		})
		Expect(err).To(HaveOccurred())
		Expect(err).To(MatchError(ContainSubstring("401 Unauthorized")))
	})
})
