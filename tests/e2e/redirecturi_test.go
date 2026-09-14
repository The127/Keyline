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

const redirectUriProjectSlug = "redirect-uri"

func init() {
	for _, backend := range testBackends {
		backend := backend
		Describe("Redirect URI ["+backend.name+"]", Ordered, ContinueOnFailure, func() {
			var h *harness

			BeforeAll(func() {
				if backend.dbMode == config.DatabaseModePostgres && !postgresBackendAvailable() {
					Skip("Postgres not available")
				}
				h = newE2eTestHarness(backend.dbMode, serviceUserLogin)

				_, err := h.Client().Project().Create(h.Ctx(), api.CreateProjectRequestDto{
					Slug: redirectUriProjectSlug,
					Name: "Redirect URI",
				})
				Expect(err).ToNot(HaveOccurred())
			})

			AfterAll(func() {
				if h != nil {
					h.Close()
				}
			})

			It("refuses a javascript redirect URI", func() {
				_, err := h.Client().Project().Application(redirectUriProjectSlug).Create(h.Ctx(), api.CreateApplicationRequestDto{
					Name:         "javascript-app",
					DisplayName:  "Javascript App",
					Type:         "public",
					RedirectUris: []string{"javascript:alert(document.domain)"},
				})

				var apiError client.ApiError
				Expect(errors.As(err, &apiError)).To(BeTrue())
				Expect(apiError.Code).To(Equal(http.StatusBadRequest))
			})

			It("accepts the private-use scheme of a native app", func() {
				_, err := h.Client().Project().Application(redirectUriProjectSlug).Create(h.Ctx(), api.CreateApplicationRequestDto{
					Name:         "native-app",
					DisplayName:  "Native App",
					Type:         "public",
					RedirectUris: []string{"com.example.app:/callback"},
				})

				Expect(err).ToNot(HaveOccurred())
			})
		})
	}
}
