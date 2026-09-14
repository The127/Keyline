//go:build e2e

package e2e

import (
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

			for _, claim := range []string{"response_type", "client_id", "redirect_uri", "scope", "state", "nonce", "response_mode"} {
				It("refuses a "+claim+" claim that is not a string", func() {
					status, _ := authorizeWithRequestObject(h, unsignedRequestObject(jwt.MapClaims{
						claim: 123,
					}))

					Expect(status).To(Equal(http.StatusBadRequest))
				})
			}

			It("refuses a request object that is not a JWT", func() {
				status, _ := authorizeWithRequestObject(h, "not.a.jwt")

				Expect(status).To(Equal(http.StatusBadRequest))
			})

			It("applies a string claim", func() {
				status, location := authorizeWithRequestObject(h, unsignedRequestObject(jwt.MapClaims{
					"response_type": "token",
				}))

				Expect(status).To(Equal(http.StatusFound))
				Expect(location.Query().Get("error")).To(Equal("unsupported_response_type"))
			})
		})
	}
}

func unsignedRequestObject(claims jwt.MapClaims) string {
	requestObject, err := jwt.NewWithClaims(jwt.SigningMethodNone, claims).SignedString(jwt.UnsafeAllowNoneSignatureType)
	Expect(err).ToNot(HaveOccurred())

	return requestObject
}

func authorizeWithRequestObject(h *harness, requestObject string) (int, *url.URL) {
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
	query.Set("code_challenge", authCodePkceChallenge(authCodePkceVerifier))
	query.Set("code_challenge_method", "S256")
	query.Set("request", requestObject)

	response, err := httpClient.Get(fmt.Sprintf("%s/oidc/%s/authorize?%s", h.ApiUrl(), h.VirtualServer(), query.Encode()))
	Expect(err).ToNot(HaveOccurred())
	_ = response.Body.Close()

	location, err := url.Parse(response.Header.Get("Location"))
	Expect(err).ToNot(HaveOccurred())

	return response.StatusCode, location
}
