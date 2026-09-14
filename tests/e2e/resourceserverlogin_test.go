//go:build e2e

package e2e

import (
	"fmt"
	"net/http"
	"net/url"

	"github.com/The127/Keyline/client/api"
	"github.com/The127/Keyline/config"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

const (
	resourceServerLoginProjectSlug = "rs-login"
	resourceServerLoginAppName     = "rs-login-app"
	resourceServerLoginRedirect    = "http://localhost:9300/callback"
)

func init() {
	for _, backend := range testBackends {
		backend := backend
		Describe("Login asking for a resource server scope ["+backend.name+"]", Ordered, ContinueOnFailure, func() {
			var h *harness

			BeforeAll(func() {
				if backend.dbMode == config.DatabaseModePostgres && !postgresBackendAvailable() {
					Skip("Postgres not available")
				}
				h = newE2eTestHarness(backend.dbMode, serviceUserLogin)

				_, err := h.Client().Project().Create(h.Ctx(), api.CreateProjectRequestDto{
					Slug: resourceServerLoginProjectSlug,
					Name: "Resource Server Login",
				})
				Expect(err).ToNot(HaveOccurred())

				_, err = h.Client().Project().Application(resourceServerLoginProjectSlug).Create(h.Ctx(), api.CreateApplicationRequestDto{
					Name:         resourceServerLoginAppName,
					DisplayName:  "Resource Server Login App",
					Type:         "public",
					RedirectUris: []string{resourceServerLoginRedirect},
				})
				Expect(err).ToNot(HaveOccurred())

				_, _, err = createResourceServerWithScope(h, resourceServerScopeFixture{
					ProjectSlug:        resourceServerLoginProjectSlug,
					ResourceServerSlug: "clusters",
					Scope:              "k8s",
					Name:               "Kubernetes",
				})
				Expect(err).ToNot(HaveOccurred())
			})

			AfterAll(func() {
				if h != nil {
					h.Close()
				}
			})

			It("refuses a resource server scope", func() {
				location := authorizeWithScope(h, "openid "+resourceServerLoginProjectSlug+":k8s")

				Expect(location.Scheme + "://" + location.Host + location.Path).To(Equal(resourceServerLoginRedirect))
				Expect(location.Query().Get("error")).To(Equal("invalid_scope"))
				Expect(location.Query().Get("state")).To(Equal("resource-server-login-state"))
			})

			It("refuses a scope the project does not have", func() {
				location := authorizeWithScope(h, "openid "+resourceServerLoginProjectSlug+":grafana")

				Expect(location.Query().Get("error")).To(Equal("invalid_scope"))
			})

			It("refuses a scope of a project that does not exist", func() {
				location := authorizeWithScope(h, "openid no-such-project:k8s")

				Expect(location.Query().Get("error")).To(Equal("invalid_scope"))
			})

			It("lets plain scopes through to the login", func() {
				location := authorizeWithScope(h, "openid email profile")

				Expect(location.Query().Get("error")).To(BeEmpty())
				Expect(location.String()).ToNot(HavePrefix(resourceServerLoginRedirect))
			})
		})
	}
}

func authorizeWithScope(h *harness, scope string) *url.URL {
	httpClient := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	query := url.Values{}
	query.Set("response_type", "code")
	query.Set("client_id", resourceServerLoginAppName)
	query.Set("redirect_uri", resourceServerLoginRedirect)
	query.Set("scope", scope)
	query.Set("state", "resource-server-login-state")
	query.Set("code_challenge", authCodePkceChallenge(authCodePkceVerifier))
	query.Set("code_challenge_method", "S256")

	response, err := httpClient.Get(fmt.Sprintf("%s/oidc/%s/authorize?%s", h.ApiUrl(), h.VirtualServer(), query.Encode()))
	Expect(err).ToNot(HaveOccurred())
	_ = response.Body.Close()
	Expect(response.StatusCode).To(Equal(http.StatusFound))

	location, err := url.Parse(response.Header.Get("Location"))
	Expect(err).ToNot(HaveOccurred())

	return location
}
