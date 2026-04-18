package e2e

import (
	"github.com/The127/Keyline/client"
	"github.com/The127/Keyline/config"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func init() {
	for _, backend := range testBackends {
		backend := backend
		Describe("Unauthorized ["+backend.name+"]", Ordered, func() {
			var h *harness

			BeforeAll(func() {
				if backend.dbMode == config.DatabaseModePostgres && !postgresBackendAvailable() {
					Skip("Postgres not available")
				}
				h = newE2eTestHarness(backend.dbMode, nil)
			})

			AfterAll(func() {
				if h != nil {
					h.Close()
				}
			})

			It("rejects requests", func() {
				_, err := h.Client().User().List(h.Ctx(), client.ListUserParams{})
				Expect(err).To(HaveOccurred())
				Expect(err).To(MatchError(ContainSubstring("401 Unauthorized")))
			})
		})
	}
}
