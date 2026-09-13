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
			var issuer string

			BeforeAll(func() {
				if backend.dbMode == config.DatabaseModePostgres && !postgresBackendAvailable() {
					Skip("Postgres not available")
				}
				h = newE2eTestHarness(backend.dbMode, nil)
				now = time.Now().Truncate(time.Second)
				h.SetTime(now)
				issuer = fmt.Sprintf("%s/oidc/%s", h.ApiUrl(), h.VirtualServer())
			})

			AfterAll(func() {
				if h != nil {
					h.Close()
				}
			})

			It("logs in with a signed assertion and gets an access token", func() {
				status, body := loginServiceUser(h, serviceUserAssertion(serviceUserPrivateKey, issuer, now, nil), nil)

				Expect(status).To(Equal(http.StatusOK))
				Expect(body["access_token"]).ToNot(BeEmpty())
				Expect(body["token_type"]).To(Equal("Bearer"))
				Expect(body["expires_in"]).To(BeNumerically(">", 0))
			})

			It("refuses a replayed assertion and accepts a fresh one", func() {
				assertion := serviceUserAssertion(serviceUserPrivateKey, issuer, now, nil)

				status, _ := loginServiceUser(h, assertion, nil)
				Expect(status).To(Equal(http.StatusOK))

				status, body := loginServiceUser(h, assertion, nil)
				expectInvalidGrant(status, body)

				status, _ = loginServiceUser(h, serviceUserAssertion(serviceUserPrivateKey, issuer, now, nil), nil)
				Expect(status).To(Equal(http.StatusOK))
			})

			It("accepts a replayed jti again once the original assertion has expired", func() {
				fixedJti := func(c jwt.MapClaims) { c["jti"] = "fixed-service-user-jti" }
				status, _ := loginServiceUser(h, serviceUserAssertion(serviceUserPrivateKey, issuer, now, fixedJti), nil)
				Expect(status).To(Equal(http.StatusOK))

				later := now.Add(2 * time.Minute)
				h.SetTime(later)
				defer h.SetTime(now)

				status, _ = loginServiceUser(h, serviceUserAssertion(serviceUserPrivateKey, issuer, later, fixedJti), nil)
				Expect(status).To(Equal(http.StatusOK))
			})

			Describe("refuses an assertion", func() {
				cases := map[string]func() string{
					"signed with the wrong key": func() string {
						return serviceUserAssertion(wrongPrivateKey, issuer, now, nil)
					},
					"with an unknown kid": func() string {
						return signServiceUserAssertion(serviceUserPrivateKey, "no-such-kid", serviceUserAssertionClaims(issuer, now))
					},
					"without a kid header": func() string {
						return signServiceUserAssertion(serviceUserPrivateKey, "", serviceUserAssertionClaims(issuer, now))
					},
					"whose iss differs from sub": func() string {
						return serviceUserAssertion(serviceUserPrivateKey, issuer, now, func(c jwt.MapClaims) { c["iss"] = "someone-else" })
					},
					"naming an unknown user": func() string {
						return serviceUserAssertion(serviceUserPrivateKey, issuer, now, func(c jwt.MapClaims) { c["iss"] = "nobody"; c["sub"] = "nobody" })
					},
					"naming a regular user": func() string {
						return serviceUserAssertion(serviceUserPrivateKey, issuer, now, func(c jwt.MapClaims) { c["iss"] = "admin"; c["sub"] = "admin" })
					},
					"addressed to another server": func() string {
						return serviceUserAssertion(serviceUserPrivateKey, issuer, now, func(c jwt.MapClaims) { c["aud"] = "https://elsewhere.example/oidc/other" })
					},
					"addressed to an application instead of the server": func() string {
						return serviceUserAssertion(serviceUserPrivateKey, issuer, now, func(c jwt.MapClaims) { c["aud"] = commands.AdminApplicationName })
					},
					"without aud": func() string {
						return serviceUserAssertion(serviceUserPrivateKey, issuer, now, func(c jwt.MapClaims) { delete(c, "aud") })
					},
					"without exp": func() string {
						return serviceUserAssertion(serviceUserPrivateKey, issuer, now, func(c jwt.MapClaims) { delete(c, "exp") })
					},
					"that has expired": func() string {
						return serviceUserAssertion(serviceUserPrivateKey, issuer, now, func(c jwt.MapClaims) { c["exp"] = now.Add(-time.Second).Unix() })
					},
					"that lives longer than five minutes": func() string {
						return serviceUserAssertion(serviceUserPrivateKey, issuer, now, func(c jwt.MapClaims) { c["exp"] = now.Add(time.Hour).Unix() })
					},
					"without jti": func() string {
						return serviceUserAssertion(serviceUserPrivateKey, issuer, now, func(c jwt.MapClaims) { delete(c, "jti") })
					},
					"signed with hmac using the public key": func() string {
						token := jwt.NewWithClaims(jwt.SigningMethodHS256, serviceUserAssertionClaims(issuer, now))
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
						status, body := loginServiceUser(h, assertion(), nil)
						expectInvalidGrant(status, body)
					})
				}
			})

			It("is refused at the token exchange grant", func() {
				status, body := loginServiceUser(h, serviceUserAssertion(serviceUserPrivateKey, issuer, now, nil), func(form url.Values) {
					form.Set("grant_type", "urn:ietf:params:oauth:grant-type:token-exchange")
					form.Set("subject_token", form.Get("assertion"))
					form.Set("subject_token_type", "urn:ietf:params:oauth:token-type:jwt")
				})

				Expect(status).To(Equal(http.StatusBadRequest))
				Expect(body["error"]).To(Equal("invalid_request"))
			})

			Describe("refuses a request", func() {
				cases := map[string]func(url.Values){
					"for an unknown application": func(form url.Values) {
						form.Set("client_id", "no-such-app")
					},
					"without client_id": func(form url.Values) {
						form.Del("client_id")
					},
					"without the openid scope": func(form url.Values) {
						form.Set("scope", "profile")
					},
					"without scope": func(form url.Values) {
						form.Del("scope")
					},
				}

				for name, mutate := range cases {
					It(name, func() {
						status, body := loginServiceUser(h, serviceUserAssertion(serviceUserPrivateKey, issuer, now, nil), mutate)
						expectInvalidGrant(status, body)
					})
				}
			})
		})
	}
}

func serviceUserAssertionClaims(issuer string, now time.Time) jwt.MapClaims {
	return jwt.MapClaims{
		"aud": issuer,
		"iss": serviceUserUsername,
		"sub": serviceUserUsername,
		"iat": now.Unix(),
		"exp": now.Add(time.Minute).Unix(),
		"jti": uuid.NewString(),
	}
}

func serviceUserAssertion(privateKeyPem string, issuer string, now time.Time, mutate func(jwt.MapClaims)) string {
	claims := serviceUserAssertionClaims(issuer, now)
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

func loginServiceUser(h *harness, assertion string, mutate func(url.Values)) (int, map[string]any) {
	form := url.Values{
		"grant_type": {"urn:ietf:params:oauth:grant-type:jwt-bearer"},
		"assertion":  {assertion},
		"client_id":  {commands.AdminApplicationName},
		"scope":      {"openid profile email"},
	}
	if mutate != nil {
		mutate(form)
	}

	resp, err := http.PostForm(fmt.Sprintf("%s/oidc/%s/token", h.ApiUrl(), h.VirtualServer()), form)
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
