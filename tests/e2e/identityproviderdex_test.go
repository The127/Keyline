//go:build e2e

package e2e

import (
	"context"
	"encoding/json"
	"fmt"
	"html"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"regexp"
	"strings"

	"github.com/The127/Keyline/api"
	"github.com/The127/Keyline/client"
	"github.com/The127/Keyline/config"
	"github.com/The127/Keyline/internal/authentication"
	"github.com/The127/Keyline/internal/commands"
	"github.com/The127/Keyline/internal/database"
	"github.com/The127/Keyline/internal/middlewares"
	"github.com/The127/Keyline/internal/repositories"
	"github.com/The127/Keyline/utils"
	"github.com/The127/ioc"
	"github.com/The127/mediatr"
	"github.com/google/uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

const (
	dexIssuer       = "http://localhost:5556/dex"
	dexClientId     = "keyline"
	dexClientSecret = "keyline-dex-secret"
	dexUserEmail    = "alice@example.com"
	dexUserPassword = "password"
	dexHarnessPort  = 25999
)

var dexFormAction = regexp.MustCompile(`<form[^>]*action="([^"]+)"`)

func dexAvailable() bool {
	resp, err := http.Get(dexIssuer + "/.well-known/openid-configuration")
	if err != nil {
		return false
	}

	defer resp.Body.Close() //nolint:errcheck
	return resp.StatusCode == http.StatusOK
}

func init() {
	for _, backend := range testBackends {
		backend := backend
		Describe("Identity provider login at dex ["+backend.name+"]", Ordered, func() {
			var h *harness
			var dexProviderId uuid.UUID

			BeforeAll(func() {
				if backend.dbMode == config.DatabaseModePostgres && !postgresBackendAvailable() {
					Skip("Postgres not available")
				}

				if !dexAvailable() {
					Skip("dex not available")
				}

				h = newE2eTestHarness(backend.dbMode, serviceUserTokenSource, withPort(dexHarnessPort))

				created, err := h.Client().VirtualServer().IdentityProviders().Create(h.Ctx(), api.CreateIdentityProviderRequestDto{
					Name:                  "dex",
					DisplayName:           "Dex",
					Issuer:                dexIssuer,
					AuthorizationEndpoint: dexIssuer + "/auth",
					TokenEndpoint:         dexIssuer + "/token",
					UserinfoEndpoint:      dexIssuer + "/userinfo",
					Scopes:                []string{"openid", "email", "profile"},
					ClientId:              dexClientId,
					ClientSecret:          dexClientSecret,
				})
				Expect(err).ToNot(HaveOccurred())
				dexProviderId = created.Id
			})

			AfterAll(func() {
				if h != nil {
					h.Close()
				}
			})

			It("comes back from dex with a code and the state of the start", func() {
				browser := newBrowser()
				status, body := startIdentityProviderLogin(browser, h, beginLogin(h), "dex")
				Expect(status).To(Equal(http.StatusFound))

				callback := loginAtDex(body["authorizationUrl"].(string))

				Expect(callback.Scheme + "://" + callback.Host + callback.Path).To(Equal("http://localhost:25999/oidc/test-vs/identity-providers/dex/callback"))
				Expect(callback.Query().Get("state")).To(Equal(stateOf(body)))
				Expect(callback.Query().Get("code")).ToNot(BeEmpty())
			})

			It("refuses a callback with an unknown state", func() {
				resp := callbackAtKeyline(newBrowser(), h, "dex", url.Values{"code": {"whatever"}, "state": {"no-such-state"}})

				Expect(resp.StatusCode).To(Equal(http.StatusBadRequest))
			})

			It("sends an unlinked subject back to the login with an error", func() {
				Expect(h.Client().VirtualServer().Patch(h.Ctx(), client.PatchVirtualServerInput{EnableRegistration: utils.Ptr(false)})).To(Succeed())
				loginToken := beginLogin(h)
				browser := newBrowser()
				_, body := startIdentityProviderLogin(browser, h, loginToken, "dex")
				callback := loginAtDex(body["authorizationUrl"].(string))

				resp := callbackAtKeyline(browser, h, "dex", callback.Query())

				expectLoginRedirect(resp, loginToken, "identity_provider")
				Expect(loginState(h, loginToken)["step"]).To(Equal("passwordVerification"))
			})

			It("logs a linked user in", func() {
				linkUserToDex(h, dexProviderId, "alice", dexSubjectOfAlice())
				loginToken := beginLogin(h)
				browser := newBrowser()
				_, body := startIdentityProviderLogin(browser, h, loginToken, "dex")
				callback := loginAtDex(body["authorizationUrl"].(string))

				resp := callbackAtKeyline(browser, h, "dex", callback.Query())

				expectLoginRedirect(resp, loginToken, "")
				Expect(loginState(h, loginToken)["step"]).To(Equal("finish"))
				Expect(h.Client().Oidc().FinishLogin(h.Ctx(), loginToken)).To(Succeed())
			})

			It("refuses a callback from another browser", func() {
				loginToken := beginLogin(h)
				browser := newBrowser()
				_, body := startIdentityProviderLogin(browser, h, loginToken, "dex")
				callback := loginAtDex(body["authorizationUrl"].(string))

				resp := callbackAtKeyline(newBrowser(), h, "dex", callback.Query())

				Expect(resp.StatusCode).To(Equal(http.StatusBadRequest))
				Expect(loginState(h, loginToken)["step"]).To(Equal("passwordVerification"))
			})

			It("refuses a reused state", func() {
				loginToken := beginLogin(h)
				browser := newBrowser()
				_, body := startIdentityProviderLogin(browser, h, loginToken, "dex")
				callback := loginAtDex(body["authorizationUrl"].(string))
				callbackAtKeyline(browser, h, "dex", callback.Query())

				resp := callbackAtKeyline(browser, h, "dex", callback.Query())

				Expect(resp.StatusCode).To(Equal(http.StatusBadRequest))
			})

			It("refuses a code dex does not know", func() {
				loginToken := beginLogin(h)
				browser := newBrowser()
				_, body := startIdentityProviderLogin(browser, h, loginToken, "dex")

				resp := callbackAtKeyline(browser, h, "dex", url.Values{"code": {"garbage"}, "state": {stateOf(body)}})

				expectLoginRedirect(resp, loginToken, "identity_provider")
			})

			It("refuses a callback under another provider's name", func() {
				_, err := h.Client().VirtualServer().IdentityProviders().Create(h.Ctx(), explicitIdentityProvider("other", "Other"))
				Expect(err).ToNot(HaveOccurred())
				loginToken := beginLogin(h)
				browser := newBrowser()
				_, body := startIdentityProviderLogin(browser, h, loginToken, "dex")
				callback := loginAtDex(body["authorizationUrl"].(string))

				resp := callbackAtKeyline(browser, h, "other", callback.Query())

				expectLoginRedirect(resp, loginToken, "identity_provider")
			})

			It("refuses an unknown subject while registration is off", func() {
				Expect(h.Client().VirtualServer().Patch(h.Ctx(), client.PatchVirtualServerInput{EnableRegistration: utils.Ptr(false)})).To(Succeed())
				loginToken := beginLogin(h)
				browser := newBrowser()
				_, body := startIdentityProviderLogin(browser, h, loginToken, "dex")
				callback := loginAtDexAs(body["authorizationUrl"].(string), "bob@example.com")

				resp := callbackAtKeyline(browser, h, "dex", callback.Query())

				expectLoginRedirect(resp, loginToken, "identity_provider")
				Expect(dexUsersNamed(h, "bob")).To(BeEmpty())
			})

			It("creates a user for an unknown subject when registration is on", func() {
				Expect(h.Client().VirtualServer().Patch(h.Ctx(), client.PatchVirtualServerInput{EnableRegistration: utils.Ptr(true)})).To(Succeed())
				loginToken := beginLogin(h)
				browser := newBrowser()
				_, body := startIdentityProviderLogin(browser, h, loginToken, "dex")
				callback := loginAtDexAs(body["authorizationUrl"].(string), "bob@example.com")

				resp := callbackAtKeyline(browser, h, "dex", callback.Query())

				expectLoginRedirect(resp, loginToken, "")
				Expect(loginState(h, loginToken)["step"]).To(Equal("finish"))
				Expect(h.Client().Oidc().FinishLogin(h.Ctx(), loginToken)).To(Succeed())
				users := dexUsersNamed(h, "bob")
				Expect(users).To(HaveLen(1))
				Expect(users[0].PrimaryEmail).To(Equal("bob@example.com"))
			})

			It("logs the created user in again instead of creating another", func() {
				loginToken := beginLogin(h)
				browser := newBrowser()
				_, body := startIdentityProviderLogin(browser, h, loginToken, "dex")
				callback := loginAtDexAs(body["authorizationUrl"].(string), "bob@example.com")

				resp := callbackAtKeyline(browser, h, "dex", callback.Query())

				expectLoginRedirect(resp, loginToken, "")
				Expect(loginState(h, loginToken)["step"]).To(Equal("finish"))
				Expect(dexUsersNamed(h, "bob")).To(HaveLen(1))
			})

			It("refuses a callback once the login moved past the password step", func() {
				createUserinfoUser(h.Scope())
				loginToken := beginLogin(h)
				browser := newBrowser()
				_, body := startIdentityProviderLogin(browser, h, loginToken, "dex")
				callback := loginAtDex(body["authorizationUrl"].(string))
				Expect(h.Client().Oidc().VerifyPassword(h.Ctx(), loginToken, userinfoUserUsername, userinfoUserPassword)).To(Succeed())

				resp := callbackAtKeyline(browser, h, "dex", callback.Query())

				expectLoginRedirect(resp, loginToken, "identity_provider")
			})
		})
	}
}

func callbackAtKeyline(browser *http.Client, h *harness, providerName string, query url.Values) *http.Response {
	resp, err := browser.Get(fmt.Sprintf("%s/oidc/%s/identity-providers/%s/callback?%s", h.ApiUrl(), h.VirtualServer(), providerName, query.Encode()))
	Expect(err).ToNot(HaveOccurred())
	defer resp.Body.Close() //nolint:errcheck
	return resp
}

func expectLoginRedirect(resp *http.Response, loginToken string, errorCode string) {
	Expect(resp.StatusCode).To(Equal(http.StatusFound))
	location, err := url.Parse(resp.Header.Get("Location"))
	Expect(err).ToNot(HaveOccurred())
	Expect(resp.Header.Get("Location")).To(HavePrefix(config.C.Frontend.ExternalUrl + "/login?"))
	Expect(location.Query().Get("token")).To(Equal(loginToken))
	Expect(location.Query().Get("error")).To(Equal(errorCode))
}

func dexSubjectOfAlice() string {
	query := url.Values{}
	query.Set("client_id", dexClientId)
	query.Set("redirect_uri", "http://localhost:25999/oidc/test-vs/identity-providers/dex/callback")
	query.Set("response_type", "code")
	query.Set("scope", "openid")
	query.Set("state", "fixture")
	callback := loginAtDex(dexIssuer + "/auth?" + query.Encode())

	resp, err := http.PostForm(dexIssuer+"/token", url.Values{
		"grant_type":    {"authorization_code"},
		"code":          {callback.Query().Get("code")},
		"redirect_uri":  {"http://localhost:25999/oidc/test-vs/identity-providers/dex/callback"},
		"client_id":     {dexClientId},
		"client_secret": {dexClientSecret},
	})
	Expect(err).ToNot(HaveOccurred())
	defer resp.Body.Close() //nolint:errcheck
	Expect(resp.StatusCode).To(Equal(http.StatusOK))

	var tokens struct {
		IdToken string `json:"id_token"`
	}

	Expect(json.NewDecoder(resp.Body).Decode(&tokens)).To(Succeed())
	return accessTokenClaims(tokens.IdToken)["sub"].(string)
}

func linkUserToDex(h *harness, identityProviderId uuid.UUID, username string, subject string) {
	subscope := h.Scope().NewScope()
	defer utils.PanicOnError(subscope.Close, "closing scope")

	ctx := middlewares.ContextWithScope(context.Background(), subscope)
	ctx = authentication.ContextWithCurrentUser(ctx, authentication.SystemUser())

	m := ioc.GetDependency[mediatr.Mediator](subscope)
	dbContext := ioc.GetDependency[database.Context](subscope)

	userResp, err := mediatr.Send[*commands.CreateUserResponse](ctx, m, commands.CreateUser{
		VirtualServerName: "test-vs",
		DisplayName:       "Alice",
		Username:          username,
		Email:             username + "@example.com",
		EmailVerified:     true,
	})
	Expect(err).ToNot(HaveOccurred())
	Expect(dbContext.SaveChanges(ctx)).To(Succeed())

	dbContext.Credentials().Insert(repositories.NewCredential(userResp.Id, &repositories.CredentialExternalIdentity{
		IdentityProviderId: identityProviderId,
		Subject:            subject,
	}))
	Expect(dbContext.SaveChanges(ctx)).To(Succeed())
}

func readAll(resp *http.Response) string {
	body, err := io.ReadAll(resp.Body)
	Expect(err).ToNot(HaveOccurred())
	return string(body)
}

func dexUsersNamed(h *harness, username string) []api.ListUsersResponseDto {
	page, err := h.Client().User().List(h.Ctx(), client.ListUserParams{Page: 1, Size: 100})
	Expect(err).ToNot(HaveOccurred())

	var users []api.ListUsersResponseDto
	for _, user := range page.Items {
		if user.Username == username {
			users = append(users, user)
		}
	}

	return users
}

func loginAtDex(authorizationUrl string) *url.URL {
	return loginAtDexAs(authorizationUrl, dexUserEmail)
}

func loginAtDexAs(authorizationUrl string, email string) *url.URL {
	jar, err := cookiejar.New(nil)
	Expect(err).ToNot(HaveOccurred())
	browser := &http.Client{
		Jar: jar,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if strings.HasPrefix(req.URL.String(), "http://localhost:25999/") {
				return http.ErrUseLastResponse
			}

			return nil
		},
	}

	page, err := browser.Get(authorizationUrl)
	Expect(err).ToNot(HaveOccurred())
	defer page.Body.Close() //nolint:errcheck
	Expect(page.StatusCode).To(Equal(http.StatusOK))

	match := dexFormAction.FindStringSubmatch(readAll(page))
	Expect(match).To(HaveLen(2), "no login form on the dex page")
	formAction, err := page.Request.URL.Parse(html.UnescapeString(match[1]))
	Expect(err).ToNot(HaveOccurred())

	result, err := browser.PostForm(formAction.String(), url.Values{
		"login":    {email},
		"password": {dexUserPassword},
	})
	Expect(err).ToNot(HaveOccurred())
	defer result.Body.Close() //nolint:errcheck
	Expect(result.StatusCode).To(Equal(http.StatusSeeOther), "dex did not redirect back: %s", readAll(result))

	callback, err := url.Parse(result.Header.Get("Location"))
	Expect(err).ToNot(HaveOccurred())
	return callback
}
