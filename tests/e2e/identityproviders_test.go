//go:build e2e

package e2e

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/The127/Keyline/api"
	"github.com/The127/Keyline/client"
	"github.com/The127/Keyline/config"
	"github.com/The127/Keyline/internal/authentication"
	"github.com/The127/Keyline/internal/commands"
	"github.com/The127/Keyline/internal/database"
	"github.com/The127/Keyline/internal/middlewares"
	"github.com/The127/Keyline/internal/queries"
	"github.com/The127/Keyline/utils"
	"github.com/The127/ioc"
	"github.com/The127/mediatr"

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

			It("fills the settings from the google preset", func() {
				_, err := h.Client().VirtualServer().IdentityProviders().Create(h.Ctx(), api.CreateIdentityProviderRequestDto{
					Name:         "google",
					DisplayName:  "Google",
					Preset:       "google",
					ClientId:     "g-client",
					ClientSecret: "g-secret",
				})
				Expect(err).ToNot(HaveOccurred())

				got, err := h.Client().VirtualServer().IdentityProviders().Get(h.Ctx(), "google")
				Expect(err).ToNot(HaveOccurred())
				Expect(got).To(Equal(api.GetIdentityProviderResponseDto{
					Name:                  "google",
					DisplayName:           "Google",
					Preset:                "google",
					Issuer:                "https://accounts.google.com",
					AuthorizationEndpoint: "https://accounts.google.com/o/oauth2/v2/auth",
					TokenEndpoint:         "https://oauth2.googleapis.com/token",
					UserinfoEndpoint:      "https://openidconnect.googleapis.com/v1/userinfo",
					Scopes:                []string{"openid", "email", "profile"},
					ClientId:              "g-client",
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

			It("starts a login at a provider", func() {
				status, body := startIdentityProviderLogin(h, beginLogin(h), "corp")

				Expect(status).To(Equal(http.StatusOK))
				authorizationUrl, err := url.Parse(body["authorizationUrl"].(string))
				Expect(err).ToNot(HaveOccurred())
				Expect(authorizationUrl.Scheme + "://" + authorizationUrl.Host + authorizationUrl.Path).To(Equal("https://idp.example/authorize"))

				query := authorizationUrl.Query()
				Expect(query.Get("client_id")).To(Equal("client-123"))
				Expect(query.Get("redirect_uri")).To(Equal(fmt.Sprintf("%s/oidc/%s/identity-providers/corp/callback", config.C.Server.ExternalUrl, h.VirtualServer())))
				Expect(query.Get("response_type")).To(Equal("code"))
				Expect(query.Get("scope")).To(Equal("openid email"))
				Expect(query.Get("state")).ToNot(BeEmpty())
				Expect(query.Get("code_challenge_method")).To(Equal("S256"))
				Expect(query.Get("code_challenge")).To(HaveLen(43))
				Expect(query.Get("nonce")).ToNot(BeEmpty())
			})

			It("uses a fresh state for every start", func() {
				loginToken := beginLogin(h)
				_, first := startIdentityProviderLogin(h, loginToken, "corp")
				_, second := startIdentityProviderLogin(h, loginToken, "corp")

				Expect(stateOf(first)).ToNot(Equal(stateOf(second)))
			})

			It("refuses to start at an unknown provider", func() {
				status, _ := startIdentityProviderLogin(h, beginLogin(h), "nobody")

				Expect(status).To(Equal(http.StatusNotFound))
			})

			It("refuses to start with an unknown login token", func() {
				status, _ := startIdentityProviderLogin(h, "no-such-login", "corp")

				Expect(status).To(Equal(http.StatusUnauthorized))
			})

			It("refuses to start once a user is identified", func() {
				createUserinfoUser(h.Scope())
				loginToken := beginLogin(h)
				Expect(h.Client().Oidc().VerifyPassword(h.Ctx(), loginToken, userinfoUserUsername, userinfoUserPassword)).To(Succeed())

				status, _ := startIdentityProviderLogin(h, loginToken, "corp")

				Expect(status).To(Equal(http.StatusUnauthorized))
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
					"with an issuer that is not an http url": func() api.CreateIdentityProviderRequestDto {
						provider := explicitIdentityProvider("bad-issuer", "Bad Issuer")
						provider.Issuer = "accounts.google.com"
						return provider
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

		Describe("Identity providers from initial configuration ["+backend.name+"]", Ordered, func() {
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

			It("creates the providers declared for the initial virtual server", func() {
				scope := h.Scope().NewScope()
				defer utils.PanicOnError(scope.Close, "closing scope")
				ctx := middlewares.ContextWithScope(context.Background(), scope)
				ctx = authentication.ContextWithCurrentUser(ctx, authentication.SystemUser())
				m := ioc.GetDependency[mediatr.Mediator](scope)
				dbContext := ioc.GetDependency[database.Context](scope)

				const vsName = "idp-config-vs"
				_, err := mediatr.Send[*commands.CreateVirtualServerResponse](ctx, m, commands.CreateVirtualServer{
					Name:                    vsName,
					DisplayName:             "IdP Config VS",
					PrimarySigningAlgorithm: config.SigningAlgorithmEdDSA,
					IdentityProviders: []commands.CreateVirtualServerIdentityProvider{
						{
							Name:         "github",
							DisplayName:  "GitHub",
							Preset:       "github",
							ClientId:     "gh-client",
							ClientSecret: "gh-secret",
						},
						{
							Name:                  "corp",
							DisplayName:           "Corp SSO",
							Issuer:                "https://idp.example",
							AuthorizationEndpoint: "https://idp.example/authorize",
							TokenEndpoint:         "https://idp.example/token",
							UserinfoEndpoint:      "https://idp.example/userinfo",
							Scopes:                []string{"openid", "email"},
							ClientId:              "client-123",
							ClientSecret:          "very-secret",
						},
					},
				})
				Expect(err).ToNot(HaveOccurred())
				Expect(dbContext.SaveChanges(ctx)).To(Succeed())

				github, err := mediatr.Send[*queries.GetIdentityProviderResult](ctx, m, queries.GetIdentityProvider{
					VirtualServerName: vsName,
					Name:              "github",
				})
				Expect(err).ToNot(HaveOccurred())
				Expect(github.Preset).To(Equal("github"))
				Expect(github.Settings.TokenEndpoint).To(Equal("https://github.com/login/oauth/access_token"))
				Expect(github.Settings.ClientSecret).To(Equal("gh-secret"))

				corp, err := mediatr.Send[*queries.GetIdentityProviderResult](ctx, m, queries.GetIdentityProvider{
					VirtualServerName: vsName,
					Name:              "corp",
				})
				Expect(err).ToNot(HaveOccurred())
				Expect(corp.Settings.Issuer).To(Equal("https://idp.example"))
				Expect(corp.Settings.Scopes).To(Equal([]string{"openid", "email"}))
			})

			Describe("refuses an initial provider", func() {
				cases := map[string]commands.CreateVirtualServerIdentityProvider{
					"that is incomplete":                        {Name: "bare", DisplayName: "Bare", ClientId: "x", ClientSecret: "y"},
					"with a name that cannot be a path segment": {Name: "a/b", DisplayName: "Slash", Preset: "github", ClientId: "x", ClientSecret: "y"},
					"without a display name":                    {Name: "nodisplay", Preset: "github", ClientId: "x", ClientSecret: "y"},
					"with an endpoint that is not an http url":  {Name: "script", DisplayName: "Script", Preset: "github", AuthorizationEndpoint: "javascript:alert(1)", ClientId: "x", ClientSecret: "y"},
					"with an empty scope":                       {Name: "emptyscope", DisplayName: "Empty", Preset: "github", Scopes: []string{"openid", ""}, ClientId: "x", ClientSecret: "y"},
				}

				for name, provider := range cases {
					It(name, func() {
						scope := h.Scope().NewScope()
						defer utils.PanicOnError(scope.Close, "closing scope")
						ctx := middlewares.ContextWithScope(context.Background(), scope)
						ctx = authentication.ContextWithCurrentUser(ctx, authentication.SystemUser())
						m := ioc.GetDependency[mediatr.Mediator](scope)

						_, err := mediatr.Send[*commands.CreateVirtualServerResponse](ctx, m, commands.CreateVirtualServer{
							Name:                    "idp-config-bad-" + provider.Name,
							DisplayName:             "Bad",
							PrimarySigningAlgorithm: config.SigningAlgorithmEdDSA,
							IdentityProviders:       []commands.CreateVirtualServerIdentityProvider{provider},
						})
						Expect(err).To(MatchError(utils.ErrHttpBadRequest))
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

func startIdentityProviderLogin(h *harness, loginToken string, name string) (int, map[string]any) {
	resp, err := http.Post(fmt.Sprintf("%s/logins/%s/identity-providers/%s/start", h.ApiUrl(), loginToken, name), "application/json", nil)
	Expect(err).ToNot(HaveOccurred())
	defer resp.Body.Close() //nolint:errcheck

	var body map[string]any
	_ = json.NewDecoder(resp.Body).Decode(&body)
	return resp.StatusCode, body
}

func stateOf(body map[string]any) string {
	authorizationUrl, err := url.Parse(body["authorizationUrl"].(string))
	Expect(err).ToNot(HaveOccurred())
	return authorizationUrl.Query().Get("state")
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
