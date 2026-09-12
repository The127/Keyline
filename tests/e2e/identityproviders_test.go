//go:build e2e

package e2e

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
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
				_, err := h.Client().VirtualServer().IdentityProviders().Create(h.Ctx(), explicitIdentityProvider("corp", "Corp SSO"))
				Expect(err).ToNot(HaveOccurred())

				state := loginState(h, beginLogin(h))

				Expect(state["identityProviders"]).To(ConsistOf(map[string]any{
					"name":        "corp",
					"displayName": "Corp SSO",
				}))
			})

			It("refuses a second identity provider with the same name", func() {
				_, err := h.Client().VirtualServer().IdentityProviders().Create(h.Ctx(), explicitIdentityProvider("corp", "Corp SSO again"))

				var apiErr client.ApiError
				Expect(errors.As(err, &apiErr)).To(BeTrue(), "expected an api error, got %v", err)
				Expect(apiErr.Code).To(Equal(http.StatusConflict))
			})

			It("stores the settings of a provider and returns them without the secret", func() {
				_, err := h.Client().VirtualServer().IdentityProviders().Create(h.Ctx(), explicitIdentityProvider("explicit", "Explicit"))
				Expect(err).ToNot(HaveOccurred())

				got, err := h.Client().VirtualServer().IdentityProviders().Get(h.Ctx(), "explicit")
				Expect(err).ToNot(HaveOccurred())
				Expect(got).To(Equal(api.GetIdentityProviderResponseDto{
					Name:                  "explicit",
					DisplayName:           "Explicit",
					AuthorizationEndpoint: "https://idp.example/authorize",
					TokenEndpoint:         "https://idp.example/token",
					UserinfoEndpoint:      "https://idp.example/userinfo",
					Scopes:                []string{"openid", "email"},
					ClientId:              "client-123",
				}))

				raw := getIdentityProviderRaw(h, "explicit")
				Expect(raw).ToNot(HaveKey("clientSecret"))
				rawJson, err := json.Marshal(raw)
				Expect(err).ToNot(HaveOccurred())
				Expect(string(rawJson)).ToNot(ContainSubstring("very-secret"))
			})

			It("keeps the client secret out of the audit log", func() {
				_, err := h.Client().VirtualServer().IdentityProviders().Create(h.Ctx(), explicitIdentityProvider("audited", "Audited"))
				Expect(err).ToNot(HaveOccurred())

				auditLog := getRaw(h, fmt.Sprintf("%s/api/virtual-servers/%s/audit", h.ApiUrl(), h.VirtualServer()))

				Expect(auditLog).To(ContainSubstring("CreateIdentityProvider"))
				Expect(auditLog).ToNot(ContainSubstring("very-secret"))
			})

			It("fills the settings from the github preset", func() {
				_, err := h.Client().VirtualServer().IdentityProviders().Create(h.Ctx(), api.CreateIdentityProviderRequestDto{
					Name:         "github",
					DisplayName:  "GitHub",
					Preset:       "github",
					ClientId:     "gh-client",
					ClientSecret: "gh-secret",
				})
				Expect(err).ToNot(HaveOccurred())

				got, err := h.Client().VirtualServer().IdentityProviders().Get(h.Ctx(), "github")
				Expect(err).ToNot(HaveOccurred())
				Expect(got).To(Equal(api.GetIdentityProviderResponseDto{
					Name:                  "github",
					DisplayName:           "GitHub",
					Preset:                "github",
					AuthorizationEndpoint: "https://github.com/login/oauth/authorize",
					TokenEndpoint:         "https://github.com/login/oauth/access_token",
					UserinfoEndpoint:      "https://api.github.com/user",
					Scopes:                []string{"read:user", "user:email"},
					ClientId:              "gh-client",
				}))
			})

			It("lets explicit settings win over the preset", func() {
				_, err := h.Client().VirtualServer().IdentityProviders().Create(h.Ctx(), api.CreateIdentityProviderRequestDto{
					Name:          "github-enterprise",
					DisplayName:   "GitHub Enterprise",
					Preset:        "github",
					TokenEndpoint: "https://github.example/login/oauth/access_token",
					Scopes:        []string{"read:user"},
					ClientId:      "ghe-client",
					ClientSecret:  "ghe-secret",
				})
				Expect(err).ToNot(HaveOccurred())

				got, err := h.Client().VirtualServer().IdentityProviders().Get(h.Ctx(), "github-enterprise")
				Expect(err).ToNot(HaveOccurred())
				Expect(got.Preset).To(Equal("github"))
				Expect(got.AuthorizationEndpoint).To(Equal("https://github.com/login/oauth/authorize"))
				Expect(got.TokenEndpoint).To(Equal("https://github.example/login/oauth/access_token"))
				Expect(got.Scopes).To(Equal([]string{"read:user"}))
			})

			It("keeps an explicit empty scope list over the preset", func() {
				_, err := h.Client().VirtualServer().IdentityProviders().Create(h.Ctx(), api.CreateIdentityProviderRequestDto{
					Name:         "github-noscopes",
					DisplayName:  "GitHub No Scopes",
					Preset:       "github",
					Scopes:       []string{},
					ClientId:     "gh-client",
					ClientSecret: "gh-secret",
				})
				Expect(err).ToNot(HaveOccurred())

				got, err := h.Client().VirtualServer().IdentityProviders().Get(h.Ctx(), "github-noscopes")
				Expect(err).ToNot(HaveOccurred())
				Expect(got.Scopes).To(Equal([]string{}))
			})

			It("reads an empty scope list back as an empty list", func() {
				provider := explicitIdentityProvider("noscopes", "No Scopes")
				provider.Scopes = nil
				_, err := h.Client().VirtualServer().IdentityProviders().Create(h.Ctx(), provider)
				Expect(err).ToNot(HaveOccurred())

				got, err := h.Client().VirtualServer().IdentityProviders().Get(h.Ctx(), "noscopes")
				Expect(err).ToNot(HaveOccurred())
				Expect(got.Scopes).To(Equal([]string{}))
			})

			Describe("refuses a provider", func() {
				cases := map[string]func() api.CreateIdentityProviderRequestDto{
					"without endpoints or client credentials": func() api.CreateIdentityProviderRequestDto {
						return api.CreateIdentityProviderRequestDto{Name: "bare", DisplayName: "Bare"}
					},
					"with an endpoint that is not an http url": func() api.CreateIdentityProviderRequestDto {
						provider := explicitIdentityProvider("script", "Script")
						provider.AuthorizationEndpoint = "javascript:alert(1)"
						return provider
					},
					"with a name that cannot be a path segment": func() api.CreateIdentityProviderRequestDto {
						return explicitIdentityProvider("a/b", "Slash")
					},
					"with an unknown preset": func() api.CreateIdentityProviderRequestDto {
						provider := explicitIdentityProvider("unknown-preset", "Unknown")
						provider.Preset = "myspace"
						return provider
					},
					"with a preset but no client credentials": func() api.CreateIdentityProviderRequestDto {
						return api.CreateIdentityProviderRequestDto{Name: "no-creds", DisplayName: "No Creds", Preset: "github"}
					},
					"with an empty scope": func() api.CreateIdentityProviderRequestDto {
						provider := explicitIdentityProvider("emptyscope", "Empty Scope")
						provider.Scopes = []string{"openid", ""}
						return provider
					},
				}

				for name, provider := range cases {
					It(name, func() {
						_, err := h.Client().VirtualServer().IdentityProviders().Create(h.Ctx(), provider())

						var apiErr client.ApiError
						Expect(errors.As(err, &apiErr)).To(BeTrue(), "expected an api error, got %v", err)
						Expect(apiErr.Code).To(Equal(http.StatusBadRequest))
					})
				}
			})
		})
	}
}

