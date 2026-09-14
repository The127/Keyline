//go:build e2e

package e2e

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/The127/Keyline/client/api"
	"github.com/The127/Keyline/config"
	"github.com/The127/Keyline/internal/authentication"
	"github.com/The127/Keyline/internal/commands"
	"github.com/The127/Keyline/internal/database"
	"github.com/The127/Keyline/internal/middlewares"
	"github.com/The127/Keyline/internal/queries"
	"github.com/The127/Keyline/internal/repositories"
	"github.com/The127/Keyline/utils"

	"github.com/The127/ioc"
	"github.com/The127/mediatr"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

const (
	exchangeProjectSlug  = "token-exchange"
	exchangeRedirectUri  = "http://localhost:9300/callback"
	exchangerAppName     = "mungcp"
	relayAppName         = "relay"
	publicAppName        = "mungbean-cli"
	targetAppName        = "cluster-a"
	untrustingAppName    = "cluster-b"
	exchangeUserUsername = "exchange-user"
	exchangeUserPassword = "exchange-password-123!"
)

func init() {
	for _, backend := range testBackends {
		backend := backend
		Describe("Trusted exchangers ["+backend.name+"]", Ordered, func() {
			var h *harness

			createApplication := func(name string, trustedExchangers []string) api.CreateApplicationResponseDto {
				created, err := h.Client().Project().Application(exchangeProjectSlug).Create(h.Ctx(), api.CreateApplicationRequestDto{
					Name:              name,
					DisplayName:       name,
					Type:              "public",
					RedirectUris:      []string{exchangeRedirectUri},
					TrustedExchangers: trustedExchangers,
				})
				Expect(err).ToNot(HaveOccurred())
				return created
			}

			getApplication := func(id api.CreateApplicationResponseDto) api.GetApplicationResponseDto {
				application, err := h.Client().Project().Application(exchangeProjectSlug).Get(h.Ctx(), id.Id)
				Expect(err).ToNot(HaveOccurred())
				return application
			}

			BeforeAll(func() {
				if backend.dbMode == config.DatabaseModePostgres && !postgresBackendAvailable() {
					Skip("Postgres not available")
				}
				h = newE2eTestHarness(backend.dbMode, serviceUserLogin)

				_, err := h.Client().Project().Create(h.Ctx(), api.CreateProjectRequestDto{
					Slug: exchangeProjectSlug,
					Name: "Token Exchange",
				})
				Expect(err).ToNot(HaveOccurred())
			})

			AfterAll(func() {
				if h != nil {
					h.Close()
				}
			})

			It("trusts nobody by default", func() {
				created := createApplication("lonely-app", nil)

				Expect(getApplication(created).TrustedExchangers).To(BeEmpty())
			})

			It("stores the trusted exchangers on creation and changes them through patch", func() {
				created := createApplication(targetAppName, []string{exchangerAppName})
				Expect(getApplication(created).TrustedExchangers).To(ConsistOf(exchangerAppName))

				Expect(h.Client().Project().Application(exchangeProjectSlug).Patch(h.Ctx(), created.Id, api.PatchApplicationRequestDto{
					TrustedExchangers: []string{},
				})).To(Succeed())
				Expect(getApplication(created).TrustedExchangers).To(BeEmpty())

				Expect(h.Client().Project().Application(exchangeProjectSlug).Patch(h.Ctx(), created.Id, api.PatchApplicationRequestDto{
					DisplayName: utils.Ptr("Cluster A"),
				})).To(Succeed())
				Expect(getApplication(created).TrustedExchangers).To(BeEmpty())
			})
		})

		Describe("Token exchange ["+backend.name+"]", Ordered, func() {
			var h *harness
			var exchangerSecret string
			var relaySecret string

			BeforeAll(func() {
				if backend.dbMode == config.DatabaseModePostgres && !postgresBackendAvailable() {
					Skip("Postgres not available")
				}
				h = newE2eTestHarness(backend.dbMode, serviceUserLogin)

				_, err := h.Client().Project().Create(h.Ctx(), api.CreateProjectRequestDto{
					Slug: exchangeProjectSlug,
					Name: "Token Exchange",
				})
				Expect(err).ToNot(HaveOccurred())

				exchanger, err := h.Client().Project().Application(exchangeProjectSlug).Create(h.Ctx(), api.CreateApplicationRequestDto{
					Name:              exchangerAppName,
					DisplayName:       exchangerAppName,
					Type:              "confidential",
					RedirectUris:      []string{exchangeRedirectUri},
					DeviceFlowEnabled: true,
				})
				Expect(err).ToNot(HaveOccurred())
				Expect(exchanger.Secret).ToNot(BeNil())
				exchangerSecret = *exchanger.Secret

				_, err = h.Client().Project().Application(exchangeProjectSlug).Create(h.Ctx(), api.CreateApplicationRequestDto{
					Name:              publicAppName,
					DisplayName:       publicAppName,
					Type:              "public",
					RedirectUris:      []string{exchangeRedirectUri},
					DeviceFlowEnabled: true,
				})
				Expect(err).ToNot(HaveOccurred())

				relay, err := h.Client().Project().Application(exchangeProjectSlug).Create(h.Ctx(), api.CreateApplicationRequestDto{
					Name:              relayAppName,
					DisplayName:       relayAppName,
					Type:              "confidential",
					RedirectUris:      []string{exchangeRedirectUri},
					TrustedExchangers: []string{exchangerAppName},
				})
				Expect(err).ToNot(HaveOccurred())
				Expect(relay.Secret).ToNot(BeNil())
				relaySecret = *relay.Secret

				_, err = h.Client().Project().Application(exchangeProjectSlug).Create(h.Ctx(), api.CreateApplicationRequestDto{
					Name:              targetAppName,
					DisplayName:       targetAppName,
					Type:              "public",
					RedirectUris:      []string{exchangeRedirectUri},
					TrustedExchangers: []string{exchangerAppName, publicAppName, relayAppName},
				})
				Expect(err).ToNot(HaveOccurred())

				_, err = h.Client().Project().Application(exchangeProjectSlug).Create(h.Ctx(), api.CreateApplicationRequestDto{
					Name:         untrustingAppName,
					DisplayName:  untrustingAppName,
					Type:         "public",
					RedirectUris: []string{exchangeRedirectUri},
				})
				Expect(err).ToNot(HaveOccurred())

				createExchangeUser(h.Scope())
			})

			exchangeRequest := func(subjectToken string) url.Values {
				return exchangeRequestFor(exchangerAppName, exchangerSecret, subjectToken, targetAppName)
			}

			Describe("refuses an exchange", func() {
				cases := map[string]struct {
					mutate func(form url.Values)
					error  string
				}{
					"without client authentication": {
						func(form url.Values) { form.Del("client_secret") }, "invalid_client",
					},
					"without any client identification": {
						func(form url.Values) { form.Del("client_id"); form.Del("client_secret") }, "invalid_client",
					},
					"with a wrong client secret": {
						func(form url.Values) { form.Set("client_secret", "not-the-secret") }, "invalid_client",
					},
					"without a subject token": {
						func(form url.Values) { form.Del("subject_token") }, "invalid_request",
					},
					"with a subject token type other than access token": {
						func(form url.Values) { form.Set("subject_token_type", "urn:ietf:params:oauth:token-type:jwt") }, "invalid_request",
					},
					"without an audience": {
						func(form url.Values) { form.Del("audience") }, "invalid_request",
					},
					"for an unknown audience": {
						func(form url.Values) { form.Set("audience", "no-such-app") }, "invalid_target",
					},
					"for an audience that does not trust the exchanger": {
						func(form url.Values) { form.Set("audience", untrustingAppName) }, "invalid_target",
					},
					"with a garbage subject token": {
						func(form url.Values) { form.Set("subject_token", "not.a.jwt") }, "invalid_request",
					},
				}

				for name, c := range cases {
					It(name, func() {
						form := exchangeRequest(loginExchangeUser(h, exchangerAppName, exchangerSecret, "openid"))
						c.mutate(form)

						status, body := exchangeToken(h, form)

						Expect(status).To(Equal(http.StatusBadRequest))
						Expect(body["error"]).To(Equal(c.error))
					})
				}

				It("by a public client the audience trusts", func() {
					form := exchangeRequest(loginExchangeUser(h, publicAppName, "", "openid"))
					form.Set("client_id", publicAppName)
					form.Del("client_secret")

					status, body := exchangeToken(h, form)

					Expect(status).To(Equal(http.StatusBadRequest))
					Expect(body["error"]).To(Equal("invalid_client"))
				})

				It("of a subject token that was issued to another application", func() {
					form := exchangeRequest(loginExchangeUser(h, publicAppName, "", "openid"))

					status, body := exchangeToken(h, form)

					Expect(status).To(Equal(http.StatusBadRequest))
					Expect(body["error"]).To(Equal("invalid_request"))
				})

				It("of an expired subject token", func() {
					form := exchangeRequest(loginExchangeUser(h, exchangerAppName, exchangerSecret, "openid"))
					h.SetTime(time.Now().Add(2 * time.Hour))
					defer h.SetTime(time.Now())

					status, body := exchangeToken(h, form)

					Expect(status).To(Equal(http.StatusBadRequest))
					Expect(body["error"]).To(Equal("invalid_request"))
				})

				It("of an id token presented as the subject token", func() {
					form := exchangeRequest(loginExchangeUserTokens(h, exchangerAppName, exchangerSecret, "openid")["id_token"].(string))

					status, body := exchangeToken(h, form)

					Expect(status).To(Equal(http.StatusBadRequest))
					Expect(body["error"]).To(Equal("invalid_request"))
				})
			})

			AfterAll(func() {
				if h != nil {
					h.Close()
				}
			})

			It("exchanges a user's token for one addressed to a target that trusts the exchanger", func() {
				// arrange
				subjectToken := loginExchangeUser(h, exchangerAppName, exchangerSecret, "openid profile")
				subjectClaims := accessTokenClaims(subjectToken)

				// act
				status, body := exchangeToken(h, url.Values{
					"grant_type":         {"urn:ietf:params:oauth:grant-type:token-exchange"},
					"client_id":          {exchangerAppName},
					"client_secret":      {exchangerSecret},
					"subject_token":      {subjectToken},
					"subject_token_type": {"urn:ietf:params:oauth:token-type:access_token"},
					"audience":           {targetAppName},
				})

				// assert
				Expect(status).To(Equal(http.StatusOK))
				Expect(body["issued_token_type"]).To(Equal("urn:ietf:params:oauth:token-type:access_token"))
				Expect(body["token_type"]).To(Equal("Bearer"))

				claims := accessTokenClaims(body["access_token"].(string))
				Expect(claims["sub"]).To(Equal(subjectClaims["sub"]))
				Expect(claims["aud"]).To(ConsistOf(targetAppName))
				Expect(claims["scopes"]).To(Equal(subjectClaims["scopes"]))
				Expect(claims["act"]).To(Equal(map[string]any{"sub": exchangerAppName}))
			})

			It("caps the exchanged token at the subject token's expiry", func() {
				// arrange
				subjectToken := loginExchangeUser(h, exchangerAppName, exchangerSecret, "openid")
				subjectExpiry := accessTokenClaims(subjectToken)["exp"].(float64)
				h.SetTime(time.Now().Add(30 * time.Minute))
				defer h.SetTime(time.Now())

				// act
				status, body := exchangeToken(h, exchangeRequest(subjectToken))

				// assert
				Expect(status).To(Equal(http.StatusOK))
				Expect(accessTokenClaims(body["access_token"].(string))["exp"]).To(Equal(subjectExpiry))
				Expect(body["expires_in"]).To(BeNumerically("<=", 30*60))
			})

			It("nests the earlier actor when an exchanged token is exchanged again", func() {
				// arrange
				status, first := exchangeToken(h, exchangeRequestFor(exchangerAppName, exchangerSecret, loginExchangeUser(h, exchangerAppName, exchangerSecret, "openid"), relayAppName))
				Expect(status).To(Equal(http.StatusOK))

				// act
				status, second := exchangeToken(h, exchangeRequestFor(relayAppName, relaySecret, first["access_token"].(string), targetAppName))

				// assert
				Expect(status).To(Equal(http.StatusOK))
				claims := accessTokenClaims(second["access_token"].(string))
				Expect(claims["aud"]).To(ConsistOf(targetAppName))
				Expect(claims["act"]).To(Equal(map[string]any{
					"sub": relayAppName,
					"act": map[string]any{"sub": exchangerAppName},
				}))
			})
		})

		Describe("Trusted exchangers from initial configuration ["+backend.name+"]", Ordered, func() {
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

			It("applies the trusted exchangers declared for an application in the initial virtual server", func() {
				scope := h.Scope().NewScope()
				defer utils.PanicOnError(scope.Close, "closing scope")
				ctx := middlewares.ContextWithScope(context.Background(), scope)
				ctx = authentication.ContextWithCurrentUser(ctx, authentication.SystemUser())
				m := ioc.GetDependency[mediatr.Mediator](scope)
				dbContext := ioc.GetDependency[database.Context](scope)

				const vsName = "token-exchange-vs"
				_, err := mediatr.Send[*commands.CreateVirtualServerResponse](ctx, m, commands.CreateVirtualServer{
					Name:                    vsName,
					DisplayName:             "Token Exchange VS",
					PrimarySigningAlgorithm: config.SigningAlgorithmEdDSA,
					Projects: []commands.CreateVirtualServerProject{{
						Slug: exchangeProjectSlug,
						Name: "Token Exchange",
						Applications: []commands.CreateVirtualServerProjectApplication{{
							Name:              targetAppName,
							DisplayName:       "Cluster A",
							Type:              "public",
							RedirectUris:      []string{exchangeRedirectUri},
							TrustedExchangers: []string{exchangerAppName},
						}},
					}},
				})
				Expect(err).ToNot(HaveOccurred())
				Expect(dbContext.SaveChanges(ctx)).To(Succeed())

				applications, _, err := dbContext.Applications().List(ctx, repositories.NewApplicationFilter().Name(targetAppName))
				Expect(err).ToNot(HaveOccurred())
				Expect(applications).To(HaveLen(1))

				application, err := mediatr.Send[*queries.GetApplicationResult](ctx, m, queries.GetApplication{
					VirtualServerName: vsName,
					ProjectSlug:       exchangeProjectSlug,
					ApplicationId:     applications[0].Id(),
				})
				Expect(err).ToNot(HaveOccurred())
				Expect(application.TrustedExchangers).To(ConsistOf(exchangerAppName))
			})
		})
	}
}

