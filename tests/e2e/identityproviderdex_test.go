//go:build e2e

package e2e

import (
	"html"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"regexp"
	"strings"

	"github.com/The127/Keyline/api"
	"github.com/The127/Keyline/config"

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

			BeforeAll(func() {
				if backend.dbMode == config.DatabaseModePostgres && !postgresBackendAvailable() {
					Skip("Postgres not available")
				}
				if !dexAvailable() {
					Skip("dex not available")
				}
				h = newE2eTestHarness(backend.dbMode, serviceUserTokenSource, withPort(dexHarnessPort))

				_, err := h.Client().VirtualServer().IdentityProviders().Create(h.Ctx(), api.CreateIdentityProviderRequestDto{
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
			})

			AfterAll(func() {
				if h != nil {
					h.Close()
				}
			})

			It("comes back from dex with a code and the state of the start", func() {
				status, body := startIdentityProviderLogin(h, beginLogin(h), "dex")
				Expect(status).To(Equal(http.StatusOK))

				callback := loginAtDex(body["authorizationUrl"].(string))

				Expect(callback.Scheme + "://" + callback.Host + callback.Path).To(Equal("http://localhost:25999/oidc/test-vs/identity-providers/dex/callback"))
				Expect(callback.Query().Get("state")).To(Equal(stateOf(body)))
				Expect(callback.Query().Get("code")).ToNot(BeEmpty())
			})
		})
	}
}

func readAll(resp *http.Response) string {
	body, err := io.ReadAll(resp.Body)
	Expect(err).ToNot(HaveOccurred())
	return string(body)
}

func loginAtDex(authorizationUrl string) *url.URL {
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
		"login":    {dexUserEmail},
		"password": {dexUserPassword},
	})
	Expect(err).ToNot(HaveOccurred())
	defer result.Body.Close() //nolint:errcheck
	Expect(result.StatusCode).To(Equal(http.StatusSeeOther), "dex did not redirect back: %s", readAll(result))

	callback, err := url.Parse(result.Header.Get("Location"))
	Expect(err).ToNot(HaveOccurred())
	return callback
}
