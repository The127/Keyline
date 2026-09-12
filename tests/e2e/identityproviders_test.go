//go:build e2e

package e2e

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"

	"github.com/The127/Keyline/api"
	"github.com/The127/Keyline/client"
	"github.com/The127/Keyline/config"
	"github.com/The127/Keyline/internal/commands"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func init() {
	for _, backend := range testBackends {
		backend := backend
		Describe("Identity providers ["+backend.name+"]", Ordered, func() {
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

			It("offers no identity providers on a fresh login", func() {
				state := loginState(h, beginLogin(h))

				Expect(state).To(HaveKeyWithValue("identityProviders", BeEmpty()))
			})

			It("offers a registered identity provider on the login", func() {
				_, err := h.Client().VirtualServer().IdentityProviders().Create(h.Ctx(), api.CreateIdentityProviderRequestDto{
					Name:        "corp",
					DisplayName: "Corp SSO",
				})
				Expect(err).ToNot(HaveOccurred())

				state := loginState(h, beginLogin(h))

				Expect(state["identityProviders"]).To(ConsistOf(map[string]any{
					"name":        "corp",
					"displayName": "Corp SSO",
				}))
			})

			It("refuses a second identity provider with the same name", func() {
				_, err := h.Client().VirtualServer().IdentityProviders().Create(h.Ctx(), api.CreateIdentityProviderRequestDto{
					Name:        "corp",
					DisplayName: "Corp SSO again",
				})

				var apiErr client.ApiError
				Expect(errors.As(err, &apiErr)).To(BeTrue(), "expected an api error, got %v", err)
				Expect(apiErr.Code).To(Equal(http.StatusConflict))
			})
		})
	}
}

func beginLogin(h *harness) string {
	httpClient := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	query := url.Values{}
	query.Set("response_type", "code")
	query.Set("client_id", commands.AdminApplicationName)
	query.Set("redirect_uri", fmt.Sprintf("%s/mgmt/%s/auth", config.C.Frontend.ExternalUrl, h.VirtualServer()))
	query.Set("scope", "openid")
	query.Set("state", "idp-state")
	query.Set("code_challenge", authCodePkceChallenge(authCodePkceVerifier))
	query.Set("code_challenge_method", "S256")

	resp, err := httpClient.Get(fmt.Sprintf("%s/oidc/%s/authorize?%s", h.ApiUrl(), h.VirtualServer(), query.Encode()))
	Expect(err).ToNot(HaveOccurred())
	defer resp.Body.Close() //nolint:errcheck
	Expect(resp.StatusCode).To(Equal(http.StatusFound))

	location, err := url.Parse(resp.Header.Get("Location"))
	Expect(err).ToNot(HaveOccurred())
	loginToken := location.Query().Get("token")
	Expect(loginToken).ToNot(BeEmpty())
	return loginToken
}

func loginState(h *harness, loginToken string) map[string]any {
	resp, err := http.Get(fmt.Sprintf("%s/logins/%s", h.ApiUrl(), loginToken))
	Expect(err).ToNot(HaveOccurred())
	defer resp.Body.Close() //nolint:errcheck
	Expect(resp.StatusCode).To(Equal(http.StatusOK))

	var state map[string]any
	Expect(json.NewDecoder(resp.Body).Decode(&state)).To(Succeed())
	return state
}