func loginExchangeUser(h *harness, appName string, appSecret string, scope string) string {
	return loginExchangeUserTokens(h, appName, appSecret, scope)["access_token"].(string)
}

func loginExchangeUserTokens(h *harness, appName string, appSecret string, scope string) map[string]any {
	deviceForm := url.Values{
		"client_id": {appName},
		"scope":     {scope},
	}
	if appSecret != "" {
		deviceForm.Set("client_secret", appSecret)
	}
	status, device := postOidcForm(h, "/device", deviceForm)
	Expect(status).To(Equal(http.StatusOK))

	loginToken, err := h.Client().Oidc().PostActivate(h.Ctx(), device["user_code"].(string))
	Expect(err).ToNot(HaveOccurred())

	Expect(h.Client().Oidc().VerifyPassword(h.Ctx(), loginToken, exchangeUserUsername, exchangeUserPassword)).To(Succeed())
	Expect(h.Client().Oidc().FinishLogin(h.Ctx(), loginToken)).To(Succeed())

	tokenForm := url.Values{
		"grant_type":  {"urn:ietf:params:oauth:grant-type:device_code"},
		"device_code": {device["device_code"].(string)},
		"client_id":   {appName},
	}
	if appSecret != "" {
		tokenForm.Set("client_secret", appSecret)
	}
	status, tokens := postOidcForm(h, "/token", tokenForm)
	Expect(status).To(Equal(http.StatusOK))
	return tokens
}

