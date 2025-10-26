package e2e

import (
	keylineClient "Keyline/client"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Unautorized", Ordered, func() {
	var h *harness

	BeforeAll(func() {
		h = newE2eTestHarness()
	})

	AfterAll(func() {
		h.Close()
	})

	It("rejects requests", func() {
		_, err := h.Client().User().List(h.Ctx(), keylineClient.ListUserParams{})
		Expect(err).To(HaveOccurred())
		Expect(err).To(MatchError(ContainSubstring("401 Unauthorized")))
	})
})
