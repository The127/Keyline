//go:build e2e

package e2e

import (
	"context"

	"github.com/The127/Keyline/api"
	"github.com/The127/Keyline/client"
	"github.com/The127/Keyline/config"
	"github.com/The127/Keyline/internal/commands"
	"github.com/The127/Keyline/utils"

	"golang.org/x/oauth2"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// serviceUserTokenSource returns a token source signed with the harness's default
// service user private key, targeting the given VS's admin application.
func serviceUserTokenSource(ctx context.Context, url string) oauth2.TokenSource {
	return &client.ServiceUserTokenSource{
		KeylineURL:    url,
		VirtualServer: "test-vs",
		PrivKeyPEM:    serviceUserPrivateKey,
		Kid:           serviceUserKid,
		Username:      serviceUserUsername,
		Application:   commands.AdminApplicationName,
	}
}

func init() {
	for _, backend := range testBackends {
		backend := backend
		Describe("User client Create ["+backend.name+"]", Ordered, func() {
			var h *harness

			BeforeAll(func() {
				if backend.dbMode == config.DatabaseModePostgres && !postgresBackendAvailable() {
					Skip("Postgres not available")
				}
				h = newE2eTestHarness(backend.dbMode, serviceUserTokenSource)
			})

			AfterAll(func() {
				if h != nil {
					h.Close()
				}
			})

			It("creates a user and returns its id", func() {
				resp, err := h.Client().User().Create(h.Ctx(), api.CreateUserRequestDto{
					Username:      "e2e-create-user",
					DisplayName:   "E2E Create User",
					Email:         "e2e-create-user@localhost",
					EmailVerified: utils.Ptr(true),
				})

				Expect(err).ToNot(HaveOccurred())
				Expect(resp.Id).ToNot(BeZero())

				got, err := h.Client().User().Get(h.Ctx(), resp.Id)
				Expect(err).ToNot(HaveOccurred())
				Expect(got.Username).To(Equal("e2e-create-user"))
				Expect(got.DisplayName).To(Equal("E2E Create User"))
				Expect(got.PrimaryEmail).To(Equal("e2e-create-user@localhost"))
				Expect(got.EmailVerified).To(BeTrue())
				Expect(got.IsServiceUser).To(BeFalse())
			})

			It("creates a user with a temporary password", func() {
				resp, err := h.Client().User().Create(h.Ctx(), api.CreateUserRequestDto{
					Username:    "e2e-temp-password-user",
					DisplayName: "Temp Password User",
					Email:       "e2e-temp-password-user@localhost",
					Password: &api.CreateUserRequestDtoPasword{
						Plain:     "initial-password",
						Temporary: true,
					},
				})

				Expect(err).ToNot(HaveOccurred())
				Expect(resp.Id).ToNot(BeZero())
			})
		})
	}
}