func exchangeRequestFor(clientId string, clientSecret string, subjectToken string, audience string) url.Values {
	return url.Values{
		"grant_type":         {"urn:ietf:params:oauth:grant-type:token-exchange"},
		"client_id":          {clientId},
		"client_secret":      {clientSecret},
		"subject_token":      {subjectToken},
		"subject_token_type": {"urn:ietf:params:oauth:token-type:access_token"},
		"audience":           {audience},
	}
}

func exchangeToken(h *harness, form url.Values) (int, map[string]any) {
	return postOidcForm(h, "/token", form)
}

func postOidcForm(h *harness, endpoint string, form url.Values) (int, map[string]any) {
	resp, err := http.PostForm(fmt.Sprintf("%s/oidc/%s%s", h.ApiUrl(), h.VirtualServer(), endpoint), form)
	Expect(err).ToNot(HaveOccurred())
	defer resp.Body.Close() //nolint:errcheck

	var body map[string]any
	Expect(json.NewDecoder(resp.Body).Decode(&body)).To(Succeed())
	return resp.StatusCode, body
}

func createExchangeUser(scope *ioc.DependencyProvider) {
	subscope := scope.NewScope()
	defer utils.PanicOnError(subscope.Close, "closing scope")

	ctx := middlewares.ContextWithScope(context.Background(), subscope)
	ctx = authentication.ContextWithCurrentUser(ctx, authentication.SystemUser())

	m := ioc.GetDependency[mediatr.Mediator](subscope)
	dbContext := ioc.GetDependency[database.Context](subscope)

	userResp, err := mediatr.Send[*commands.CreateUserResponse](ctx, m, commands.CreateUser{
		VirtualServerName: "test-vs",
		DisplayName:       "Exchange User",
		Username:          exchangeUserUsername,
		Email:             "exchange-user@test.local",
		EmailVerified:     true,
	})
	Expect(err).ToNot(HaveOccurred())
	Expect(dbContext.SaveChanges(ctx)).To(Succeed())

	dbContext.Credentials().Insert(repositories.NewCredential(userResp.Id, &repositories.CredentialPasswordDetails{
		HashedPassword: utils.HashPassword(exchangeUserPassword),
		Temporary:      false,
	}))
	Expect(dbContext.SaveChanges(ctx)).To(Succeed())
}
