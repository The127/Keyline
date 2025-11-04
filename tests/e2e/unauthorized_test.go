package e2e

import (
	"Keyline/client"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Unauthorized", Ordered, func() {
	var h *harness

	BeforeAll(func() {
		h = newE2eTestHarness(nil)
	})

	AfterAll(func() {
		h.Close()
	})

	It("rejects requests", func() {
		_, err := h.Client().User().List(h.Ctx(), client.ListUserParams{})
		Expect(err).To(HaveOccurred())
		Expect(err).To(MatchError(ContainSubstring("401 Unauthorized")))
	})
})
