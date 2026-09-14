//go:build e2e

package e2e

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/The127/Keyline/config"

	"github.com/golang-jwt/jwt/v5"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func init() {
	for _, backend := range testBackends {
		backend := backend
		Describe("Request object ["+backend.name+"]", Ordered, ContinueOnFailure, func() {
			var h *harness

			BeforeAll(func() {
				if backend.dbMode == config.DatabaseModePostgres && !postgresBackendAvailable() {
					Skip("Postgres not available")
				}
				h = newE2eTestHarness(backend.dbMode, nil)
				Expect(setupAuthCodeFixtures(h.Scope())).To(Succeed())
			})

			AfterAll(func() {
				if h != nil {
					h.Close()
				}
			})

			It("answers a request object with request_not_supported", func() {
				requestObject, err := jwt.NewWithClaims(jwt.SigningMethodNone, jwt.MapClaims{
					"scope": "openid",
				}).SignedString(jwt.UnsafeAllowNoneSignatureType)
				Expect(err).ToNot(HaveOccurred())

				status, location := authorizeWith(h, url.Values{"request": {requestObject}})

				Expect(status).To(Equal(http.StatusFound))
				Expect(location.Scheme + "://" + location.Host + location.Path).To(Equal(authCodeRedirect))
				Expect(location.Query().Get("error")).To(Equal("request_not_supported"))
				Expect(location.Query().Get("state")).To(Equal("request-object-state"))
			})

			It("answers a request that is not a JWT with request_not_supported", func() {
				status, location := authorizeWith(h, url.Values{"request": {"not.a.jwt"}})

				Expect(status).To(Equal(http.StatusFound))
				Expect(location.Query().Get("error")).To(Equal("request_not_supported"))
			})

			It("answers a request_uri with request_uri_not_supported", func() {
				status, location := authorizeWith(h, url.Values{"request_uri": {"https://client.example/request.jwt"}})

				Expect(status).To(Equal(http.StatusFound))
				Expect(location.Query().Get("error")).To(Equal("request_uri_not_supported"))
			})

			It("advertises neither request objects nor request_uri", func() {
				response, err := http.Get(fmt.Sprintf("%s/oidc/%s/.well-known/openid-configuration", h.ApiUrl(), h.VirtualServer()))
				Expect(err).ToNot(HaveOccurred())
				defer response.Body.Close() //nolint:errcheck

				var discovery map[string]any
				Expect(json.NewDecoder(response.Body).Decode(&discovery)).To(Succeed())

				Expect(discovery).To(HaveKeyWithValue("request_parameter_supported", false))
				Expect(discovery).To(HaveKeyWithValue("request_uri_parameter_supported", false))
				Expect(discovery).ToNot(HaveKey("request_object_signing_alg_values_supported"))
			})
		})
	}
}

func authorizeWith(h *harness, extra url.Values) (int, *url.URL) {
	httpClient := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	query := url.Values{}
	query.Set("response_type", "code")
	query.Set("client_id", authCodePublicAppName)
	query.Set("redirect_uri", authCodeRedirect)
	query.Set("scope", "openid")
	query.Set("state", "request-object-state")
	query.Set("code_challenge", authCodePkceChallenge(authCodePkceVerifier))
	query.Set("code_challenge_method", "S256")
	for key, values := range extra {
		query[key] = values
	}

	response, err := httpClient.Get(fmt.Sprintf("%s/oidc/%s/authorize?%s", h.ApiUrl(), h.VirtualServer(), query.Encode()))
	Expect(err).ToNot(HaveOccurred())
	_ = response.Body.Close()

	location, err := url.Parse(response.Header.Get("Location"))
	Expect(err).ToNot(HaveOccurred())

	return response.StatusCode, location
}