func explicitIdentityProvider(name string, displayName string) api.CreateIdentityProviderRequestDto {
	return api.CreateIdentityProviderRequestDto{
		Name:                  name,
		DisplayName:           displayName,
		AuthorizationEndpoint: "https://idp.example/authorize",
		TokenEndpoint:         "https://idp.example/token",
		UserinfoEndpoint:      "https://idp.example/userinfo",
		Scopes:                []string{"openid", "email"},
		ClientId:              "client-123",
		ClientSecret:          "very-secret",
	}
}

func getIdentityProviderRaw(h *harness, name string) map[string]any {
	body := getRaw(h, fmt.Sprintf("%s/api/virtual-servers/%s/identity-providers/%s", h.ApiUrl(), h.VirtualServer(), name))

	var raw map[string]any
	Expect(json.Unmarshal([]byte(body), &raw)).To(Succeed())
	return raw
}

func getRaw(h *harness, url string) string {
	token, err := serviceUserTokenSource(h.Ctx(), h.ApiUrl()).Token()
	Expect(err).ToNot(HaveOccurred())

	request, err := http.NewRequest(http.MethodGet, url, nil)
	Expect(err).ToNot(HaveOccurred())
	request.Header.Set("Authorization", "Bearer "+token.AccessToken)

	resp, err := http.DefaultClient.Do(request)
	Expect(err).ToNot(HaveOccurred())
	defer resp.Body.Close() //nolint:errcheck
	Expect(resp.StatusCode).To(Equal(http.StatusOK))

	body, err := io.ReadAll(resp.Body)
	Expect(err).ToNot(HaveOccurred())
	return string(body)
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
