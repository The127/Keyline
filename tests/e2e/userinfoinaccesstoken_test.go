//go:build e2e

package e2e

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/The127/Keyline/api"
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
	userinfoProjectSlug  = "userinfo-claims"
	userinfoRedirectUri  = "http://localhost:9200/callback"
	userinfoAppWith      = "userinfo-app"
	userinfoAppWithout   = "plain-app"
	userinfoUserUsername = "userinfo-user"
	userinfoUserPassword = "userinfo-password-123!"
	userinfoUserName     = "Userinfo User"
	userinfoUserEmail    = "userinfo-user@test.local"
)

var userinfoClaimNames = []string{"preferred_username", "name", "email", "email_verified"}

func init() {
	for _, backend := range testBackends {
		backend := backend
		Describe("Userinfo claims in access tokens ["+backend.name+"]", Ordered, func() {
			var h *harness

			createApplication := func(name string, userinfoInAccessToken bool) api.CreateApplicationResponseDto {
				created, err := h.Client().Project().Application(userinfoProjectSlug).Create(h.Ctx(), api.CreateApplicationRequestDto{
					Name:                  name,
					DisplayName:           name,
					Type:                  "public",
					RedirectUris:          []string{userinfoRedirectUri},
					DeviceFlowEnabled:     true,
					UserinfoInAccessToken: userinfoInAccessToken,
				})
				Expect(err).ToNot(HaveOccurred())
				return created
			}

			getApplication := func(id api.CreateApplicationResponseDto) api.GetApplicationResponseDto {
				application, err := h.Client().Project().Application(userinfoProjectSlug).Get(h.Ctx(), id.Id)
				Expect(err).ToNot(HaveOccurred())
				return application
			}

			runDeviceFlowAs := func(appName string, scope string) api.DeviceTokenResponse {
				deviceResp, err := h.Client().Oidc().BeginDeviceFlow(h.Ctx(), appName, scope)
				Expect(err).ToNot(HaveOccurred())

				loginToken, err := h.Client().Oidc().PostActivate(h.Ctx(), deviceResp.UserCode)
				Expect(err).ToNot(HaveOccurred())

				Expect(h.Client().Oidc().VerifyPassword(h.Ctx(), loginToken, userinfoUserUsername, userinfoUserPassword)).To(Succeed())
				Expect(h.Client().Oidc().FinishLogin(h.Ctx(), loginToken)).To(Succeed())

				tokenResp, err := h.Client().Oidc().PollDeviceToken(h.Ctx(), appName, deviceResp.DeviceCode)
				Expect(err).ToNot(HaveOccurred())
				return tokenResp
			}

			BeforeAll(func() {
				if backend.dbMode == config.DatabaseModePostgres && !postgresBackendAvailable() {
					Skip("Postgres not available")
				}
				h = newE2eTestHarness(backend.dbMode, serviceUserTokenSource)

				_, err := h.Client().Project().Create(h.Ctx(), api.CreateProjectRequestDto{
					Slug: userinfoProjectSlug,
					Name: "Userinfo Claims",
				})
				Expect(err).ToNot(HaveOccurred())

				createApplication(userinfoAppWith, true)
				createApplication(userinfoAppWithout, false)
				createUserinfoUser(h.Scope())
			})

			AfterAll(func() {
				if h != nil {
					h.Close()
				}
			})

			It("defaults to keeping userinfo out of the access token", func() {
				created := createApplication("default-app", false)
				Expect(getApplication(created).UserinfoInAccessToken).To(BeFalse())
			})

			It("stores the setting on creation and changes it through patch", func() {
				created := createApplication("patched-app", true)
				Expect(getApplication(created).UserinfoInAccessToken).To(BeTrue())

				Expect(h.Client().Project().Application(userinfoProjectSlug).Patch(h.Ctx(), created.Id, api.PatchApplicationRequestDto{
					UserinfoInAccessToken: utils.Ptr(false),
				})).To(Succeed())
				Expect(getApplication(created).UserinfoInAccessToken).To(BeFalse())

				Expect(h.Client().Project().Application(userinfoProjectSlug).Patch(h.Ctx(), created.Id, api.PatchApplicationRequestDto{
					DisplayName: utils.Ptr("Patched"),
				})).To(Succeed())
				Expect(getApplication(created).UserinfoInAccessToken).To(BeFalse())
			})

			It("puts the userinfo claims for the granted scopes into the access token", func() {
				tokenResp := runDeviceFlowAs(userinfoAppWith, "openid profile email")

				claims := accessTokenClaims(tokenResp.AccessToken)
				Expect(claims["preferred_username"]).To(Equal(userinfoUserUsername))
				Expect(claims["name"]).To(Equal(userinfoUserName))
				Expect(claims["email"]).To(Equal(userinfoUserEmail))
				Expect(claims["email_verified"]).To(Equal(true))
			})

			It("limits the userinfo claims to the granted scopes", func() {
				tokenResp := runDeviceFlowAs(userinfoAppWith, "openid email")

				claims := accessTokenClaims(tokenResp.AccessToken)
				Expect(claims["email"]).To(Equal(userinfoUserEmail))
				Expect(claims).ToNot(HaveKey("preferred_username"))
				Expect(claims).ToNot(HaveKey("name"))
			})

			It("keeps the userinfo claims out of the access token when the setting is off", func() {
				tokenResp := runDeviceFlowAs(userinfoAppWithout, "openid profile email")

				claims := accessTokenClaims(tokenResp.AccessToken)
				for _, name := range userinfoClaimNames {
					Expect(claims).ToNot(HaveKey(name))
				}
			})

			It("serves the same claims from the userinfo endpoint either way", func() {
				for _, appName := range []string{userinfoAppWith, userinfoAppWithout} {
					tokenResp := runDeviceFlowAs(appName, "openid profile email")

					userinfo := fetchUserinfo(h, tokenResp.AccessToken)
					Expect(userinfo["preferred_username"]).To(Equal(userinfoUserUsername))
					Expect(userinfo["name"]).To(Equal(userinfoUserName))
					Expect(userinfo["email"]).To(Equal(userinfoUserEmail))
					Expect(userinfo["email_verified"]).To(Equal(true))
					Expect(userinfo["sub"]).ToNot(BeEmpty())
				}
			})
		})

		Describe("Userinfo claims in access tokens from initial configuration ["+backend.name+"]", Ordered, func() {
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

			It("applies the setting declared for an application in the initial virtual server", func() {
				scope := h.Scope().NewScope()
				defer utils.PanicOnError(scope.Close, "closing scope")
				ctx := middlewares.ContextWithScope(context.Background(), scope)
				ctx = authentication.ContextWithCurrentUser(ctx, authentication.SystemUser())
				m := ioc.GetDependency[mediatr.Mediator](scope)
				dbContext := ioc.GetDependency[database.Context](scope)

				const vsName = "userinfo-claims-vs"
				_, err := mediatr.Send[*commands.CreateVirtualServerResponse](ctx, m, commands.CreateVirtualServer{
					Name:                    vsName,
					DisplayName:             "Userinfo Claims VS",
					PrimarySigningAlgorithm: config.SigningAlgorithmEdDSA,
					Projects: []commands.CreateVirtualServerProject{{
						Slug: userinfoProjectSlug,
						Name: "Userinfo Claims",
						Applications: []commands.CreateVirtualServerProjectApplication{{
							Name:                  userinfoAppWith,
							DisplayName:           "Userinfo App",
							Type:                  "public",
							RedirectUris:          []string{userinfoRedirectUri},
							UserinfoInAccessToken: true,
						}},
					}},
				})
				Expect(err).ToNot(HaveOccurred())
				Expect(dbContext.SaveChanges(ctx)).To(Succeed())

				applications, _, err := dbContext.Applications().List(ctx, repositories.NewApplicationFilter().Name(userinfoAppWith))
				Expect(err).ToNot(HaveOccurred())
				Expect(applications).To(HaveLen(1))

				application, err := mediatr.Send[*queries.GetApplicationResult](ctx, m, queries.GetApplication{
					VirtualServerName: vsName,
					ProjectSlug:       userinfoProjectSlug,
					ApplicationId:     applications[0].Id(),
				})
				Expect(err).ToNot(HaveOccurred())
				Expect(application.UserinfoInAccessToken).To(BeTrue())
			})
		})
	}
}

