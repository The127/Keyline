//go:build e2e

package e2e

import (
	"errors"

	"github.com/The127/Keyline/api"
	"github.com/The127/Keyline/client"
	"github.com/The127/Keyline/config"
	"github.com/The127/Keyline/internal/commands"

	"github.com/google/uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// findApplicationId locates an application in the given project by name
// via the application listing endpoint.
func findApplicationId(h *harness, projectSlug, appName string) uuid.UUID {
	apps, err := h.Client().Project().Application(projectSlug).List(h.Ctx(), client.ListApplicationParams{
		Page: 0,
		Size: 100,
	})
	Expect(err).ToNot(HaveOccurred())
	for _, a := range apps.Items {
		if a.Name == appName {
			return a.Id
		}
	}
	Fail("application '" + appName + "' not found in project '" + projectSlug + "'")
	return uuid.Nil
}

func init() {
	for _, backend := range testBackends {
		backend := backend
		Describe("PatchApplication system-application guard ["+backend.name+"]", Ordered, func() {
			var h *harness

			BeforeAll(func() {
				if backend.dbMode == config.DatabaseModePostgres && !postgresBackendAvailable() {
					Skip("Postgres not available")
				}
				// Default service user holds VirtualServerAdmin perms,
				// which include `application:update`. That's the
				// attacker profile for this bug -- a caller who has
				// `application:update` but should still not be able to
				// edit the system admin-ui application's redirect URI
				// list.
				h = newE2eTestHarness(backend.dbMode, serviceUserTokenSource)
			})

			AfterAll(func() {
				if h != nil {
					h.Close()
				}
			})

			It("refuses PATCH of redirectUris on the system admin-ui application (the bypass)", func() {
				adminUiId := findApplicationId(h, systemProjectSlug, commands.AdminApplicationName)

				// The legitimate redirect URI for admin-ui is generated
				// from frontend.externalUrl in CreateVirtualServer; we
				// don't need to know it exactly because the guard fires
				// before any list mutation happens.
				attacker := []string{"http://attacker.example/cb"}
				err := h.Client().Project().Application(systemProjectSlug).Patch(
					h.Ctx(),
					adminUiId,
					api.PatchApplicationRequestDto{RedirectUris: attacker},
				)
				Expect(err).To(HaveOccurred())

				var apiErr client.ApiError
				Expect(errors.As(err, &apiErr)).To(BeTrue(), "expected client.ApiError, got %T: %v", err, err)
				Expect(apiErr.Code).To(Equal(400),
					"PATCH redirectUris on a system application must be rejected -- "+
						"otherwise an `application:update` holder can splice an "+
						"attacker-controlled URL into the admin-ui's allowlist and "+
						"intercept any victim's auth code")

				// Database state must be untouched: admin-ui must still
				// be a system application with its original redirect
				// URIs (we can't introspect the original list cheaply
				// without a richer GET; but we can re-PATCH the same
				// attacker URIs and confirm the gate fires identically,
				// proving no side effect of the first call leaked).
				err = h.Client().Project().Application(systemProjectSlug).Patch(
					h.Ctx(),
					adminUiId,
					api.PatchApplicationRequestDto{RedirectUris: attacker},
				)
				Expect(err).To(HaveOccurred())
				Expect(errors.As(err, &apiErr)).To(BeTrue())
				Expect(apiErr.Code).To(Equal(400))
			})

			It("refuses PATCH of postLogoutUris on the system admin-ui application", func() {
				adminUiId := findApplicationId(h, systemProjectSlug, commands.AdminApplicationName)

				attacker := []string{"http://attacker.example/post-logout"}
				err := h.Client().Project().Application(systemProjectSlug).Patch(
					h.Ctx(),
					adminUiId,
					api.PatchApplicationRequestDto{PostLogoutUris: attacker},
				)
				Expect(err).To(HaveOccurred())

				var apiErr client.ApiError
				Expect(errors.As(err, &apiErr)).To(BeTrue(), "expected client.ApiError, got %T: %v", err, err)
				Expect(apiErr.Code).To(Equal(400))
			})

			It("still allows PATCH of redirectUris on a non-system application (regression check)", func() {
				// Provision a fresh non-system project + app. The
				// harness service user has ProjectCreate /
				// ApplicationCreate. System-project app creation is
				// gated to system users, which is unrelated to the
				// guard under test, so we use a regular project.
				projSlug := "patchapp-regression-" + backend.name
				_, err := h.Client().Project().Create(h.Ctx(), api.CreateProjectRequestDto{
					Slug:        projSlug,
					Name:        "Patch Application Regression",
					Description: "Non-system project for the PatchApplication guard regression check.",
				})
				Expect(err).ToNot(HaveOccurred())

				create, err := h.Client().Project().Application(projSlug).Create(h.Ctx(), api.CreateApplicationRequestDto{
					Name:           "regression-non-system-" + backend.name,
					DisplayName:    "Regression non-system app",
					Type:           "public",
					RedirectUris:   []string{"https://legit.example/callback"},
					PostLogoutUris: []string{"https://legit.example/logout"},
				})
				Expect(err).ToNot(HaveOccurred())
				Expect(create.Id).ToNot(BeZero())

				err = h.Client().Project().Application(projSlug).Patch(
					h.Ctx(),
					create.Id,
					api.PatchApplicationRequestDto{
						RedirectUris:   []string{"https://legit.example/callback", "https://other.example/callback"},
						PostLogoutUris: []string{"https://legit.example/logout", "https://other.example/logout"},
					},
				)
				Expect(err).ToNot(HaveOccurred(),
					"the system-application guard must NOT fire on a non-system application")

				app, err := h.Client().Project().Application(projSlug).Get(h.Ctx(), create.Id)
				Expect(err).ToNot(HaveOccurred())
				Expect(app.SystemApplication).To(BeFalse())
				Expect(app.RedirectUris).To(ConsistOf(
					"https://legit.example/callback",
					"https://other.example/callback",
				))
				Expect(app.PostLogoutRedirectUris).To(ConsistOf(
					"https://legit.example/logout",
					"https://other.example/logout",
				))
			})

			It("still allows PATCH of non-redirect fields on the system application", func() {
				// DisplayName, AccessTokenHeaderType, etc. remain
				// editable on system applications. The guard is scoped
				// narrowly to the auth-flow-critical redirect URI
				// fields. This pins that narrow scope.
				adminUiId := findApplicationId(h, systemProjectSlug, commands.AdminApplicationName)

				newDisplay := "Admin Application (renamed by test)"
				err := h.Client().Project().Application(systemProjectSlug).Patch(
					h.Ctx(),
					adminUiId,
					api.PatchApplicationRequestDto{DisplayName: &newDisplay},
				)
				Expect(err).ToNot(HaveOccurred())

				app, err := h.Client().Project().Application(systemProjectSlug).Get(h.Ctx(), adminUiId)
				Expect(err).ToNot(HaveOccurred())
				Expect(app.DisplayName).To(Equal(newDisplay))
			})
		})
	}
}
