//go:build e2e

package e2e

import (
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/The127/Keyline/config"
	"github.com/The127/Keyline/internal/commands"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

const wrongPrivateKey = "-----BEGIN PRIVATE KEY-----\nMFECAQEwBQYDK2VwBCIEIGlwiAvmqJTxz8n1Ewwwy7/XG2LJphqbOhfKcfg2l9YU\ngSEAi+MvQpVxlYdQrbbsn5xltPOYbU00qJtkEHPO2uzUmKQ=\n-----END PRIVATE KEY-----\n"

func init() {
	for _, backend := range testBackends {
		backend := backend
		Describe("Service user login ["+backend.name+"]", Ordered, func() {
			var h *harness
			var now time.Time

			BeforeAll(func() {
				if backend.dbMode == config.DatabaseModePostgres && !postgresBackendAvailable() {
					Skip("Postgres not available")
				}
				h = newE2eTestHarness(backend.dbMode, nil)
				now = time.Now().Truncate(time.Second)
				h.SetTime(now)
			})

			AfterAll(func() {
				if h != nil {
					h.Close()
				}
			})

			It("exchanges a signed assertion for an access token", func() {
				status, body := exchangeServiceUserToken(h, serviceUserAssertion(serviceUserPrivateKey, now, nil))

				Expect(status).To(Equal(http.StatusOK))
				Expect(body["access_token"]).ToNot(BeEmpty())
				Expect(body["token_type"]).To(Equal("Bearer"))
				Expect(body["expires_in"]).To(BeNumerically(">", 0))
			})

			It("refuses a replayed assertion and accepts a fresh one", func() {
				assertion := serviceUserAssertion(serviceUserPrivateKey, now, nil)

				status, _ := exchangeServiceUserToken(h, assertion)
				Expect(status).To(Equal(http.StatusOK))

				status, body := exchangeServiceUserToken(h, assertion)
				expectInvalidGrant(status, body)

				status, _ = exchangeServiceUserToken(h, serviceUserAssertion(serviceUserPrivateKey, now, nil))
				Expect(status).To(Equal(http.StatusOK))
			})

			It("accepts a replayed jti again once the original assertion has expired", func() {
				fixedJti := func(c jwt.MapClaims) { c["jti"] = "fixed-service-user-jti" }
				status, _ := exchangeServiceUserToken(h, serviceUserAssertion(serviceUserPrivateKey, now, fixedJti))
				Expect(status).To(Equal(http.StatusOK))

				later := now.Add(2 * time.Minute)
				h.SetTime(later)
				defer h.SetTime(now)

				status, _ = exchangeServiceUserToken(h, serviceUserAssertion(serviceUserPrivateKey, later, fixedJti))
				Expect(status).To(Equal(http.StatusOK))
			})

			Describe("refuses an assertion", func() {
				cases := map[string]func() string{
					"signed with the wrong key": func() string {
						return serviceUserAssertion(wrongPrivateKey, now, nil)
					},
					"with an unknown kid": func() string {
						return signServiceUserAssertion(serviceUserPrivateKey, "no-such-kid", serviceUserAssertionClaims(now))
					},
					"without a kid header": func() string {
						return signServiceUserAssertion(serviceUserPrivateKey, "", serviceUserAssertionClaims(now))
					},
					"whose iss differs from sub": func() string {
						return serviceUserAssertion(serviceUserPrivateKey, now, func(c jwt.MapClaims) { c["iss"] = "someone-else" })
					},
					"naming an unknown user": func() string {
						return serviceUserAssertion(serviceUserPrivateKey, now, func(c jwt.MapClaims) { c["iss"] = "nobody"; c["sub"] = "nobody" })
					},
					"naming a regular user": func() string {
						return serviceUserAssertion(serviceUserPrivateKey, now, func(c jwt.MapClaims) { c["iss"] = "admin"; c["sub"] = "admin" })
					},
					"addressed to an unknown application": func() string {
						return serviceUserAssertion(serviceUserPrivateKey, now, func(c jwt.MapClaims) { c["aud"] = "no-such-app" })
					},
					"without aud": func() string {
						return serviceUserAssertion(serviceUserPrivateKey, now, func(c jwt.MapClaims) { delete(c, "aud") })
					},
					"without the openid scope": func() string {
						return serviceUserAssertion(serviceUserPrivateKey, now, func(c jwt.MapClaims) { c["scopes"] = "profile" })
					},
					"without scopes": func() string {
						return serviceUserAssertion(serviceUserPrivateKey, now, func(c jwt.MapClaims) { delete(c, "scopes") })
					},
					"without exp": func() string {
						return serviceUserAssertion(serviceUserPrivateKey, now, func(c jwt.MapClaims) { delete(c, "exp") })
					},
					"that has expired": func() string {
						return serviceUserAssertion(serviceUserPrivateKey, now, func(c jwt.MapClaims) { c["exp"] = now.Add(-time.Second).Unix() })
					},
					"that lives longer than five minutes": func() string {
						return serviceUserAssertion(serviceUserPrivateKey, now, func(c jwt.MapClaims) { c["exp"] = now.Add(time.Hour).Unix() })
					},
					"without jti": func() string {
						return serviceUserAssertion(serviceUserPrivateKey, now, func(c jwt.MapClaims) { delete(c, "jti") })
					},
					"signed with hmac using the public key": func() string {
						token := jwt.NewWithClaims(jwt.SigningMethodHS256, serviceUserAssertionClaims(now))
						token.Header["kid"] = serviceUserKid
						signed, err := token.SignedString([]byte(serviceUserPublicKey))
						Expect(err).ToNot(HaveOccurred())
						return signed
					},
					"that is garbage": func() string {
						return "not.a.jwt"
					},
				}

				for name, assertion := range cases {
					It(name, func() {
						status, body := exchangeServiceUserToken(h, assertion())
						expectInvalidGrant(status, body)
					})
				}
			})
		})
	}
}

func serviceUserAssertionClaims(now time.Time) jwt.MapClaims {
	return jwt.MapClaims{
		"aud":    commands.AdminApplicationName,
		"iss":    serviceUserUsername,
		"sub":    serviceUserUsername,
		"scopes": "openid profile email",
		"iat":    now.Unix(),
		"exp":    now.Add(time.Minute).Unix(),
		"jti":    uuid.NewString(),
	}
}

func serviceUserAssertion(privateKeyPem string, now time.Time, mutate func(jwt.MapClaims)) string {
	claims := serviceUserAssertionClaims(now)
	if mutate != nil {
		mutate(claims)
	}
	return signServiceUserAssertion(privateKeyPem, serviceUserKid, claims)
}

func signServiceUserAssertion(privateKeyPem string, kid string, claims jwt.MapClaims) string {
	block, _ := pem.Decode([]byte(privateKeyPem))
	Expect(block).ToNot(BeNil())

	key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	Expect(err).ToNot(HaveOccurred())

	token := jwt.NewWithClaims(jwt.SigningMethodEdDSA, claims)
	if kid != "" {
		token.Header["kid"] = kid
	}
	signed, err := token.SignedString(key)
	Expect(err).ToNot(HaveOccurred())
	return signed
}

func exchangeServiceUserToken(h *harness, assertion string) (int, map[string]any) {
	resp, err := http.PostForm(fmt.Sprintf("%s/oidc/%s/token", h.ApiUrl(), h.VirtualServer()),
		url.Values{
			"grant_type":         {"urn:ietf:params:oauth:grant-type:token-exchange"},
			"subject_token":      {assertion},
			"subject_token_type": {"urn:ietf:params:oauth:token-type:access_token"},
		},
	)
	Expect(err).ToNot(HaveOccurred())
	defer resp.Body.Close() //nolint:errcheck

	var body map[string]any
	Expect(json.NewDecoder(resp.Body).Decode(&body)).To(Succeed())
	return resp.StatusCode, body
}

func expectInvalidGrant(status int, body map[string]any) {
	Expect(status).To(Equal(http.StatusBadRequest))
	Expect(body["error"]).To(Equal("invalid_grant"))
}