func createUserinfoUser(scope *ioc.DependencyProvider) {
	subscope := scope.NewScope()
	defer utils.PanicOnError(subscope.Close, "closing scope")

	ctx := middlewares.ContextWithScope(context.Background(), subscope)
	ctx = authentication.ContextWithCurrentUser(ctx, authentication.SystemUser())

	m := ioc.GetDependency[mediatr.Mediator](subscope)
	dbContext := ioc.GetDependency[database.Context](subscope)

	userResp, err := mediatr.Send[*commands.CreateUserResponse](ctx, m, commands.CreateUser{
		VirtualServerName: "test-vs",
		DisplayName:       userinfoUserName,
		Username:          userinfoUserUsername,
		Email:             userinfoUserEmail,
		EmailVerified:     true,
	})
	Expect(err).ToNot(HaveOccurred())
	Expect(dbContext.SaveChanges(ctx)).To(Succeed())

	dbContext.Credentials().Insert(repositories.NewCredential(userResp.Id, &repositories.CredentialPasswordDetails{
		HashedPassword: utils.HashPassword(userinfoUserPassword),
		Temporary:      false,
	}))
	Expect(dbContext.SaveChanges(ctx)).To(Succeed())
}

func fetchUserinfo(h *harness, accessToken string) map[string]any {
	req, err := http.NewRequest(http.MethodGet, h.ApiUrl()+"/oidc/test-vs/userinfo", nil)
	Expect(err).ToNot(HaveOccurred())
	req.Header.Set("Authorization", "Bearer "+accessToken)

	resp, err := http.DefaultClient.Do(req)
	Expect(err).ToNot(HaveOccurred())
	defer utils.PanicOnError(resp.Body.Close, "closing body")
	Expect(resp.StatusCode).To(Equal(http.StatusOK))

	var userinfo map[string]any
	Expect(json.NewDecoder(resp.Body).Decode(&userinfo)).To(Succeed())
	return userinfo
}
