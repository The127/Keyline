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

const applicationNameProjectSlug = "application-name"

func init() {
	for _, backend := range testBackends {
		backend := backend
		Describe("Application name ["+backend.name+"]", Ordered, func() {
			var h *harness

			BeforeAll(func() {
				if backend.dbMode == config.DatabaseModePostgres && !postgresBackendAvailable() {
					Skip("Postgres not available")
				}
				h = newE2eTestHarness(backend.dbMode, serviceUserLogin)

				_, err := h.Client().Project().Create(h.Ctx(), api.CreateProjectRequestDto{
					Slug: applicationNameProjectSlug,
					Name: "Application Name",
				})
				Expect(err).ToNot(HaveOccurred())
			})

			AfterAll(func() {
				if h != nil {
					h.Close()
				}
			})

			It("refuses a name with a colon", func() {
				_, err := h.Client().Project().Application(applicationNameProjectSlug).Create(h.Ctx(), api.CreateApplicationRequestDto{
					Name:         "application-name:api",
					DisplayName:  "API",
					Type:         "public",
					RedirectUris: []string{"http://localhost:9100/callback"},
				})

				var apiError client.ApiError
				Expect(errors.As(err, &apiError)).To(BeTrue())
				Expect(apiError.Code).To(Equal(http.StatusBadRequest))
			})
		})
	}
}
