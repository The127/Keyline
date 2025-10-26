package e2e

import (
	"Keyline/client"

	ginkgo "github.com/onsi/ginkgo/v2"
	gomega "github.com/onsi/gomega"
)

var _ = ginkgo.Describe("Unautorized", ginkgo.Ordered, func() {
	var h *harness

	ginkgo.BeforeAll(func() {
		h = newE2eTestHarness()
	})

	ginkgo.AfterAll(func() {
		h.Close()
	})

	ginkgo.It("rejects requests", func() {
		_, err := h.Client().User().List(h.Ctx(), client.ListUserParams{})
		gomega.Expect(err).To(gomega.HaveOccurred())
		gomega.Expect(err).To(gomega.MatchError(gomega.ContainSubstring("401 Unauthorized")))
	})
})
