//go:build e2e

package e2e

import (
	"errors"
	"net/http"

	"github.com/The127/Keyline/client"
	"github.com/The127/Keyline/client/api"
	"github.com/The127/Keyline/config"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func init() {
	for _, backend := range testBackends {
		backend := backend
		Describe("Project slug ["+backend.name+"]", Ordered, func() {
			var h *harness

			BeforeAll(func() {
				if backend.dbMode == config.DatabaseModePostgres && !postgresBackendAvailable() {
					Skip("Postgres not available")
				}
				h = newE2eTestHarness(backend.dbMode, serviceUserLogin)
			})

			AfterAll(func() {
				if h != nil {
					h.Close()
				}
			})

			It("refuses a slug with a colon", func() {
				_, err := h.Client().Project().Create(h.Ctx(), api.CreateProjectRequestDto{
					Slug: "team:clusters",
					Name: "Team Clusters",
				})

				var apiError client.ApiError
				Expect(errors.As(err, &apiError)).To(BeTrue())
				Expect(apiError.Code).To(Equal(http.StatusBadRequest))
			})
		})
	}
}
